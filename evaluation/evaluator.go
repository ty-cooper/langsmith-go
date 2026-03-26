package evaluation

import (
	langsmith "github.com/tcoooper/langsmith-go"
)

// RunEvaluator evaluates a run against its reference example.
type RunEvaluator interface {
	EvaluateRun(run langsmith.Run, example *langsmith.Example) (*EvaluationResult, error)
}

// RunEvaluatorFunc is an adapter to allow the use of ordinary functions as RunEvaluators.
type RunEvaluatorFunc func(run langsmith.Run, example *langsmith.Example) (*EvaluationResult, error)

// EvaluateRun calls the wrapped function.
func (f RunEvaluatorFunc) EvaluateRun(run langsmith.Run, example *langsmith.Example) (*EvaluationResult, error) {
	return f(run, example)
}

// SummaryEvaluator evaluates a collection of runs/examples together.
type SummaryEvaluator interface {
	EvaluateSummary(runs []RunExample) ([]SummaryEvaluationResult, error)
}

// SummaryEvaluatorFunc is an adapter to allow the use of ordinary functions as SummaryEvaluators.
type SummaryEvaluatorFunc func(runs []RunExample) ([]SummaryEvaluationResult, error)

// EvaluateSummary calls the wrapped function.
func (f SummaryEvaluatorFunc) EvaluateSummary(runs []RunExample) ([]SummaryEvaluationResult, error) {
	return f(runs)
}

// StringEvaluator evaluates by comparing string predictions against string references.
type StringEvaluator struct {
	Key        string
	EvalFunc   func(prediction, reference string) (*EvaluationResult, error)
	InputKey   string
	PredictionKey string
	ReferenceKey  string
}

// EvaluateRun implements RunEvaluator for StringEvaluator.
func (e *StringEvaluator) EvaluateRun(run langsmith.Run, example *langsmith.Example) (*EvaluationResult, error) {
	predKey := e.PredictionKey
	if predKey == "" {
		predKey = "output"
	}
	refKey := e.ReferenceKey
	if refKey == "" {
		refKey = "output"
	}

	prediction := ""
	if run.Outputs != nil {
		if v, ok := run.Outputs[predKey]; ok {
			if s, ok := v.(string); ok {
				prediction = s
			}
		}
	}

	reference := ""
	if example != nil && example.Outputs != nil {
		if v, ok := example.Outputs[refKey]; ok {
			if s, ok := v.(string); ok {
				reference = s
			}
		}
	}

	result, err := e.EvalFunc(prediction, reference)
	if err != nil {
		return nil, err
	}
	if result.Key == "" {
		result.Key = e.Key
	}
	return result, nil
}

// ExactMatch returns a RunEvaluator that checks if the prediction exactly matches the reference.
func ExactMatch(key string) RunEvaluator {
	return &StringEvaluator{
		Key: key,
		EvalFunc: func(prediction, reference string) (*EvaluationResult, error) {
			match := prediction == reference
			score := 0.0
			if match {
				score = 1.0
			}
			return &EvaluationResult{
				Key:   key,
				Score: &score,
			}, nil
		},
	}
}
