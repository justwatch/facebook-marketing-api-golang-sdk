package fb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// uploadSource is a replayable upload body. Seekable input is read in place;
// anything else is spooled once to a temporary file. Either way a retry can
// re-send the file without the SDK ever holding it in memory.
type uploadSource struct {
	data  io.ReaderAt
	start int64
	size  int64
	tmp   *os.File
}

func newUploadSource(r io.Reader) (*uploadSource, error) {
	if f, ok := r.(interface {
		io.ReaderAt
		io.Seeker
	}); ok {
		start, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("seek upload: %w", err)
		}
		end, err := f.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("seek upload end: %w", err)
		}
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return nil, fmt.Errorf("rewind upload: %w", err)
		}

		return &uploadSource{data: f, start: start, size: end - start}, nil
	}

	tmp, err := os.CreateTemp("", "fb-upload-*")
	if err != nil {
		return nil, fmt.Errorf("create upload spool: %w", err)
	}
	size, err := io.Copy(tmp, r)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())

		return nil, fmt.Errorf("spool upload: %w", err)
	}

	return &uploadSource{data: tmp, size: size, tmp: tmp}, nil
}

// reader returns an independent reader over the whole upload.
func (s *uploadSource) reader() *io.SectionReader {
	return io.NewSectionReader(s.data, s.start, s.size)
}

func (s *uploadSource) close() error {
	if s.tmp == nil {
		return nil
	}
	err := s.tmp.Close()
	if rmErr := os.Remove(s.tmp.Name()); err == nil {
		err = rmErr
	}

	return err
}

// newMultipartRequest streams the upload between small multipart headers and
// sets GetBody so the retry transport can replay it without buffering.
func newMultipartRequest(ctx context.Context, url, fileField, fileName string, fields map[string]string, src *uploadSource) (*http.Request, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("write field %q: %w", k, err)
		}
	}
	if _, err := w.CreateFormFile(fileField, fileName); err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	prefix := append([]byte(nil), buf.Bytes()...)
	buf.Reset()
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}
	suffix := append([]byte(nil), buf.Bytes()...)

	body := func() (io.ReadCloser, error) {
		return io.NopCloser(io.MultiReader(bytes.NewReader(prefix), src.reader(), bytes.NewReader(suffix))), nil
	}
	first, _ := body()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, first)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}
	req.ContentLength = int64(len(prefix)) + src.size + int64(len(suffix))
	req.GetBody = body
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req, nil
}
