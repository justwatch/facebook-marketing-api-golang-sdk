package v24

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-kit/log"
	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

func TestGetInstagramPermalinksByMediaIDs(t *testing.T) {
	mediaIDs := make([]string, 26)
	for i := range mediaIDs {
		mediaIDs[i] = fmt.Sprintf("media-%d", i)
	}

	var requests atomic.Int32
	client := fb.NewClient(log.NewNopLogger(), "token", "")
	client.Client = &http.Client{Transport: instagramCommentsRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
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

		batchResponses := make([]graphBatchResponse, len(batch))
		for i, batchRequest := range batch {
			requestURL, err := url.Parse(batchRequest.RelativeURL)
			if err != nil {
				t.Fatalf("parse relative URL: %v", err)
			}
			if got, want := requestURL.Query().Get("fields"), "permalink"; got != want {
				t.Fatalf("fields = %q, want %q", got, want)
			}
			body, err := json.Marshal(struct {
				Permalink string `json:"permalink"`
			}{Permalink: "https://instagram.com/p/" + requestURL.Path})
			if err != nil {
				t.Fatalf("marshal permalink response: %v", err)
			}
			batchResponses[i] = graphBatchResponse{Code: http.StatusOK, Body: string(body)}
		}
		responseBody, err := json.Marshal(batchResponses)
		if err != nil {
			t.Fatalf("marshal batch responses: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(string(responseBody))),
			Request:    request,
		}, nil
	})}
	service := &PostService{c: client}

	got, err := service.GetInstagramPermalinksByMediaIDs(context.Background(), mediaIDs)
	if err != nil {
		t.Fatalf("GetInstagramPermalinksByMediaIDs() error = %v", err)
	}
	if requests.Load() != 2 {
		t.Fatalf("requests = %d, want 2", requests.Load())
	}
	if permalink := got["media-25"]; permalink != "https://instagram.com/p/media-25" {
		t.Fatalf("permalink for media-25 = %q", permalink)
	}
}

func TestListInstagramCommentsByMediaIDs(t *testing.T) {
	mediaIDs := make([]string, 26)
	for i := range mediaIDs {
		mediaIDs[i] = fmt.Sprintf("media-%d", i)
	}

	var requests atomic.Int32
	client := fb.NewClient(log.NewNopLogger(), "token", "")
	client.Client = &http.Client{Transport: instagramCommentsRoundTripFunc(func(request *http.Request) (*http.Response, error) {
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

		if len(batch) == 0 || len(batch) > 25 {
			t.Fatalf("batch size = %d, want 1 through 25", len(batch))
		}

		batchResponses := make([]graphBatchResponse, len(batch))
		for i, batchRequest := range batch {
			requestURL, err := url.Parse(batchRequest.RelativeURL)
			if err != nil {
				t.Fatalf("parse relative URL: %v", err)
			}
			if got, want := requestURL.Query().Get("fields"), "comments.limit(1000){id,text,replies.limit(1000){id,text}}"; got != want {
				t.Fatalf("fields = %q, want %q", got, want)
			}

			batchResponses[i] = graphBatchResponse{Code: http.StatusOK, Body: `{}`}
			switch requestURL.Path {
			case "media-0":
				batchResponses[i].Body = `{
					"comments": {
						"data": [{
							"id": "comment-1",
							"text": "first",
							"replies": {
								"data": [{"id": "reply-1", "text": "reply"}]
							}
						}]
					}
				}`
			case "media-25":
				batchResponses[i].Body = `{
					"comments": {
						"data": [{"id": "comment-2", "text": "second"}]
					}
				}`
			}
		}
		responseBody, err := json.Marshal(batchResponses)
		if err != nil {
			t.Fatalf("marshal batch responses: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(string(responseBody))),
			Request:    request,
		}, nil
	})}
	service := &PostService{c: client}

	got, err := service.ListInstagramCommentsByMediaIDs(context.Background(), mediaIDs)
	if err != nil {
		t.Fatalf("ListInstagramCommentsByMediaIDs() error = %v", err)
	}
	if requests.Load() != 2 {
		t.Fatalf("requests = %d, want 2", requests.Load())
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
