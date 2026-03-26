package langsmith

import (
	"os"
	"strings"
)

const (
	defaultEndpoint = "https://api.smith.langchain.com"
	defaultProject  = "default"
)

// GetAPIKey returns the LangSmith API key from environment variables.
// It checks LANGCHAIN_API_KEY first, then LANGSMITH_API_KEY.
func GetAPIKey() string {
	if key := os.Getenv("LANGCHAIN_API_KEY"); key != "" {
		return key
	}
	return os.Getenv("LANGSMITH_API_KEY")
}

// GetEndpoint returns the LangSmith API endpoint from environment variables.
// It checks LANGCHAIN_ENDPOINT first, then LANGSMITH_ENDPOINT, defaulting to https://api.smith.langchain.com.
func GetEndpoint() string {
	if endpoint := os.Getenv("LANGCHAIN_ENDPOINT"); endpoint != "" {
		return strings.TrimRight(endpoint, "/")
	}
	if endpoint := os.Getenv("LANGSMITH_ENDPOINT"); endpoint != "" {
		return strings.TrimRight(endpoint, "/")
	}
	return defaultEndpoint
}

// GetProject returns the LangSmith project name from environment variables.
// It checks LANGCHAIN_PROJECT first, then LANGSMITH_PROJECT, defaulting to "default".
func GetProject() string {
	if project := os.Getenv("LANGCHAIN_PROJECT"); project != "" {
		return project
	}
	if project := os.Getenv("LANGSMITH_PROJECT"); project != "" {
		return project
	}
	return defaultProject
}

// TracingEnabled returns whether tracing is enabled via environment variables.
func TracingEnabled() bool {
	v := os.Getenv("LANGCHAIN_TRACING_V2")
	if v == "" {
		v = os.Getenv("LANGSMITH_TRACING")
	}
	return strings.EqualFold(v, "true") || v == "1"
}
