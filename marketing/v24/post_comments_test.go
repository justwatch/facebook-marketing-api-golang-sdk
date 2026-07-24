package v24

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

func TestCountCommentsByPostIDsBatchesAndResolvesAttachedObjects(t *testing.T) {
	postIDs := make([]string, 51)
	for i := range postIDs {
		postIDs[i] = fmt.Sprintf("post-%d", i)
	}
	postIDs[0] = "page_shell"

	requests := 0
	client := fb.NewClient(log.NewNopLogger(), "token", "")
	client.Client = &http.Client{Transport: postCommentsRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.URL.Path != "/v24.0/" {
			t.Fatalf("request path = %q, want %q", request.URL.Path, "/v24.0/")
		}
		if got, want := request.URL.Query().Get("fields"), "object_id,comments.limit(0).summary(true)"; got != want {
			t.Fatalf("fields = %q, want %q", got, want)
		}

		var body string
		switch request.URL.Query().Get("ids") {
		case strings.Join(postIDs[:50], ","):
			body = `{
				"page_shell": {
					"object_id": "reel",
					"comments": {"summary": {"total_count": 0}}
				},
				"post-1": {
					"comments": {"summary": {"total_count": 2}}
				},
				"post-2": {
					"comments": {"summary": {"total_count": 0}}
				}
			}`
		case postIDs[50]:
			body = `{
				"post-50": {
					"comments": {"summary": {"total_count": 4}}
				}
			}`
		case "reel":
			body = `{
				"reel": {
					"comments": {"summary": {"total_count": 3}}
				}
			}`
		default:
			t.Fatalf("unexpected ids query: %q", request.URL.Query().Get("ids"))
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}
	service := &PostService{c: client}

	got, err := service.CountCommentsByPostIDs(context.Background(), postIDs)
	if err != nil {
		t.Fatalf("CountCommentsByPostIDs() error = %v", err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}

	want := map[string]uint64{
		"page_shell": 3,
		"post-1":     2,
		"post-2":     0,
		"post-50":    4,
	}
	for postID, wantCount := range want {
		if count, ok := got[postID]; !ok || count != wantCount {
			t.Fatalf("count for %s = %d, %t; want %d, true", postID, count, ok, wantCount)
		}
	}
	if _, ok := got["post-3"]; ok {
		t.Fatal("missing API response for post-3 should not produce a count")
	}
}

type postCommentsRoundTripFunc func(*http.Request) (*http.Response, error)

func (f postCommentsRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
