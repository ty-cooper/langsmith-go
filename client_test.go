package langsmith

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient(WithAPIKey(""))
	if err == nil {
		t.Error("expected error when API key is empty")
	}
}

func TestNewClientWithOptions(t *testing.T) {
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

	if client.Endpoint() != "https://custom.example.com" {
		t.Errorf("endpoint mismatch: %s", client.Endpoint())
	}
	if client.Project() != "my-project" {
		t.Errorf("project mismatch: %s", client.Project())
	}
}

func TestClientServerInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Error("missing API key header")
		}
		json.NewEncoder(w).Encode(ServerInfo{Version: "0.6.0"})
	}))
	defer server.Close()

	client, err := NewClient(
		WithAPIKey("test-key"),
		WithEndpoint(server.URL),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	info, err := client.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Version != "0.6.0" {
		t.Errorf("version mismatch: %s", info.Version)
	}
}

func TestClientRetryOnServerError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
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
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	info, err := client.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if info.Version != "1.0" {
		t.Errorf("version mismatch: %s", info.Version)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClientNoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
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
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	_, err = client.ServerInfo(context.Background())
	if err == nil {
		t.Error("expected error for 404")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for 4xx, got %d", attempts)
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "not found" {
		t.Errorf("expected 'not found', got %q", apiErr.Message)
	}
}
