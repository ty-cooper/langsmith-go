package langsmith

import (
	"context"
	"net/http"
	"testing"
)

func TestRunTreeContext(t *testing.T) {
	rt := NewRunTree("test", RunTypeChain)

	ctx := ContextWithRunTree(context.Background(), rt)
	got := RunTreeFromContext(ctx)
	if got == nil {
		t.Fatal("expected RunTree from context")
	}
	if got.ID != rt.ID {
		t.Errorf("ID mismatch: %s vs %s", got.ID, rt.ID)
	}
}

func TestRunTreeFromContextNil(t *testing.T) {
	got := RunTreeFromContext(context.Background())
	if got != nil {
		t.Error("expected nil from empty context")
	}
}

func TestRunTreeCreateChild(t *testing.T) {
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
	if len(parent.Children()) != 1 {
		t.Errorf("expected 1 child, got %d", len(parent.Children()))
	}
}

func TestRunTreeEnd(t *testing.T) {
	rt := NewRunTree("test", RunTypeChain)
	rt.End(
		WithEndOutputs(map[string]interface{}{"result": "done"}),
	)
	if rt.EndTime == nil {
		t.Error("EndTime should be set after End()")
	}
	if rt.Outputs["result"] != "done" {
		t.Error("Outputs should be set")
	}
}

func TestRunTreeEndWithError(t *testing.T) {
	rt := NewRunTree("test", RunTypeChain)
	rt.End(WithEndError("something failed"))
	if rt.Error == nil || *rt.Error != "something failed" {
		t.Error("Error should be set")
	}
}

func TestRunTreeHeaders(t *testing.T) {
	rt := NewRunTree("test", RunTypeChain,
		WithRunTreeSessionName("my-project"),
	)

	headers := rt.ToHeaders()
	if headers.Get(HeaderParentID) != rt.ID {
		t.Error("parent ID header mismatch")
	}
	baggage := headers.Get(HeaderBaggage)
	if baggage == "" {
		t.Fatal("expected non-empty baggage")
	}

	// Round-trip through headers.
	httpHeaders := http.Header{}
	httpHeaders.Set(HeaderParentID, headers.Get(HeaderParentID))
	httpHeaders.Set(HeaderBaggage, headers.Get(HeaderBaggage))

	restored := RunTreeFromHeaders(httpHeaders, nil)
	if restored == nil {
		t.Fatal("expected restored RunTree")
	}
	if restored.ID != rt.ID {
		t.Errorf("ID mismatch: %s vs %s", restored.ID, rt.ID)
	}
	if restored.TraceID != rt.TraceID {
		t.Errorf("TraceID mismatch: %s vs %s", restored.TraceID, rt.TraceID)
	}
	if restored.SessionName != "my-project" {
		t.Errorf("SessionName mismatch: %s", restored.SessionName)
	}
}

func TestRunTreeFromHeadersEmpty(t *testing.T) {
	got := RunTreeFromHeaders(http.Header{}, nil)
	if got != nil {
		t.Error("expected nil for empty headers")
	}
}

func TestRunTreeMetadataAndTags(t *testing.T) {
	rt := NewRunTree("test", RunTypeChain)
	rt.AddMetadata(map[string]interface{}{"key": "value"})
	rt.AddTags("tag1", "tag2")
	rt.AddEvent(map[string]interface{}{"name": "event1"})

	if rt.Metadata["key"] != "value" {
		t.Error("metadata not set")
	}
	if len(rt.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(rt.Tags))
	}
	if len(rt.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(rt.Events))
	}
}

func TestTraceNested(t *testing.T) {
	// Test that Trace creates nested runs properly.
	root := NewRunTree("root", RunTypeChain)
	ctx := ContextWithRunTree(context.Background(), root)

	err := Trace(ctx, "child-step", RunTypeLLM, func(ctx context.Context) error {
		rt := RunTreeFromContext(ctx)
		if rt == nil {
			t.Error("expected RunTree in child context")
		}
		if rt.ParentRunID == nil || *rt.ParentRunID != root.ID {
			t.Error("child should reference root as parent")
		}
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTraceFuncGeneric(t *testing.T) {
	root := NewRunTree("root", RunTypeChain)
	ctx := ContextWithRunTree(context.Background(), root)

	result, err := TraceFunc(ctx, "compute", RunTypeChain, func(ctx context.Context) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}
