package evaluation

import (
	"time"

	langsmith "github.com/ty-cooper/langsmith-go"
)

// EvaluationResult represents the result of a single evaluator on a single example.
type EvaluationResult struct {
	Key            string         `json:"key"`
	Score          *float64       `json:"score,omitempty"`
	Value          any            `json:"value,omitempty"`
	Comment        *string        `json:"comment,omitempty"`
	Correction     any            `json:"correction,omitempty"`
	FeedbackConfig map[string]any `json:"feedback_config,omitempty"`
	SourceRunID    *string        `json:"source_run_id,omitempty"`
	TargetRunID    *string        `json:"target_run_id,omitempty"`
}

// EvaluationResults is a collection of evaluation results from a single evaluator invocation.
type EvaluationResults struct {
	Results []EvaluationResult `json:"results"`
}

// ExperimentResultRow represents a single row in experiment results.
type ExperimentResultRow struct {
	Run               langsmith.Run      `json:"run"`
	Example           langsmith.Example  `json:"example"`
	EvaluationResults []EvaluationResult `json:"evaluation_results"`
}

// ExperimentResults represents the full results of an evaluation experiment.
type ExperimentResults struct {
	ExperimentName string         `json:"experiment_name"`
	ProjectID      string         `json:"project_id,omitempty"`
	DatasetID      string         `json:"dataset_id"`
	DatasetName    string         `json:"dataset_name"`
	CreatedAt      time.Time      `json:"created_at"`
	Results        []ExperimentResultRow `json:"results"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// SummaryEvaluationResult represents an aggregate evaluation result.
type SummaryEvaluationResult struct {
	Key     string   `json:"key"`
	Score   *float64 `json:"score,omitempty"`
	Value   any      `json:"value,omitempty"`
	Comment *string  `json:"comment,omitempty"`
}

// TargetFunc is the function under evaluation. It takes example inputs
// and returns outputs.
type TargetFunc func(inputs map[string]any) (map[string]any, error)

// RunExample pairs a run with the example that produced it.
type RunExample struct {
	Run     langsmith.Run
	Example langsmith.Example
}
