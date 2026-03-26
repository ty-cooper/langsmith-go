package langsmith

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
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

	batchOnce sync.Once
	batchCfg  internal.BatchWorkerConfig
	batch     *internal.BatchWorker
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

	if !cfg.allowInsecureHTTP && !strings.HasPrefix(cfg.endpoint, "https://") {
		return nil, &LangSmithError{Message: "endpoint must use HTTPS when transmitting API keys. Use WithAllowInsecureHTTP() to override for local development"}
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

	c.batchCfg = internal.BatchWorkerConfig{
		APIURL:     c.endpoint,
		APIKey:     c.apiKey,
		HTTPClient: c.httpClient,
		BatchSize:  cfg.batchSize,
		Interval:   cfg.batchInterval,
	}
	if cfg.logger != nil {
		c.batchCfg.Logger = cfg.logger
	}
	if cfg.onBatchError != nil {
		c.batchCfg.OnError = cfg.onBatchError
	}

	return c, nil
}

// Close shuts down the client and flushes any pending batched runs.
// Safe to call even if no batched operations were ever performed.
func (c *Client) Close() {
	// Run through the Once so we either get the existing worker or
	// create one that immediately drains (no items queued). This
	// avoids racing with a concurrent initBatch() call.
	c.initBatch().Close()
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

// maxResponseBytes is the maximum response body size the client will read.
// This prevents OOM from a misbehaving or compromised server.
const maxResponseBytes = 10 * 1024 * 1024 // 10 MB

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
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
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
			if !isRetryableNetError(err) {
				return lastErr
			}
			continue
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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

		apiErr := parseAPIError(resp.StatusCode, respBody)
		if !apiErr.IsRetryable() {
			return apiErr
		}
		lastErr = apiErr
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequestRaw is like doRequest but accepts pre-built body bytes and a custom
// content type. Used for multipart uploads and other non-JSON request bodies.
func (c *Client) doRequestRaw(ctx context.Context, method, path string, bodyBytes []byte, contentType string, result any) error {
	fullURL := c.endpoint + path

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return &LangSmithError{Message: "failed to create request", Err: err}
		}
		req.Header.Set("X-API-Key", c.apiKey)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = &LangSmithError{Message: "request failed", Err: err}
			if !isRetryableNetError(err) {
				return lastErr
			}
			continue
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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

		apiErr := parseAPIError(resp.StatusCode, respBody)
		if !apiErr.IsRetryable() {
			return apiErr
		}
		lastErr = apiErr
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// maxErrorBodyBytes is the maximum number of bytes stored in APIError.Body
// to prevent sensitive server responses from propagating through error chains.
const maxErrorBodyBytes = 512

// parseAPIError builds an APIError from a non-2xx response, truncating the
// body to avoid leaking large or sensitive payloads into error strings/logs.
func parseAPIError(statusCode int, respBody []byte) *APIError {
	body := string(respBody)
	if len(body) > maxErrorBodyBytes {
		body = body[:maxErrorBodyBytes] + "...(truncated)"
	}
	apiErr := &APIError{
		StatusCode: statusCode,
		Body:       body,
	}
	var errResp struct {
		Detail string `json:"detail"`
	}
	if json.Unmarshal(respBody, &errResp) == nil && errResp.Detail != "" {
		apiErr.Message = errResp.Detail
	}
	return apiErr
}

// initBatch lazily starts the background batch worker on first use.
func (c *Client) initBatch() *internal.BatchWorker {
	c.batchOnce.Do(func() {
		c.batch = internal.NewBatchWorker(c.batchCfg)
	})
	return c.batch
}

// submitBatch submits a run to the background batch worker.
// Returns false if the queue is full or the worker has been closed.
func (c *Client) submitBatch(action string, payload any) bool {
	return c.initBatch().Submit(internal.BatchItem{
		Action:     action,
		RunPayload: payload,
	})
}

// isRetryableNetError returns true for transient network errors (timeouts,
// temporary failures). Non-transient errors like DNS resolution failures
// or TLS handshake errors are not retried.
func isRetryableNetError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}
