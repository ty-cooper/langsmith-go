package langsmith

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunTreeFromContext_NilOnEmpty(t *testing.T) {
	t.Parallel()

	if got := RunTreeFromContext(context.Background()); got != nil {
		t.Error("expected nil from empty context")
	}
}

func TestRunTree_ContextRoundTrip(t *testing.T) {
	t.Parallel()

	rt := NewRunTree("test", RunTypeChain)
	ctx := ContextWithRunTree(context.Background(), rt)
	got := RunTreeFromContext(ctx)

	if got == nil {
		t.Fatal("expected RunTree from context")
	}
	if got.ID != rt.ID {
		t.Errorf("ID = %q, want %q", got.ID, rt.ID)
	}
}

func TestRunTree_CreateChild_InheritsTrace(t *testing.T) {
	t.Parallel()

	parent := NewRunTree("parent", RunTypeChain)
	child := parent.CreateChild("child", RunTypeLLM)

	if child.ParentRunID == nil || *child.ParentRunID != parent.ID {
		t.Error("child should reference parent ID")
	}
	if child.TraceID != parent.TraceID {
		t.Error("child should inherit trace ID")
	}
	if child.DottedOrder == parent.DottedOrder {
		t.Error("child should have extended dotted order")
	}
	if got := len(parent.Children()); got != 1 {
		t.Errorf("len(Children()) = %d, want 1", got)
	}
}

func TestRunTree_End_SetsOutputsAndTime(t *testing.T) {
	t.Parallel()

	rt := NewRunTree("test", RunTypeChain)
	rt.End(WithEndOutputs(map[string]any{"result": "done"}))

	if rt.EndTime == nil {
		t.Fatal("EndTime should be set")
	}
	if rt.Outputs["result"] != "done" {
		t.Errorf("Outputs[result] = %v, want done", rt.Outputs["result"])
	}
}

func TestRunTree_End_SetsError(t *testing.T) {
	t.Parallel()

	rt := NewRunTree("test", RunTypeChain)
	rt.End(WithEndError("something failed"))

	if rt.Error == nil || *rt.Error != "something failed" {
		t.Errorf("Error = %v, want %q", rt.Error, "something failed")
	}
}

func TestRunTree_HeaderRoundTrip(t *testing.T) {
	t.Parallel()

	rt := NewRunTree("test", RunTypeChain, WithRunTreeSessionName("my-project"))
	headers := rt.ToHeaders()

	if headers.Get(HeaderParentID) != rt.ID {
		t.Errorf("parent ID header = %q, want %q", headers.Get(HeaderParentID), rt.ID)
	}
	if headers.Get(HeaderBaggage) == "" {
		t.Fatal("expected non-empty baggage header")
	}

	restored := RunTreeFromHeaders(headers, nil)
	if restored == nil {
		t.Fatal("expected restored RunTree")
	}
	if restored.ID != rt.ID {
		t.Errorf("restored ID = %q, want %q", restored.ID, rt.ID)
	}
	if restored.TraceID != rt.TraceID {
		t.Errorf("restored TraceID = %q, want %q", restored.TraceID, rt.TraceID)
	}
	if restored.SessionName != "my-project" {
		t.Errorf("restored SessionName = %q, want my-project", restored.SessionName)
	}
}

func TestRunTreeFromHeaders_EmptyReturnsNil(t *testing.T) {
	t.Parallel()

	if got := RunTreeFromHeaders(http.Header{}, nil); got != nil {
		t.Error("expected nil for empty headers")
	}
}

func TestRunTree_AddMetadata_MergesKeys(t *testing.T) {
	t.Parallel()

	rt := NewRunTree("test", RunTypeChain)
	rt.AddMetadata(map[string]any{"a": 1})
	rt.AddMetadata(map[string]any{"b": 2})

	if rt.Metadata["a"] != 1 || rt.Metadata["b"] != 2 {
		t.Errorf("Metadata = %v, want a=1, b=2", rt.Metadata)
	}
}

