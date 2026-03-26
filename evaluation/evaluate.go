package evaluation

import (
	"context"
	"fmt"
	"sync"
	"time"

	langsmith "github.com/tcoooper/langsmith-go"
	"github.com/tcoooper/langsmith-go/internal"
)

// EvaluateOptions configures an evaluation run.
type EvaluateOptions struct {
	// Evaluators are the evaluators to run on each example.
	Evaluators []RunEvaluator
	// SummaryEvaluators are run on the full set of results.
	SummaryEvaluators []SummaryEvaluator
	// ExperimentPrefix is prepended to the experiment name.
	ExperimentPrefix string
	// Metadata is attached to the experiment.
	Metadata map[string]interface{}
	// MaxConcurrency limits parallel target invocations. Default is 5.
	MaxConcurrency int
	// Description for the experiment.
	Description string
}

// Evaluate runs a target function against all examples in a dataset
// and evaluates the results.
func Evaluate(
	ctx context.Context,
	client *langsmith.Client,
	datasetID string,
	target TargetFunc,
	opts EvaluateOptions,
) (*ExperimentResults, error) {
	// Fetch dataset info.
	dataset, err := client.ReadDataset(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to read dataset: %w", err)
	}

	// Fetch all examples.
	examples, err := client.ListExamples(ctx, langsmith.ListExamplesOptions{
		DatasetID: &datasetID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list examples: %w", err)
	}

	if len(examples) == 0 {
		return nil, fmt.Errorf("dataset %q has no examples", dataset.Name)
	}

	// Create experiment project.
	experimentName := fmt.Sprintf("%s-%d", dataset.Name, time.Now().Unix())
	if opts.ExperimentPrefix != "" {
		experimentName = fmt.Sprintf("%s-%s-%d", opts.ExperimentPrefix, dataset.Name, time.Now().Unix())
	}

	project, err := client.CreateProject(ctx, langsmith.TracerSessionCreate{
		Name:               experimentName,
		ReferenceDatasetID: &datasetID,
		Extra: map[string]interface{}{
			"metadata": opts.Metadata,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create experiment project: %w", err)
	}

	// Run target on all examples with concurrency control.
	maxConc := opts.MaxConcurrency
	if maxConc <= 0 {
		maxConc = 5
	}
	sem := make(chan struct{}, maxConc)

	type resultItem struct {
		idx     int
		run     langsmith.Run
		example langsmith.Example
		err     error
	}
	results := make([]resultItem, len(examples))
	var wg sync.WaitGroup

	for i, example := range examples {
		wg.Add(1)
		go func(idx int, ex langsmith.Example) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			startTime := time.Now()
			outputs, runErr := target(ex.Inputs)
			endTime := time.Now()

			var errStr *string
			if runErr != nil {
				s := runErr.Error()
				errStr = &s
			}

			run := langsmith.RunCreate{
				ID:                 internal.UUID7(),
				Name:               "evaluation",
				RunType:            langsmith.RunTypeChain,
				StartTime:          startTime,
				EndTime:            &endTime,
				Inputs:             ex.Inputs,
				Outputs:            outputs,
				Error:              errStr,
				SessionID:          &project.ID,
				ReferenceExampleID: &ex.ID,
			}

			created, createErr := client.CreateRun(ctx, run)
			if createErr != nil {
				results[idx] = resultItem{idx: idx, example: ex, err: createErr}
				return
			}

			results[idx] = resultItem{
				idx:     idx,
				run:     *created,
				example: ex,
			}
		}(i, example)
	}
	wg.Wait()

	// Run evaluators on each result.
	var experimentRows []ExperimentResultRow
	for _, res := range results {
		if res.err != nil {
			continue
		}

		var evalResults []EvaluationResult
		for _, evaluator := range opts.Evaluators {
			evalResult, err := evaluator.EvaluateRun(res.run, &res.example)
			if err != nil {
				continue
			}
			if evalResult != nil {
				// Submit feedback.
				_, _ = client.CreateFeedback(ctx, langsmith.FeedbackCreate{
					RunID:   &res.run.ID,
					Key:     evalResult.Key,
					Score:   evalResult.Score,
					Value:   evalResult.Value,
					Comment: evalResult.Comment,
				})
				evalResults = append(evalResults, *evalResult)
			}
		}

		experimentRows = append(experimentRows, ExperimentResultRow{
			Run:               res.run,
			Example:           res.example,
			EvaluationResults: evalResults,
		})
	}

	// Run summary evaluators.
	if len(opts.SummaryEvaluators) > 0 {
		var runExamples []RunExample
		for _, row := range experimentRows {
			runExamples = append(runExamples, RunExample{
				Run:     row.Run,
				Example: row.Example,
			})
		}
		for _, summaryEval := range opts.SummaryEvaluators {
			_, _ = summaryEval.EvaluateSummary(runExamples)
		}
	}

	return &ExperimentResults{
		ExperimentName: experimentName,
		ProjectID:      project.ID,
		DatasetID:      dataset.ID,
		DatasetName:    dataset.Name,
		CreatedAt:      time.Now(),
		Results:        experimentRows,
		Metadata:       opts.Metadata,
	}, nil
}

// EvaluateExisting evaluates an existing experiment's runs.
func EvaluateExisting(
	ctx context.Context,
	client *langsmith.Client,
	projectName string,
	opts EvaluateOptions,
) (*ExperimentResults, error) {
	// Find the project.
	project, err := client.ReadProjectByName(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to find project %q: %w", projectName, err)
	}

	// Get runs from the project.
	runs, err := client.ListRuns(ctx, langsmith.ListRunsOptions{
		ProjectID: &project.ID,
		IsRoot:    langsmith.BoolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}

	// Build results.
	var experimentRows []ExperimentResultRow
	for _, run := range runs {
		var example *langsmith.Example
		if run.ReferenceExampleID != nil {
			ex, err := client.ReadExample(ctx, *run.ReferenceExampleID)
			if err == nil {
				example = ex
			}
		}

		var evalResults []EvaluationResult
		for _, evaluator := range opts.Evaluators {
			evalResult, err := evaluator.EvaluateRun(run, example)
			if err != nil {
				continue
			}
			if evalResult != nil {
				_, _ = client.CreateFeedback(ctx, langsmith.FeedbackCreate{
					RunID:   &run.ID,
					Key:     evalResult.Key,
					Score:   evalResult.Score,
					Value:   evalResult.Value,
					Comment: evalResult.Comment,
				})
				evalResults = append(evalResults, *evalResult)
			}
		}

		row := ExperimentResultRow{
			Run:               run,
			EvaluationResults: evalResults,
		}
		if example != nil {
			row.Example = *example
		}
		experimentRows = append(experimentRows, row)
	}

	return &ExperimentResults{
		ExperimentName: projectName,
		ProjectID:      project.ID,
		CreatedAt:      time.Now(),
		Results:        experimentRows,
		Metadata:       opts.Metadata,
	}, nil
}

