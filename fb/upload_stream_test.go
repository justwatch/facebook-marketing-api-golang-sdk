package fb

import (
	"context"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/go-kit/log"
)

func TestUploadSourceBorrowsSeekableInputFromItsCurrentOffset(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "upload")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString("hello world"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(6, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	src, err := newUploadSource(f)
	if err != nil {
		t.Fatal(err)
	}
	if src.tmp != nil || src.size != 5 {
		t.Fatalf("seekable input was spooled or mis-sized: tmp=%v size=%d", src.tmp != nil, src.size)
	}
	for i := 0; i < 2; i++ {
		got, err := io.ReadAll(src.reader())
		if err != nil || string(got) != "world" {
			t.Fatalf("replay %d = %q, %v", i, got, err)
		}
	}
	if err := src.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.Name()); err != nil {
		t.Fatalf("borrowed file must survive close: %v", err)
	}
}

func TestUploadSourceSpoolsNonSeekableInputAndCleansUp(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	src, err := newUploadSource(struct{ io.Reader }{strings.NewReader("streamed")})
	if err != nil {
		t.Fatal(err)
	}
	if src.tmp == nil {
		t.Fatal("non-seekable input must be spooled")
	}
	name := src.tmp.Name()
	got, err := io.ReadAll(src.reader())
	if err != nil || string(got) != "streamed" {
		t.Fatalf("spooled read = %q, %v", got, err)
	}
	if err := src.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Fatalf("spool file remains after close: %v", err)
	}
}

func TestMultipartRequestReplaysWithExactLength(t *testing.T) {
	src, err := newUploadSource(strings.NewReader("image bytes"))
	if err != nil {
		t.Fatal(err)
	}
	req, err := newMultipartRequest(context.Background(), "https://graph.facebook.com/v24.0/act_1/adimages", "video_file_chunk", "logo.png", map[string]string{"extra": "field"}, src)
	if err != nil {
		t.Fatal(err)
	}
	if req.GetBody == nil {
		t.Fatal("GetBody must be set so the retry transport can replay the body")
	}
	for i := 0; i < 2; i++ {
		body, err := req.GetBody()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(body)
		if err != nil {
			t.Fatal(err)
		}
		if int64(len(raw)) != req.ContentLength {
			t.Fatalf("replay %d: body is %d bytes, Content-Length says %d", i, len(raw), req.ContentLength)
		}
		parsed := &http.Request{Method: http.MethodPost, Header: req.Header.Clone(), Body: io.NopCloser(strings.NewReader(string(raw)))}
		if err := parsed.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("replay %d: %v", i, err)
		}
		if parsed.FormValue("extra") != "field" {
			t.Fatalf("replay %d: extra field = %q", i, parsed.FormValue("extra"))
		}
		fh := parsed.MultipartForm.File["video_file_chunk"]
		if len(fh) != 1 || fh[0].Filename != "logo.png" {
			t.Fatalf("replay %d: file part = %+v", i, fh)
		}
		file, _ := fh[0].Open()
		content, _ := io.ReadAll(file)
		if string(content) != "image bytes" {
			t.Fatalf("replay %d: file content = %q", i, content)
		}
	}
}

func TestUploadFileStreamsMultipartThroughClient(t *testing.T) {
	var gotFilename, gotContent string
	var gotReplayable bool
	fake := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotReplayable = req.GetBody != nil && req.ContentLength > 0
		if err := req.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		fh := req.MultipartForm.File["video_file_chunk"][0]
		gotFilename = fh.Filename
		file, _ := fh.Open()
		content, _ := io.ReadAll(file)
		gotContent = string(content)

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"images":{}}`)), Header: make(http.Header)}, nil
	})
	c := &Client{l: log.NewNopLogger(), Client: &http.Client{Transport: fake}}

	var res struct{}
	if err := c.UploadFile(context.Background(), "https://graph.facebook.com/v24.0/act_1/adimages", "logo.png", strings.NewReader("image bytes"), nil, &res); err != nil {
		t.Fatal(err)
	}
	if !gotReplayable || gotFilename != "logo.png" || gotContent != "image bytes" {
		t.Fatalf("replayable=%v filename=%q content=%q", gotReplayable, gotFilename, gotContent)
	}
}

func TestUploadFileUsesBoundedMemoryForLargeFiles(t *testing.T) {
	// A sparse file exercises the real streaming path without a 500 MiB fixture.
	f, err := os.CreateTemp(t.TempDir(), "large")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	const size = 500 << 20
	if err := f.Truncate(size); err != nil {
		t.Fatal(err)
	}
	fake := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		n, err := io.Copy(io.Discard, req.Body)
		if err != nil || n != req.ContentLength || n < size {
			t.Fatalf("server saw %d bytes: %v", n, err)
		}

		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})
	c := &Client{l: log.NewNopLogger(), Client: &http.Client{Transport: fake}}

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	var res struct{}
	if err := c.UploadFile(context.Background(), "https://graph.facebook.com/v24.0/act_1/advideos", "large.mp4", f, nil, &res); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 8<<20 {
		t.Fatalf("500 MiB upload allocated %d bytes, want under 8 MiB", allocated)
	}
}
