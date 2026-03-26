package langsmith

import (
	"net/http"
	"time"
)

// ClientOption configures the Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	apiKey     string
	endpoint   string
	project    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	batchSize  int
	batchInterval time.Duration
}

func defaultConfig() *clientConfig {
	return &clientConfig{
		apiKey:        GetAPIKey(),
		endpoint:      GetEndpoint(),
		project:       GetProject(),
		timeout:       30 * time.Second,
		maxRetries:    3,
		batchSize:     100,
		batchInterval: time.Second,
	}
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) ClientOption {
	return func(c *clientConfig) {
		c.apiKey = key
	}
}

// WithEndpoint sets the LangSmith API endpoint URL.
func WithEndpoint(endpoint string) ClientOption {
	return func(c *clientConfig) {
		c.endpoint = endpoint
	}
}

// WithProject sets the default project name.
func WithProject(project string) ClientOption {
	return func(c *clientConfig) {
		c.project = project
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries for failed requests.
func WithMaxRetries(retries int) ClientOption {
	return func(c *clientConfig) {
		c.maxRetries = retries
	}
}

// WithBatchSize sets the maximum number of runs in a single batch request.
func WithBatchSize(size int) ClientOption {
	return func(c *clientConfig) {
		c.batchSize = size
	}
}

// WithBatchInterval sets how often the batch worker flushes.
func WithBatchInterval(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.batchInterval = d
	}
}
