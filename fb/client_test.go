package fb

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-kit/log"
)

// TestGetList_SkipsObjectMetaCannotSerialize: halve to limit=1, skip the object via an id-only read, resume with the original fields and limit.
func TestGetList_SkipsObjectMetaCannotSerialize(t *testing.T) {
	const reduceData = `{"error":{"message":"Please reduce the amount of data you're asking for, then retry your request","code":1}}`
	var urls []string

	fake := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		urls = append(urls, req.URL.String())
		q := req.URL.Query()
		body, status := reduceData, http.StatusInternalServerError
		switch {
		case q.Get("fields") == "id" && q.Get("limit") == "1":
			body, status = `{"data":[{"id":"bad"}],"paging":{"next":"https://graph.facebook.com/v24.0/act_1/campaigns?fields=id&limit=1&after=c2"}}`, http.StatusOK
		case q.Get("after") == "c2":
			body, status = `{"data":[{"id":"good","name":"x"}],"paging":{}}`, http.StatusOK
		}

		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	c := &Client{l: log.NewNopLogger(), Client: &http.Client{Transport: fake}}

	var res []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := c.GetList(context.Background(), "https://graph.facebook.com/v24.0/act_1/campaigns?fields=id%2Cname&limit=2", &res)
	if err != nil {
		t.Fatalf("GetList: %v", err)
	}
	if len(res) != 1 || res[0].ID != "good" || res[0].Name != "x" {
		t.Fatalf("got %+v, want only the readable object", res)
	}

	want := []string{
		"https://graph.facebook.com/v24.0/act_1/campaigns?fields=id%2Cname&limit=2",
		"https://graph.facebook.com/v24.0/act_1/campaigns?fields=id%2Cname&limit=1",
		"https://graph.facebook.com/v24.0/act_1/campaigns?fields=id&limit=1",
		"https://graph.facebook.com/v24.0/act_1/campaigns?after=c2&fields=id%2Cname&limit=2",
	}
	if strings.Join(urls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(urls, "\n"), strings.Join(want, "\n"))
	}
}
