package langsmith

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRunSerializationRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	errMsg := "something went wrong"
	run := Run{
		ID:        "test-id",
		Name:      "test-run",
		RunType:   RunTypeLLM,
		StartTime: now,
		Error:     &errMsg,
		Inputs:    map[string]interface{}{"prompt": "hello"},
		Outputs:   map[string]interface{}{"response": "world"},
		Tags:      []string{"test", "unit"},
		Metadata:  map[string]interface{}{"version": "1.0"},
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Run
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != run.ID {
		t.Errorf("ID mismatch: %s vs %s", decoded.ID, run.ID)
	}
	if decoded.Name != run.Name {
		t.Errorf("Name mismatch: %s vs %s", decoded.Name, run.Name)
	}
	if decoded.RunType != RunTypeLLM {
		t.Errorf("RunType mismatch: %s", decoded.RunType)
	}
	if decoded.Error == nil || *decoded.Error != errMsg {
		t.Error("Error mismatch")
	}
	if len(decoded.Tags) != 2 {
		t.Errorf("Tags length mismatch: %d", len(decoded.Tags))
	}
}

func TestDatasetSerialization(t *testing.T) {
	desc := "test dataset"
	ds := Dataset{
		ID:          "ds-1",
		Name:        "my-dataset",
		Description: &desc,
		DataType:    DataTypeKV,
		CreatedAt:   time.Now().UTC(),
	}

	data, err := json.Marshal(ds)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Dataset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Name != "my-dataset" {
		t.Errorf("Name mismatch: %s", decoded.Name)
	}
	if decoded.DataType != DataTypeKV {
		t.Errorf("DataType mismatch: %s", decoded.DataType)
	}
}

func TestFeedbackSerialization(t *testing.T) {
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
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Feedback
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Key != "accuracy" {
		t.Errorf("Key mismatch: %s", decoded.Key)
	}
	if decoded.Score == nil || *decoded.Score != 0.95 {
		t.Error("Score mismatch")
	}
}

func TestHelperPtrFunctions(t *testing.T) {
	s := StringPtr("hello")
	if *s != "hello" {
		t.Error("StringPtr failed")
	}

	i := IntPtr(42)
	if *i != 42 {
		t.Error("IntPtr failed")
	}

	f := Float64Ptr(3.14)
	if *f != 3.14 {
		t.Error("Float64Ptr failed")
	}

	b := BoolPtr(true)
	if *b != true {
		t.Error("BoolPtr failed")
	}

	now := time.Now()
	tp := TimePtr(now)
	if !tp.Equal(now) {
		t.Error("TimePtr failed")
	}
}
