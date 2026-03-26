package evaluation

import (
	"testing"

	langsmith "github.com/tcoooper/langsmith-go"
)

func TestExactMatch_MatchingStrings_ReturnsScore1(t *testing.T) {
	t.Parallel()

	evaluator := ExactMatch("correctness")
	run := langsmith.Run{
		Outputs: map[string]any{"output": "hello"},
	}
	example := &langsmith.Example{
		Outputs: map[string]any{"output": "hello"},
	}

	result, err := evaluator.EvaluateRun(run, example)
	if err != nil {
		t.Fatalf("EvaluateRun: %v", err)
	}
	if result.Key != "correctness" {
		t.Errorf("Key = %q, want correctness", result.Key)
	}
	if result.Score == nil || *result.Score != 1.0 {
		t.Errorf("Score = %v, want 1.0", result.Score)
	}
}

func TestExactMatch_DifferentStrings_ReturnsScore0(t *testing.T) {
	t.Parallel()

	evaluator := ExactMatch("correctness")
	run := langsmith.Run{
		Outputs: map[string]any{"output": "hello"},
	}
	example := &langsmith.Example{
		Outputs: map[string]any{"output": "world"},
	}

	result, err := evaluator.EvaluateRun(run, example)
	if err != nil {
		t.Fatalf("EvaluateRun: %v", err)
	}
	if result.Score == nil || *result.Score != 0.0 {
		t.Errorf("Score = %v, want 0.0", result.Score)
	}
}

func TestRunEvaluatorFunc_CalledCorrectly(t *testing.T) {
	t.Parallel()

	called := false
	fn := RunEvaluatorFunc(func(run langsmith.Run, example *langsmith.Example) (*EvaluationResult, error) {
		called = true
		return &EvaluationResult{Key: "test", Score: langsmith.Float64Ptr(0.5)}, nil
	})

	result, err := fn.EvaluateRun(langsmith.Run{}, nil)
	if err != nil {
		t.Fatalf("EvaluateRun: %v", err)
	}
	if !called {
		t.Error("function was not called")
	}
	if result.Key != "test" {
		t.Errorf("Key = %q, want test", result.Key)
	}
}

func TestStringEvaluator_CustomKeys(t *testing.T) {
	t.Parallel()

	eval := &StringEvaluator{
		Key:           "custom",
		PredictionKey: "answer",
		ReferenceKey:  "expected",
		EvalFunc: func(prediction, reference string) (*EvaluationResult, error) {
			score := 0.0
			if prediction == reference {
				score = 1.0
			}
			return &EvaluationResult{Score: &score}, nil
		},
	}

	run := langsmith.Run{Outputs: map[string]any{"answer": "42"}}
	example := &langsmith.Example{Outputs: map[string]any{"expected": "42"}}

	result, err := eval.EvaluateRun(run, example)
	if err != nil {
		t.Fatalf("EvaluateRun: %v", err)
	}
	if result.Key != "custom" {
		t.Errorf("Key = %q, want custom", result.Key)
	}
	if result.Score == nil || *result.Score != 1.0 {
		t.Errorf("Score = %v, want 1.0", result.Score)
	}
}
