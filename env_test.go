package langsmith

import (
	"os"
	"testing"
)

func TestGetAPIKey(t *testing.T) {
	// Clean up env after test.
	origLC := os.Getenv("LANGCHAIN_API_KEY")
	origLS := os.Getenv("LANGSMITH_API_KEY")
	defer func() {
		os.Setenv("LANGCHAIN_API_KEY", origLC)
		os.Setenv("LANGSMITH_API_KEY", origLS)
	}()

	os.Setenv("LANGCHAIN_API_KEY", "")
	os.Setenv("LANGSMITH_API_KEY", "")

	if got := GetAPIKey(); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	os.Setenv("LANGSMITH_API_KEY", "ls-key")
	if got := GetAPIKey(); got != "ls-key" {
		t.Errorf("expected ls-key, got %q", got)
	}

	os.Setenv("LANGCHAIN_API_KEY", "lc-key")
	if got := GetAPIKey(); got != "lc-key" {
		t.Errorf("expected lc-key (LANGCHAIN takes precedence), got %q", got)
	}
}

func TestGetEndpoint(t *testing.T) {
	origLC := os.Getenv("LANGCHAIN_ENDPOINT")
	origLS := os.Getenv("LANGSMITH_ENDPOINT")
	defer func() {
		os.Setenv("LANGCHAIN_ENDPOINT", origLC)
		os.Setenv("LANGSMITH_ENDPOINT", origLS)
	}()

	os.Setenv("LANGCHAIN_ENDPOINT", "")
	os.Setenv("LANGSMITH_ENDPOINT", "")

	if got := GetEndpoint(); got != defaultEndpoint {
		t.Errorf("expected default %q, got %q", defaultEndpoint, got)
	}

	os.Setenv("LANGSMITH_ENDPOINT", "https://custom.example.com/")
	if got := GetEndpoint(); got != "https://custom.example.com" {
		t.Errorf("expected trailing slash stripped, got %q", got)
	}

	os.Setenv("LANGCHAIN_ENDPOINT", "https://lc.example.com")
	if got := GetEndpoint(); got != "https://lc.example.com" {
		t.Errorf("expected LANGCHAIN to take precedence, got %q", got)
	}
}

func TestGetProject(t *testing.T) {
	origLC := os.Getenv("LANGCHAIN_PROJECT")
	origLS := os.Getenv("LANGSMITH_PROJECT")
	defer func() {
		os.Setenv("LANGCHAIN_PROJECT", origLC)
		os.Setenv("LANGSMITH_PROJECT", origLS)
	}()

	os.Setenv("LANGCHAIN_PROJECT", "")
	os.Setenv("LANGSMITH_PROJECT", "")

	if got := GetProject(); got != defaultProject {
		t.Errorf("expected default %q, got %q", defaultProject, got)
	}

	os.Setenv("LANGSMITH_PROJECT", "my-project")
	if got := GetProject(); got != "my-project" {
		t.Errorf("expected my-project, got %q", got)
	}
}

func TestTracingEnabled(t *testing.T) {
	origV2 := os.Getenv("LANGCHAIN_TRACING_V2")
	origLS := os.Getenv("LANGSMITH_TRACING")
	defer func() {
		os.Setenv("LANGCHAIN_TRACING_V2", origV2)
		os.Setenv("LANGSMITH_TRACING", origLS)
	}()

	os.Setenv("LANGCHAIN_TRACING_V2", "")
	os.Setenv("LANGSMITH_TRACING", "")
	if TracingEnabled() {
		t.Error("expected false when both unset")
	}

	os.Setenv("LANGSMITH_TRACING", "true")
	if !TracingEnabled() {
		t.Error("expected true for LANGSMITH_TRACING=true")
	}

	os.Setenv("LANGCHAIN_TRACING_V2", "1")
	if !TracingEnabled() {
		t.Error("expected true for LANGCHAIN_TRACING_V2=1")
	}
}
