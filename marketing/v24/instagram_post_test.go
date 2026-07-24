package v24

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

func TestListInstagramCommentsByMediaIDs(t *testing.T) {
	mediaIDs := make([]string, 26)
	for i := range mediaIDs {
		mediaIDs[i] = fmt.Sprintf("media-%d", i)
	}

	requests := 0
	client := fb.NewClient(log.NewNopLogger(), "token", "")
	client.Client = &http.Client{Transport: instagramCommentsRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.URL.Path != "/v24.0/" {
			t.Fatalf("request path = %q, want %q", request.URL.Path, "/v24.0/")
		}
		if got, want := request.URL.Query().Get("fields"), "comments.limit(1000){id,text,replies.limit(1000){id,text}}"; got != want {
			t.Fatalf("fields = %q, want %q", got, want)
		}

		var body string
		switch request.URL.Query().Get("ids") {
		case strings.Join(mediaIDs[:25], ","):
			body = `{
				"media-0": {
					"comments": {
						"data": [{
							"id": "comment-1",
							"text": "first",
							"replies": {
								"data": [{"id": "reply-1", "text": "reply"}]
							}
						}]
					}
				}
			}`
		case mediaIDs[25]:
			body = `{
				"media-25": {
					"comments": {
						"data": [{"id": "comment-2", "text": "second"}]
					}
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

	got, err := service.ListInstagramCommentsByMediaIDs(context.Background(), mediaIDs)
	if err != nil {
		t.Fatalf("ListInstagramCommentsByMediaIDs() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}

	wantFirst := []InstagramComment{
		{ID: "comment-1", Text: "first"},
		{ID: "reply-1", Text: "reply"},
	}
	if !reflect.DeepEqual(got["media-0"], wantFirst) {
		t.Fatalf("comments for media-0 = %#v, want %#v", got["media-0"], wantFirst)
	}
	if len(got["media-1"]) != 0 {
		t.Fatalf("comments for media-1 = %#v, want empty", got["media-1"])
	}
	wantLast := []InstagramComment{{ID: "comment-2", Text: "second"}}
	if !reflect.DeepEqual(got["media-25"], wantLast) {
		t.Fatalf("comments for media-25 = %#v, want %#v", got["media-25"], wantLast)
	}
}

type instagramCommentsRoundTripFunc func(*http.Request) (*http.Response, error)

func (f instagramCommentsRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
