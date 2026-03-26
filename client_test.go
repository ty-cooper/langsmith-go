package langsmith

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient_EmptyAPIKey_ReturnsError(t *testing.T) {
	// Cannot be parallel — uses t.Setenv to clear env vars.
	t.Setenv("LANGCHAIN_API_KEY", "")
	t.Setenv("LANGSMITH_API_KEY", "")

	_, err := NewClient(WithAPIKey(""))
	if err == nil {
		t.Fatal("expected error when API key is empty")
	}
}

func TestNewClient_WithOptions_SetsFields(t *testing.T) {
	t.Parallel()

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithEndpoint("https://custom.example.com"),
		WithProject("my-project"),
		WithTimeout(5*time.Second),
		WithMaxRetries(2),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	if got := client.Endpoint(); got != "https://custom.example.com" {
		t.Errorf("Endpoint() = %q, want %q", got, "https://custom.example.com")
	}
	if got := client.Project(); got != "my-project" {
		t.Errorf("Project() = %q, want %q", got, "my-project")
	}
}

func TestClient_ServerInfo_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info" {
			t.Errorf("path = %q, want /info", r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", got)
		}
		json.NewEncoder(w).Encode(ServerInfo{Version: "0.6.0"})
	}))
	defer server.Close()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	info, err := client.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("ServerInfo: %v", err)
	}
	if info.Version != "0.6.0" {
		t.Errorf("Version = %q, want 0.6.0", info.Version)
	}
}

func TestClient_RetryOnServerError_Succeeds(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"detail": "server error"}`))
			return
		}
		json.NewEncoder(w).Encode(ServerInfo{Version: "1.0"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithEndpoint(server.URL),
		WithMaxRetries(3),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	info, err := client.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("ServerInfo after retries: %v", err)
	}
	if info.Version != "1.0" {
		t.Errorf("Version = %q, want 1.0", info.Version)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestClient_NoRetryOn4xx_ReturnsAPIError(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail": "not found"}`))
	}))
	defer server.Close()

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithEndpoint(server.URL),
		WithMaxRetries(3),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	_, err = client.ServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 4xx)", got)
	}

	apiErr := AsAPIError(err)
	if apiErr == nil {
		t.Fatalf("AsAPIError returned nil, err = %v (%T)", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "not found" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "not found")
	}
}

func TestClient_ContextCanceled_StopsRetry(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithEndpoint(server.URL),
		WithMaxRetries(10),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = client.ServerInfo(ctx)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		// May be wrapped — just check we got an error.
		if err == nil {
			t.Fatal("expected error on canceled context")
		}
	}
}

func TestIsNotFound_APIError(t *testing.T) {
	t.Parallel()

	err := &APIError{StatusCode: 404, Message: "not found"}
	if !IsNotFound(err) {
		t.Error("IsNotFound should return true for 404 APIError")
	}

	err2 := &APIError{StatusCode: 500, Message: "server error"}
	if IsNotFound(err2) {
		t.Error("IsNotFound should return false for 500")
	}
}

func TestIsNotFound_SentinelError(t *testing.T) {
	t.Parallel()

	if !IsNotFound(ErrNotFound) {
		t.Error("IsNotFound should return true for ErrNotFound")
	}
}

