package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type EdgeInvalidator struct {
	edgeURLs []string
	token    string
	client   *http.Client
}

func NewEdgeInvalidator(token string, edgeURLs ...string) *EdgeInvalidator {
	return &EdgeInvalidator{
		edgeURLs: edgeURLs,
		token:    token,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Invalidate sends a cache invalidation request to every configured edge.
func (i *EdgeInvalidator) Invalidate(assetID string) []error {
	errs := make([]error, len(i.edgeURLs))

	var wg sync.WaitGroup

	for index, edgeURL := range i.edgeURLs {
		wg.Add(1)

		go func(index int, edgeURL string) {
			defer wg.Done()

			endpoint := strings.TrimRight(edgeURL, "/") + "/internal/cache/" + url.PathEscape(assetID) + "/invalidate"

			req, err := http.NewRequest(http.MethodPost, endpoint, nil)
			if err != nil {
				errs[index] = fmt.Errorf(
					"create request for %s: %w", edgeURL, err,
				)
				return
			}

			req.Header.Set("X-Edge-Token", i.token)

			resp, err := i.client.Do(req)
			if err != nil {
				errs[index] = fmt.Errorf(
					"notify edge %s: %w", edgeURL, err,
				)
				return
			}
			defer resp.Body.Close()

			// Drain the response body so the connection can be reused.
			_, _ = io.Copy(io.Discard, resp.Body)

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				errs[index] = fmt.Errorf(
					"edge %s returned status %s",
					edgeURL,
					resp.Status,
				)
			}
		}(index, edgeURL)
	}

	wg.Wait()

	var failures []error

	for _, err := range errs {
		if err != nil {
			failures = append(failures, err)
		}
	}

	return failures
}
