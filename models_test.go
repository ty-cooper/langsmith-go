package langsmith

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRun_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Millisecond)
	errMsg := "something went wrong"
	run := Run{
		ID:        "test-id",
		Name:      "test-run",
		RunType:   RunTypeLLM,
		StartTime: now,
		Error:     &errMsg,
		Inputs:    map[string]any{"prompt": "hello"},
		Outputs:   map[string]any{"response": "world"},
		Tags:      []string{"test", "unit"},
		Metadata:  map[string]any{"version": "1.0"},
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Run
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != run.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, run.ID)
	}
	if decoded.RunType != RunTypeLLM {
		t.Errorf("RunType = %q, want %q", decoded.RunType, RunTypeLLM)
	}
	if decoded.Error == nil || *decoded.Error != errMsg {
		t.Errorf("Error = %v, want %q", decoded.Error, errMsg)
	}
	if len(decoded.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(decoded.Tags))
	}
}

func TestDataset_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	desc := "test dataset"
	ds := Dataset{
		ID:          "ds-1",
		Name:        "my-dataset",
		Description: &desc,
		DataType:    DataTypeKV,
		CreatedAt:   time.Now().UTC().Truncate(time.Millisecond),
	}

	data, err := json.Marshal(ds)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Dataset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Name != "my-dataset" {
		t.Errorf("Name = %q, want %q", decoded.Name, "my-dataset")
	}
	if decoded.DataType != DataTypeKV {
		t.Errorf("DataType = %q, want %q", decoded.DataType, DataTypeKV)
	}
}

func TestFeedback_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	score := 0.95
	runID := "run-123"
	fb := Feedback{
		ID:    "fb-1",
		RunID: &runID,
		Key:   "accuracy",
		Score: &score,
	}

	data, err := json.Marshal(fb)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Feedback
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Key != "accuracy" {
		t.Errorf("Key = %q, want %q", decoded.Key, "accuracy")
	}
	if decoded.Score == nil || *decoded.Score != 0.95 {
		t.Errorf("Score = %v, want 0.95", decoded.Score)
	}
}

func TestPtrHelpers(t *testing.T) {
	t.Parallel()

	if got := *StringPtr("hello"); got != "hello" {
		t.Errorf("StringPtr = %q", got)
	}
	if got := *IntPtr(42); got != 42 {
		t.Errorf("IntPtr = %d", got)
	}
	if got := *Float64Ptr(3.14); got != 3.14 {
		t.Errorf("Float64Ptr = %f", got)
	}
	if got := *BoolPtr(true); got != true {
		t.Errorf("BoolPtr = %v", got)
	}
	now := time.Now()
	if got := TimePtr(now); !got.Equal(now) {
		t.Errorf("TimePtr = %v, want %v", got, now)
	}
}
