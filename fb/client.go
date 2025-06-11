package fb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cenk/backoff"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// Client holds an http.Client and provides additional functionality.
type Client struct {
	l log.Logger
	*http.Client
}

// NewClient returns a http.Client containing a special transport with injects the version, token, and clientkey.
func NewClient(l log.Logger, token, clientKey string) *Client {
	if l == nil {
		l = log.NewNopLogger()
	}

	return &Client{
		l:      l,
		Client: &http.Client{Transport: newTokenTransport(token, clientKey, newRetryTransport(newLogAppUsageTransport(l, nil)))},
	}
}

func (c *Client) handleResponse(resp *http.Response, res interface{}, req []byte) error {
	defer resp.Body.Close()

	buf := &bytes.Buffer{}
	ec := &ErrorContainer{}
	err := json.NewDecoder(io.TeeReader(resp.Body, buf)).Decode(ec)
	if err != nil {
		return err
	} else if err = ec.GetError(); err != nil {
		c.handleError(err, resp, req)

		return err
	} else if resp.StatusCode != http.StatusOK {
		c.handleError(nil, resp, req)

		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	return json.Unmarshal(buf.Bytes(), res)
}

func (c *Client) handleError(err error, res *http.Response, req []byte) {
	if err == nil {
		_ = level.Warn(c.l).Log("msg", "received unexpected status code", "url", res.Request.URL.String(), "status", res.StatusCode, "method", res.Request.Method, "body", string(req))

		return
	}

	e, ok := err.(*Error)
	if !ok {
		_ = level.Warn(c.l).Log("msg", "received unexpected error", "url", res.Request.URL.String(), "status", res.StatusCode, "err", err, "type", fmt.Sprintf("%T", err), "method", res.Request.Method, "body", string(req))

		return
	}

	_ = level.Warn(c.l).Log("msg", "received facebook error", "url", res.Request.URL.String(), "status", res.StatusCode,
		"message", e.Message,
		"type", e.Type,
		"code", e.Code,
		"error_subcode", e.ErrorSubcode,
		"fbtrace_id", e.FbtraceID,
		"is_transient", e.IsTransient,
		"error_user_title", e.ErrorUserTitle,
		"error_user_msg", e.ErrorUserMsg,
		"error_data", e.ErrorData,
		"method", res.Request.Method,
		"body", string(req),
	)
}

// GetJSON retrieves url and parses the resulting body into v.
func (c *Client) GetJSON(ctx context.Context, url string, res interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	return c.handleResponse(resp, res, nil)
}

// GetList uses reflection to append to res when the result is a list.
func (c *Client) GetList(ctx context.Context, url string, res interface{}) error {
	stats := StatFromContext(ctx)
	for url != "" {
		resp := &listResponse{}
		err := c.GetJSON(ctx, url, resp)
		if err != nil {
			return err
		}

		n, err := appendJSON(resp.Data, res)
		if err != nil {
			return err
		}

		if stats != nil {
			stats.Add(uint64(n))
		}

		url = resp.Paging.Paging.Next
	}

	return nil
}

// ReadList writes json.RawMessage to a chan when the response is a list.
func (c *Client) ReadList(ctx context.Context, url string, res chan<- json.RawMessage) error {
	stats := StatFromContext(ctx)
	for url != "" {
		resp := &listElementsResponse{}
		err := c.GetJSON(ctx, url, resp)
		if err != nil {
			return err
		}

		for _, d := range resp.Data {
			res <- d
		}

		if stats != nil {
			stats.Add(uint64(len(resp.Data)))
		}

		url = resp.Paging.Paging.Next
	}

	return nil
}

// PostJSON encodes req as JSON into a buffer, sends this as a POST body to the url and parses the response as JSON into res.
func (c *Client) PostJSON(ctx context.Context, url string, req, res interface{}) error {
	var r io.Reader
	if req != nil {
		b := &bytes.Buffer{}
		err := json.NewEncoder(b).Encode(req)
		if err != nil {
			return err
		}
		r = b
	}

	var debugBuf *bytes.Buffer
	if r != nil {
		debugBuf = &bytes.Buffer{}
		r = io.TeeReader(r, debugBuf)
	}

	request, err := http.NewRequest(http.MethodPost, url, r)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(request.WithContext(ctx))
	if err != nil {
		return err
	}

	var b []byte
	if debugBuf != nil {
		b = debugBuf.Bytes()
	}

	return c.handleResponse(resp, res, b)
}

// Send a Post request encoded as a form.
func (c *Client) PostForm(ctx context.Context, endpointUrl string, formBody url.Values, res interface{}) error {
	var encodedBody io.Reader = strings.NewReader(formBody.Encode())
	var debugBuf *bytes.Buffer = &bytes.Buffer{}
	encodedBody = io.TeeReader(encodedBody, debugBuf)
	apiRequest, err := http.NewRequest(http.MethodPost, endpointUrl, encodedBody)
	if err != nil {
		return fmt.Errorf("cannot prepare request request: %w", err)
	}

	resp, err := c.Client.Do(apiRequest.WithContext(ctx))
	if err != nil {
		fmt.Printf("cannot execute the request: %s", err.Error())
		return err
	}

	return c.handleResponse(resp, res, debugBuf.Bytes())
}

// DeleteJSON sends a DELETE request to url with a body and marshals the response to res.
func (c *Client) DeleteJSON(ctx context.Context, url string, req, res interface{}) error {
	var r io.Reader
	if req != nil {
		b := &bytes.Buffer{}
		err := json.NewEncoder(b).Encode(req)
		if err != nil {
			return err
		}
		r = b
	}

	var debugBuf *bytes.Buffer
	if r != nil {
		debugBuf = &bytes.Buffer{}
		r = io.TeeReader(r, debugBuf)
	}

	httpReq, err := http.NewRequest(http.MethodDelete, url, r)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq.WithContext(ctx))
	if err != nil {
		return err
	}

	var b []byte
	if debugBuf != nil {
		b = debugBuf.Bytes()
	}

	return c.handleResponse(resp, res, b)
}

