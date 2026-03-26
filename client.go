package langsmith

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ty-cooper/langsmith-go/internal"
)

// Client is the main LangSmith API client.
type Client struct {
	apiKey     string
	endpoint   string
	project    string
	httpClient *http.Client
	maxRetries int
	batch      *internal.BatchWorker
}

// NewClient creates a new LangSmith client with the given options.
func NewClient(opts ...ClientOption) (*Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiKey == "" {
		return nil, &LangSmithError{Message: "API key is required. Set LANGCHAIN_API_KEY or use WithAPIKey()"}
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.timeout}
	}

	c := &Client{
		apiKey:     cfg.apiKey,
		endpoint:   strings.TrimRight(cfg.endpoint, "/"),
		project:    cfg.project,
		httpClient: httpClient,
		maxRetries: cfg.maxRetries,
	}

	batchCfg := internal.BatchWorkerConfig{
		APIURL:     c.endpoint,
		APIKey:     c.apiKey,
		HTTPClient: c.httpClient,
		BatchSize:  cfg.batchSize,
		Interval:   cfg.batchInterval,
	}
	if cfg.logger != nil {
		batchCfg.Logger = cfg.logger
	}
	if cfg.onBatchError != nil {
		batchCfg.OnError = cfg.onBatchError
	}
	c.batch = internal.NewBatchWorker(batchCfg)

	return c, nil
}

// Close shuts down the client and flushes any pending batched runs.
func (c *Client) Close() {
	if c.batch != nil {
		c.batch.Close()
	}
}

// Endpoint returns the configured API endpoint.
func (c *Client) Endpoint() string { return c.endpoint }

// Project returns the configured default project name.
func (c *Client) Project() string { return c.project }

// ServerInfo retrieves the LangSmith server info.
func (c *Client) ServerInfo(ctx context.Context) (*ServerInfo, error) {
	var info ServerInfo
	if err := c.get(ctx, "/info", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// --- HTTP Helpers ---

func (c *Client) get(ctx context.Context, path string, query url.Values, result any) error {
	return c.doRequest(ctx, http.MethodGet, path, query, nil, result)
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	return c.doRequest(ctx, http.MethodPost, path, nil, body, result)
}

func (c *Client) put(ctx context.Context, path string, body any, result any) error {
	return c.doRequest(ctx, http.MethodPut, path, nil, body, result)
}

func (c *Client) patch(ctx context.Context, path string, body any, result any) error {
	return c.doRequest(ctx, http.MethodPatch, path, nil, body, result)
}

func (c *Client) del(ctx context.Context, path string, query url.Values) error {
	return c.doRequest(ctx, http.MethodDelete, path, query, nil, nil)
}

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body any, result any) error {
	fullURL := c.endpoint + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	// Marshal body once, reuse for retries.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return &LangSmithError{Message: "failed to marshal request body", Err: err}
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return &LangSmithError{Message: "failed to create request", Err: err}
		}
		req.Header.Set("X-API-Key", c.apiKey)
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &LangSmithError{Message: "request failed", Err: err}
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = &LangSmithError{Message: "failed to read response body", Err: err}
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil && len(respBody) > 0 {
				if err := json.Unmarshal(respBody, result); err != nil {
					return &LangSmithError{Message: "failed to decode response", Err: err}
				}
			}
			return nil
		}

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
		// Try to extract message from JSON error response.
		var errResp struct {
			Detail string `json:"detail"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Detail != "" {
			apiErr.Message = errResp.Detail
		}

		if !apiErr.IsRetryable() {
			return apiErr
		}
		lastErr = apiErr
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// submitBatch submits a run to the background batch worker.
// Returns false if the queue is full or the worker has been closed.
func (c *Client) submitBatch(action string, payload any) bool {
	return c.batch.Submit(internal.BatchItem{
		Action:     action,
		RunPayload: payload,
	})
}
