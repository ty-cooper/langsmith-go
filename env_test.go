package langsmith

import "testing"

// Env tests cannot use t.Parallel because t.Setenv prevents it.

func TestGetAPIKey_Precedence(t *testing.T) {
	tests := []struct {
		name      string
		langchain string
		langsmith string
		want      string
	}{
		{name: "both_empty", langchain: "", langsmith: "", want: ""},
		{name: "langsmith_only", langchain: "", langsmith: "ls-key", want: "ls-key"},
		{name: "langchain_takes_precedence", langchain: "lc-key", langsmith: "ls-key", want: "lc-key"},
		{name: "langchain_only", langchain: "lc-key", langsmith: "", want: "lc-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LANGCHAIN_API_KEY", tt.langchain)
			t.Setenv("LANGSMITH_API_KEY", tt.langsmith)

			if got := GetAPIKey(); got != tt.want {
				t.Errorf("GetAPIKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetEndpoint_Precedence(t *testing.T) {
	tests := []struct {
		name      string
		langchain string
		langsmith string
		want      string
	}{
		{name: "default", langchain: "", langsmith: "", want: defaultEndpoint},
		{name: "strips_trailing_slash", langchain: "", langsmith: "https://custom.example.com/", want: "https://custom.example.com"},
		{name: "langchain_takes_precedence", langchain: "https://lc.example.com", langsmith: "https://ls.example.com", want: "https://lc.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LANGCHAIN_ENDPOINT", tt.langchain)
			t.Setenv("LANGSMITH_ENDPOINT", tt.langsmith)

			if got := GetEndpoint(); got != tt.want {
				t.Errorf("GetEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetProject_Precedence(t *testing.T) {
	tests := []struct {
		name      string
		langchain string
		langsmith string
		want      string
	}{
		{name: "default", langchain: "", langsmith: "", want: defaultProject},
		{name: "langsmith_only", langchain: "", langsmith: "my-project", want: "my-project"},
		{name: "langchain_takes_precedence", langchain: "lc-proj", langsmith: "ls-proj", want: "lc-proj"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LANGCHAIN_PROJECT", tt.langchain)
			t.Setenv("LANGSMITH_PROJECT", tt.langsmith)

			if got := GetProject(); got != tt.want {
				t.Errorf("GetProject() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTracingEnabled_Values(t *testing.T) {
	tests := []struct {
		name    string
		v2      string
		tracing string
		want    bool
	}{
		{name: "both_unset", v2: "", tracing: "", want: false},
		{name: "langsmith_true", v2: "", tracing: "true", want: true},
		{name: "langsmith_TRUE", v2: "", tracing: "TRUE", want: true},
		{name: "langchain_1", v2: "1", tracing: "", want: true},
		{name: "langchain_false", v2: "false", tracing: "", want: false},
		{name: "langchain_takes_precedence", v2: "true", tracing: "false", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LANGCHAIN_TRACING_V2", tt.v2)
			t.Setenv("LANGSMITH_TRACING", tt.tracing)

			if got := TracingEnabled(); got != tt.want {
				t.Errorf("TracingEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
