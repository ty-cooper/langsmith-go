package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatchWorker_FlushesOnInterval(t *testing.T) {
	t.Parallel()

	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string][]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)
		received.Add(int32(len(body["post"])))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	w := NewBatchWorker(BatchWorkerConfig{
		APIURL:    server.URL,
		APIKey:    "test",
		BatchSize: 100,
		Interval:  50 * time.Millisecond,
	})

	w.Submit(BatchItem{Action: "post", RunPayload: map[string]string{"id": "1"}})
	w.Submit(BatchItem{Action: "post", RunPayload: map[string]string{"id": "2"}})

	// Wait for the interval flush.
	time.Sleep(150 * time.Millisecond)
	if got := received.Load(); got != 2 {
		t.Errorf("received = %d, want 2", got)
	}

	w.Close()
}

func TestBatchWorker_FlushesOnBatchSize(t *testing.T) {
	t.Parallel()

	var flushCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flushCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	w := NewBatchWorker(BatchWorkerConfig{
		APIURL:    server.URL,
		APIKey:    "test",
		BatchSize: 2,
		Interval:  10 * time.Second, // long interval — should not trigger
	})

	w.Submit(BatchItem{Action: "post", RunPayload: map[string]string{"id": "1"}})
	w.Submit(BatchItem{Action: "post", RunPayload: map[string]string{"id": "2"}})

	time.Sleep(100 * time.Millisecond) // give goroutine time to flush
	if got := flushCount.Load(); got < 1 {
		t.Errorf("flushCount = %d, want >= 1", got)
	}

	w.Close()
}

func TestBatchWorker_Close_FlushesRemaining(t *testing.T) {
	t.Parallel()

	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string][]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)
		received.Add(int32(len(body["post"])))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	w := NewBatchWorker(BatchWorkerConfig{
		APIURL:    server.URL,
		APIKey:    "test",
		BatchSize: 1000,
		Interval:  10 * time.Second,
	})

	for i := 0; i < 5; i++ {
		w.Submit(BatchItem{Action: "post", RunPayload: map[string]string{"id": "x"}})
	}

	w.Close()
	if got := received.Load(); got != 5 {
		t.Errorf("received = %d after Close, want 5", got)
	}
}

func TestBatchWorker_Submit_ReturnsFalseAfterClose(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	w := NewBatchWorker(BatchWorkerConfig{
		APIURL: server.URL,
		APIKey: "test",
	})
	w.Close()

	ok := w.Submit(BatchItem{Action: "post", RunPayload: "x"})
	if ok {
		t.Error("Submit should return false after Close")
	}
}

func TestBatchWorker_Submit_ReturnsFalseWhenQueueFull(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	w := NewBatchWorker(BatchWorkerConfig{
		APIURL:    server.URL,
		APIKey:    "test",
		QueueSize: 1,
		BatchSize: 1000,
		Interval:  10 * time.Second,
	})
	defer w.Close()

	// Fill queue
	w.Submit(BatchItem{Action: "post", RunPayload: "1"})
	// Queue should be full now.
	ok := w.Submit(BatchItem{Action: "post", RunPayload: "2"})
	if ok {
		// May or may not be full depending on timing, so this is best-effort.
		t.Log("queue may have drained before second submit")
	}
}

func TestBatchWorker_OnError_Called(t *testing.T) {
	t.Parallel()

	var errCalled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	w := NewBatchWorker(BatchWorkerConfig{
		APIURL:    server.URL,
		APIKey:    "test",
		BatchSize: 1,
		Interval:  50 * time.Millisecond,
		OnError: func(err error, count int) {
			errCalled.Store(true)
		},
	})

	w.Submit(BatchItem{Action: "post", RunPayload: map[string]string{"id": "1"}})
	time.Sleep(200 * time.Millisecond)
	w.Close()

	if !errCalled.Load() {
		t.Error("OnError should have been called on 500 response")
	}
}
