package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls the external ML severity prediction service.
type Client interface {
	Health(ctx context.Context) (*HealthResponse, error)
	Score(ctx context.Context, req ScoreRequest) (*ScoreResponse, error)
	Feedback(ctx context.Context, req FeedbackRequest) (*FeedbackResponse, error)
}

type httpClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &httpClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *httpClient) Health(ctx context.Context) (*HealthResponse, error) {
	var out HealthResponse
	if err := c.doJSON(ctx, http.MethodGet, "/health", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) Score(ctx context.Context, req ScoreRequest) (*ScoreResponse, error) {
	var out ScoreResponse
	if err := c.doJSON(ctx, http.MethodPost, "/score", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) Feedback(ctx context.Context, req FeedbackRequest) (*FeedbackResponse, error) {
	var out FeedbackResponse
	if err := c.doJSON(ctx, http.MethodPost, "/feedback", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ml service request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ml service %s %s: status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode ml response: %w", err)
		}
	}
	return nil
}
