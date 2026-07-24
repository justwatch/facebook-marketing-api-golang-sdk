package v24

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-kit/log"
	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

func TestCountCommentsByPostIDsBatchesAndResolvesAttachedObjects(t *testing.T) {
	postIDs := make([]string, 11)
	for i := range postIDs {
		postIDs[i] = fmt.Sprintf("post-%d", i)
	}
	postIDs[0] = "page_shell"

	var requests atomic.Int32
	client := fb.NewClient(log.NewNopLogger(), "token", "")
	client.Client = &http.Client{Transport: postCommentsRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		if request.URL.Path != "/v24.0/" {
			t.Fatalf("request path = %q, want %q", request.URL.Path, "/v24.0/")
		}
		if request.Method != http.MethodPost {
			t.Fatalf("request method = %q, want %q", request.Method, http.MethodPost)
		}

		requestBody, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read batch request: %v", err)
		}
		form, err := url.ParseQuery(string(requestBody))
		if err != nil {
			t.Fatalf("parse batch form: %v", err)
		}
		batch := []graphBatchRequest{}
		if err := json.Unmarshal([]byte(form.Get("batch")), &batch); err != nil {
			t.Fatalf("parse batch requests: %v", err)
		}
		if len(batch) == 0 || len(batch) > 10 {
			t.Fatalf("batch size = %d, want 1 through 10", len(batch))
		}

		batchResponses := make([]graphBatchResponse, len(batch))
		for i, batchRequest := range batch {
			requestURL, err := url.Parse(batchRequest.RelativeURL)
			if err != nil {
				t.Fatalf("parse relative URL: %v", err)
			}

			batchResponses[i] = graphBatchResponse{Code: http.StatusOK}
			switch {
			case strings.HasSuffix(requestURL.Path, "/comments"):
				if requestURL.Query().Get("limit") != "0" || requestURL.Query().Get("summary") != "true" {
					t.Fatalf("unexpected comments query: %q", requestURL.RawQuery)
				}
				postID := strings.TrimSuffix(requestURL.Path, "/comments")
				count := uint64(1)
				switch postID {
				case "page_shell", "post-2":
					count = 0
				case "post-1":
					count = 2
				case "post-10":
					count = 4
				case "reel":
					count = 3
				}
				batchResponses[i].Body = fmt.Sprintf(`{"summary":{"total_count":%d}}`, count)
			case strings.HasSuffix(requestURL.Path, "/attachments"):
				if got, want := requestURL.Query().Get("fields"), "target"; got != want {
					t.Fatalf("attachment fields = %q, want %q", got, want)
				}
				postID := strings.TrimSuffix(requestURL.Path, "/attachments")
				batchResponses[i].Body = `{"data":[]}`
				if postID == "page_shell" {
					batchResponses[i].Body = `{"data":[{"target":{"id":"reel"}}]}`
				}
			default:
				t.Fatalf("unexpected relative URL: %q", batchRequest.RelativeURL)
			}
		}
		body, err := json.Marshal(batchResponses)
		if err != nil {
			t.Fatalf("marshal batch responses: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Request:    request,
		}, nil
	})}
	service := &PostService{c: client}

	got, err := service.CountCommentsByPostIDs(context.Background(), postIDs)
	if err != nil {
		t.Fatalf("CountCommentsByPostIDs() error = %v", err)
	}
	if requests.Load() != 4 {
		t.Fatalf("requests = %d, want 4", requests.Load())
	}

	want := map[string]uint64{
		"page_shell": 3,
		"post-1":     2,
		"post-2":     0,
		"post-10":    4,
	}
	for postID, wantCount := range want {
		if count, ok := got[postID]; !ok || count != wantCount {
			t.Fatalf("count for %s = %d, %t; want %d, true", postID, count, ok, wantCount)
		}
	}
	if count := got["post-3"]; count != 1 {
		t.Fatalf("count for post-3 = %d, want 1", count)
	}
}

type postCommentsRoundTripFunc func(*http.Request) (*http.Response, error)

func (f postCommentsRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
