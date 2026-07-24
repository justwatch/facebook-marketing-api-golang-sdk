package v24

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
	"golang.org/x/sync/errgroup"
)

type graphBatchRequest struct {
	Method      string `json:"method"`
	RelativeURL string `json:"relative_url"`
}

type graphBatchResponse struct {
	Code int    `json:"code"`
	Body string `json:"body"`
}

func (ps *PostService) getBatch(ctx context.Context, relativeURLs []string) ([]graphBatchResponse, error) {
	batch := make([]graphBatchRequest, len(relativeURLs))
	for i, relativeURL := range relativeURLs {
		batch[i] = graphBatchRequest{
			Method:      http.MethodGet,
			RelativeURL: relativeURL,
		}
	}
	batchJSON, err := json.Marshal(batch)
	if err != nil {
		return nil, err
	}

	form := url.Values{"batch": {string(batchJSON)}}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fb.NewRoute(Version, "/").String(),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := ps.c.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		if graphErr := graphError(body); graphErr != nil {
			return nil, graphErr
		}
		return nil, fmt.Errorf("unexpected status %s", response.Status)
	}

	batchResponses := []graphBatchResponse{}
	if err := json.Unmarshal(body, &batchResponses); err != nil {
		return nil, err
	}
	if len(batchResponses) != len(relativeURLs) {
		return nil, fmt.Errorf("received %d Facebook batch responses for %d requests", len(batchResponses), len(relativeURLs))
	}
	return batchResponses, nil
}

func (ps *PostService) getBatches(ctx context.Context, relativeURLs []string, batchSize, concurrency int) ([]graphBatchResponse, error) {
	responses := make([]graphBatchResponse, len(relativeURLs))
	group, groupCtx := errgroup.WithContext(ctx)
	semaphore := make(chan struct{}, concurrency)

	for start := 0; start < len(relativeURLs); start += batchSize {
		end := start + batchSize
		if end > len(relativeURLs) {
			end = len(relativeURLs)
		}

		batchStart := start
		batchURLs := append([]string(nil), relativeURLs[start:end]...)
		group.Go(func() error {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-groupCtx.Done():
				return groupCtx.Err()
			}

			batchResponses, err := ps.getBatch(groupCtx, batchURLs)
			if err != nil {
				return err
			}
			copy(responses[batchStart:batchStart+len(batchResponses)], batchResponses)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return responses, nil
}

func decodeGraphBatchResponse(response graphBatchResponse, target interface{}) error {
	if response.Code != http.StatusOK {
		if graphErr := graphError([]byte(response.Body)); graphErr != nil {
			return graphErr
		}
		return fmt.Errorf("unexpected Facebook batch status %d", response.Code)
	}
	return json.Unmarshal([]byte(response.Body), target)
}

func graphError(body []byte) error {
	container := fb.ErrorContainer{}
	if err := json.Unmarshal(body, &container); err != nil {
		return nil
	}
	return container.GetError()
}