// PostValues sends an POST request to the Facebook Graph API.
func (c *Client) PostValues(ctx context.Context, u string, vals url.Values) error {
	if len(vals) == 0 {
		return nil
	}

	request, err := http.NewRequest(http.MethodPost, u, strings.NewReader(vals.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.Client.Do(request.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := &ErrorContainer{}
	err = json.NewDecoder(resp.Body).Decode(res)
	if err != nil {
		return err
	}

	return res.GetError()
}

// Delete sends a DELETE request to the given URL.
func (c *Client) Delete(ctx context.Context, url string) error {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	err = resp.Body.Close()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}

// UploadFile uses a multipart form for uploading a file from r.
func (c *Client) UploadFile(ctx context.Context, url, name string, r io.Reader, additionalFields map[string]string, res interface{}) error {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	for k, v := range additionalFields {
		w, err := bodyWriter.CreateFormField(k)
		if err != nil {
			return fmt.Errorf("err adding additional field '%s': %w", k, err)
		}

		_, err = io.Copy(w, strings.NewReader(v))
		if err != nil {
			return fmt.Errorf("err writing additional field '%s': %w", k, err)
		}
	}

	fileWriter, err := bodyWriter.CreateFormFile("video_file_chunk", name)
	if err != nil {
		return err
	}

	_, err = io.Copy(fileWriter, r)
	if err != nil {
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	b := bodyBuf.Bytes()

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 6 * time.Second
	bo.MaxElapsedTime = 5 * time.Minute

	return backoff.Retry(func() error {
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", contentType)

		resp, err := c.Client.Do(request.WithContext(ctx))
		if err != nil {
			return err
		}

		return c.handleResponse(resp, res, nil)
	}, backoff.WithContext(bo, ctx))
}
