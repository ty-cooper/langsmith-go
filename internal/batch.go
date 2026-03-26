package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// BatchItem represents a single item to be batched for ingestion.
type BatchItem struct {
	// Action is either "post" (create) or "patch" (update).
	Action string
	// RunPayload is the JSON-serializable run data.
	RunPayload any
}

// OnFlushErrorFunc is called when a batch flush fails.
// The error and the number of items that were lost are provided.
type OnFlushErrorFunc func(err error, itemCount int)

// BatchWorker handles background batching and ingestion of runs.
type BatchWorker struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
	queue      chan BatchItem
	maxSize    int
	interval   time.Duration
	onError    OnFlushErrorFunc
	logger     *slog.Logger

	closed atomic.Bool
	wg     sync.WaitGroup
	done   chan struct{}
}

// BatchWorkerConfig holds configuration for the batch worker.
type BatchWorkerConfig struct {
	APIURL     string
	APIKey     string
	HTTPClient *http.Client
	QueueSize  int
	BatchSize  int
	Interval   time.Duration
	OnError    OnFlushErrorFunc
	Logger     *slog.Logger
}

// NewBatchWorker creates and starts a new background batch worker.
//
// The worker runs a single goroutine that collects items from the queue
// and flushes them in batches. Call Close to stop the worker and flush
// remaining items.
func NewBatchWorker(cfg BatchWorkerConfig) *BatchWorker {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1000
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	w := &BatchWorker{
		apiURL:     cfg.APIURL,
		apiKey:     cfg.APIKey,
		httpClient: cfg.HTTPClient,
		queue:      make(chan BatchItem, cfg.QueueSize),
		maxSize:    cfg.BatchSize,
		interval:   cfg.Interval,
		onError:    cfg.OnError,
		logger:     cfg.Logger,
		done:       make(chan struct{}),
	}

	w.wg.Add(1)
	go w.run()
	return w
}

// Submit adds an item to the batch queue. Non-blocking.
// Returns false if the worker is closed or the queue is full.
func (w *BatchWorker) Submit(item BatchItem) bool {
	if w.closed.Load() {
		return false
	}
	select {
	case w.queue <- item:
		return true
	default:
		return false
	}
}

// Close signals the worker to stop, then blocks until all queued items
// are flushed. Safe to call multiple times.
func (w *BatchWorker) Close() {
	if w.closed.CompareAndSwap(false, true) {
		close(w.done)
	}
	w.wg.Wait()
}

func (w *BatchWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	var batch []BatchItem

	for {
		select {
		case item := <-w.queue:
			batch = append(batch, item)
			if len(batch) >= w.maxSize {
				w.flush(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				w.flush(batch)
				batch = nil
			}
		case <-w.done:
			// Drain remaining items from queue without closing the channel.
			for {
				select {
				case item := <-w.queue:
					batch = append(batch, item)
				default:
					if len(batch) > 0 {
						w.flush(batch)
					}
					return
				}
			}
		}
	}
}

func (w *BatchWorker) flush(batch []BatchItem) {
	if len(batch) == 0 {
		return
	}

	var posts []any
	var patches []any
	for _, item := range batch {
		switch item.Action {
		case "post":
			posts = append(posts, item.RunPayload)
		case "patch":
			patches = append(patches, item.RunPayload)
		}
	}

	payload := map[string]any{}
	if len(posts) > 0 {
		payload["post"] = posts
	}
	if len(patches) > 0 {
		payload["patch"] = patches
	}

	body, err := json.Marshal(payload)
	if err != nil {
		w.reportError(fmt.Errorf("batch flush: marshal payload: %w", err), len(batch))
		return
	}

	url := fmt.Sprintf("%s/runs/batch", w.apiURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		w.reportError(fmt.Errorf("batch flush: create request: %w", err), len(batch))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", w.apiKey)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.reportError(fmt.Errorf("batch flush: http request: %w", err), len(batch))
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		w.reportError(fmt.Errorf("batch flush: unexpected status %d", resp.StatusCode), len(batch))
	}
}

func (w *BatchWorker) reportError(err error, itemCount int) {
	if w.onError != nil {
		w.onError(err, itemCount)
		return
	}
	w.logger.Error("langsmith batch flush failed",
		"error", err,
		"items_lost", itemCount,
	)
}