func TestRunTree_AddTags_Appends(t *testing.T) {
	t.Parallel()

	rt := NewRunTree("test", RunTypeChain)
	rt.AddTags("a", "b")
	rt.AddTags("c")

	if got := len(rt.Tags); got != 3 {
		t.Errorf("len(Tags) = %d, want 3", got)
	}
}

func TestTrace_CreatesNestedRun(t *testing.T) {
	t.Parallel()

	root := NewRunTree("root", RunTypeChain)
	ctx := ContextWithRunTree(context.Background(), root)

	err := Trace(ctx, "child-step", RunTypeLLM, func(ctx context.Context) error {
		rt := RunTreeFromContext(ctx)
		if rt == nil {
			t.Error("expected RunTree in child context")
			return nil
		}
		if rt.ParentRunID == nil || *rt.ParentRunID != root.ID {
			t.Errorf("child parent = %v, want %q", rt.ParentRunID, root.ID)
		}
		return nil
	})
	if err != nil {
		t.Errorf("Trace returned error: %v", err)
	}
}

func TestTraceFunc_ReturnsTypedValue(t *testing.T) {
	t.Parallel()

	root := NewRunTree("root", RunTypeChain)
	ctx := ContextWithRunTree(context.Background(), root)

	result, err := TraceFunc(ctx, "compute", RunTypeChain, func(ctx context.Context) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("TraceFunc: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
}

func TestGetRunURL_NoTrailingQuestionMark(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithAPIKey("key"), WithEndpoint("https://api.smith.langchain.com"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	got := client.GetRunURL("run-123")
	if got[len(got)-1] == '?' {
		t.Errorf("URL ends with '?': %q", got)
	}

	gotWithProject := client.GetRunURL("run-123", WithRunURLProjectID("proj-1"))
	if gotWithProject[len(gotWithProject)-1] == '?' {
		t.Errorf("URL with project ends with '?': %q", gotWithProject)
	}
}

func TestTracingMiddleware_CapturesStatusCode(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint(backend.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	var capturedRT *RunTree
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRT = RunTreeFromContext(r.Context())
		w.WriteHeader(http.StatusCreated)
	})

	handler := TracingMiddleware(client, "test-project")(inner)
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if capturedRT == nil {
		t.Fatal("expected RunTree in request context")
	}
	if capturedRT.Name != "GET /api/test" {
		t.Errorf("Name = %q, want %q", capturedRT.Name, "GET /api/test")
	}
}

func TestTracingMiddleware_PanicRecovery(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint(backend.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	var capturedRT *RunTree
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRT = RunTreeFromContext(r.Context())
		panic("test panic")
	})

	handler := TracingMiddleware(client, "test-project")(inner)
	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to be re-raised")
		}
		if capturedRT == nil {
			t.Fatal("expected RunTree to be captured")
		}
		if capturedRT.Error == nil {
			t.Fatal("expected Error to be set on panic")
		}
		if capturedRT.EndTime == nil {
			t.Fatal("expected EndTime to be set on panic")
		}
	}()

	handler.ServeHTTP(rec, req)
}

func TestTracingMiddleware_InheritsParentFromHeaders(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	client, err := NewClient(WithAPIKey("test-key"), WithEndpoint(backend.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	parent := NewRunTree("parent", RunTypeChain, WithRunTreeClient(client), WithRunTreeSessionName("upstream"))
	parentHeaders := parent.ToHeaders()

	var capturedRT *RunTree
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRT = RunTreeFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := TracingMiddleware(client, "test-project")(inner)
	req := httptest.NewRequest("POST", "/api/data", nil)
	for k, v := range parentHeaders {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedRT == nil {
		t.Fatal("expected RunTree")
	}
	if capturedRT.ParentRunID == nil || *capturedRT.ParentRunID != parent.ID {
		t.Errorf("ParentRunID = %v, want %q", capturedRT.ParentRunID, parent.ID)
	}
	if capturedRT.TraceID != parent.TraceID {
		t.Errorf("TraceID = %q, want %q", capturedRT.TraceID, parent.TraceID)
	}
}