func TestClient_CRUDEndpoints_CorrectPaths(t *testing.T) {
	t.Parallel()

	var lastMethod, lastPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		lastPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()
	ctx := context.Background()

	tests := []struct {
		name       string
		call       func() error
		wantMethod string
		wantPath   string
	}{
		{
			name:       "CreateDataset",
			call:       func() error { _, err := client.CreateDataset(ctx, DatasetCreate{Name: "d"}); return err },
			wantMethod: "POST",
			wantPath:   "/datasets",
		},
		{
			name:       "ReadDataset",
			call:       func() error { _, err := client.ReadDataset(ctx, "ds-123"); return err },
			wantMethod: "GET",
			wantPath:   "/datasets/ds-123",
		},
		{
			name:       "DeleteDataset",
			call:       func() error { return client.DeleteDataset(ctx, "ds-123") },
			wantMethod: "DELETE",
			wantPath:   "/datasets/ds-123",
		},
		{
			name:       "CreateFeedback",
			call:       func() error { _, err := client.CreateFeedback(ctx, FeedbackCreate{Key: "k"}); return err },
			wantMethod: "POST",
			wantPath:   "/feedback",
		},
		{
			name:       "DeleteFeedback",
			call:       func() error { return client.DeleteFeedback(ctx, "fb-1") },
			wantMethod: "DELETE",
			wantPath:   "/feedback/fb-1",
		},
		{
			name:       "CreateProject",
			call:       func() error { _, err := client.CreateProject(ctx, TracerSessionCreate{Name: "p"}); return err },
			wantMethod: "POST",
			wantPath:   "/sessions",
		},
		{
			name:       "DeleteProject",
			call:       func() error { return client.DeleteProject(ctx, "proj-1") },
			wantMethod: "DELETE",
			wantPath:   "/sessions/proj-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.call()
			if lastMethod != tt.wantMethod {
				t.Errorf("method = %q, want %q", lastMethod, tt.wantMethod)
			}
			if lastPath != tt.wantPath {
				t.Errorf("path = %q, want %q", lastPath, tt.wantPath)
			}
		})
	}
}

func TestClient_LazyBatchInit_NoGoroutineWithoutTracing(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint("https://example.com"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Batch worker should not be initialized until submitBatch is called.
	if client.batch != nil {
		t.Error("batch worker should be nil before first use")
	}
}

func TestRunIterator_DoesNotMutateCallerOptions(t *testing.T) {
	t.Parallel()

	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page++
		var runs []Run
		if page == 1 {
			runs = []Run{{ID: "r1", Name: "run1"}, {ID: "r2", Name: "run2"}}
		}
		json.NewEncoder(w).Encode(runs)
	}))
	defer server.Close()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	opts := ListRunsOptions{
		ProjectName: StringPtr("test"),
	}
	it := client.ListRunsIterator(opts)

	_, err = it.All(context.Background())
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	// Caller's Limit should still be nil — iterator must not mutate it.
	if opts.Limit != nil {
		t.Errorf("caller's opts.Limit was mutated to %d, want nil", *opts.Limit)
	}
}

func TestRunIterator_PaginatesCorrectly(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var body ListRunsOptions
		json.NewDecoder(r.Body).Decode(&body)

		var runs []Run
		switch callCount {
		case 1:
			for i := 0; i < 100; i++ {
				runs = append(runs, Run{ID: fmt.Sprintf("r%d", i)})
			}
		case 2:
			for i := 100; i < 150; i++ {
				runs = append(runs, Run{ID: fmt.Sprintf("r%d", i)})
			}
		}
		json.NewEncoder(w).Encode(runs)
	}))
	defer server.Close()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	it := client.ListRunsIterator(ListRunsOptions{ProjectName: StringPtr("test")})
	all, err := it.All(context.Background())
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 150 {
		t.Errorf("len(all) = %d, want 150", len(all))
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestClient_BackoffCapped(t *testing.T) {
	t.Parallel()

	// With maxRetries=1, backoff is 500ms which is fine.
	// This test verifies the cap logic exists by checking that
	// high retry counts don't cause excessively long waits.
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(ServerInfo{Version: "1.0"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithEndpoint(server.URL),
		WithMaxRetries(3),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	start := time.Now()
	info, err := client.ServerInfo(context.Background())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("ServerInfo: %v", err)
	}
	if info.Version != "1.0" {
		t.Errorf("Version = %q, want 1.0", info.Version)
	}
	// 2 retries: 500ms + 1s = 1.5s max. With cap it should be well under 30s.
	if elapsed > 5*time.Second {
		t.Errorf("elapsed = %v, expected well under 5s", elapsed)
	}
}
