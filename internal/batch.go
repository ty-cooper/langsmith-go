package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// BatchItem represents a single item to be batched for ingestion.
type BatchItem struct {
	// Action is either "post" (create) or "patch" (update).
	Action string
	// RunPayload is the JSON-serializable run data.
	RunPayload interface{}
}

// BatchWorker handles background batching and ingestion of runs.
type BatchWorker struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
	queue      chan BatchItem
	maxSize    int
	interval   time.Duration
	wg         sync.WaitGroup
	cancel     context.CancelFunc
	ctx        context.Context
}

// BatchWorkerConfig holds configuration for the batch worker.
type BatchWorkerConfig struct {
	APIURL     string
	APIKey     string
	HTTPClient *http.Client
	QueueSize  int
	BatchSize  int
	Interval   time.Duration
}

// NewBatchWorker creates and starts a new background batch worker.
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

	ctx, cancel := context.WithCancel(context.Background())
	w := &BatchWorker{
		apiURL:     cfg.APIURL,
		apiKey:     cfg.APIKey,
		httpClient: cfg.HTTPClient,
		queue:      make(chan BatchItem, cfg.QueueSize),
		maxSize:    cfg.BatchSize,
		interval:   cfg.Interval,
		cancel:     cancel,
		ctx:        ctx,
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// Submit adds an item to the batch queue. Non-blocking; drops if queue is full.
func (w *BatchWorker) Submit(item BatchItem) {
	select {
	case w.queue <- item:
	default:
		// Queue full — drop silently to avoid blocking the caller.
	}
}

// Close stops the worker and flushes remaining items.
func (w *BatchWorker) Close() {
	w.cancel()
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
		case <-w.ctx.Done():
			// Drain remaining items from queue.
			close(w.queue)
			for item := range w.queue {
				batch = append(batch, item)
			}
			if len(batch) > 0 {
				w.flush(batch)
			}
			return
		}
	}
}

func (w *BatchWorker) flush(batch []BatchItem) {
	if len(batch) == 0 {
		return
	}

	var posts []interface{}
	var patches []interface{}
	for _, item := range batch {
		switch item.Action {
		case "post":
			posts = append(posts, item.RunPayload)
		case "patch":
			patches = append(patches, item.RunPayload)
		}
	}

	payload := map[string]interface{}{}
	if len(posts) > 0 {
		payload["post"] = posts
	}
	if len(patches) > 0 {
		payload["patch"] = patches
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	url := fmt.Sprintf("%s/runs/batch", w.apiURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", w.apiKey)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
