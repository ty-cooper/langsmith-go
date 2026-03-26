package evaluation

import (
	"context"
	"fmt"
	"sync"
	"time"

	langsmith "github.com/ty-cooper/langsmith-go"
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
	Metadata map[string]any
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
	dataset, err := client.ReadDataset(ctx, datasetID)
	if err != nil {
		return nil, fmt.Errorf("evaluate: read dataset: %w", err)
	}

	examples, err := listAllExamples(ctx, client, datasetID)
	if err != nil {
		return nil, fmt.Errorf("evaluate: list examples: %w", err)
	}
	if len(examples) == 0 {
		return nil, fmt.Errorf("evaluate: dataset %q has no examples", dataset.Name)
	}

	experimentName := fmt.Sprintf("%s-%d", dataset.Name, time.Now().Unix())
	if opts.ExperimentPrefix != "" {
		experimentName = fmt.Sprintf("%s-%s-%d", opts.ExperimentPrefix, dataset.Name, time.Now().Unix())
	}

	project, err := client.CreateProject(ctx, langsmith.TracerSessionCreate{
		Name:               experimentName,
		ReferenceDatasetID: &datasetID,
		Extra:              map[string]any{"metadata": opts.Metadata},
	})
	if err != nil {
		return nil, fmt.Errorf("evaluate: create experiment project: %w", err)
	}

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

			// Acquire semaphore, but bail if the context is cancelled
			// to avoid leaking blocked goroutines.
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[idx] = resultItem{idx: idx, example: ex, err: ctx.Err()}
				return
			}
			defer func() { <-sem }()

			startTime := time.Now()
			outputs, runErr := target(ctx, ex.Inputs)
			endTime := time.Now()

			var errStr *string
			if runErr != nil {
				s := runErr.Error()
				errStr = &s
			}

			run := langsmith.RunCreate{
				ID:                 langsmith.NewID(),
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

	var experimentRows []ExperimentResultRow
	var runErrors []error
	for _, res := range results {
		if res.err != nil {
			runErrors = append(runErrors, res.err)
			continue
		}

		var evalResults []EvaluationResult
		for _, evaluator := range opts.Evaluators {
			evalResult, evalErr := evaluator.EvaluateRun(res.run, &res.example)
			if evalErr != nil {
				runErrors = append(runErrors, fmt.Errorf("evaluator on run %s: %w", res.run.ID, evalErr))
				continue
			}
			if evalResult != nil {
				_, fbErr := client.CreateFeedback(ctx, langsmith.FeedbackCreate{
					RunID:   &res.run.ID,
					Key:     evalResult.Key,
					Score:   evalResult.Score,
					Value:   evalResult.Value,
					Comment: evalResult.Comment,
				})
				if fbErr != nil {
					runErrors = append(runErrors, fmt.Errorf("create feedback for run %s: %w", res.run.ID, fbErr))
				}
				evalResults = append(evalResults, *evalResult)
			}
		}

		experimentRows = append(experimentRows, ExperimentResultRow{
			Run:               res.run,
			Example:           res.example,
			EvaluationResults: evalResults,
		})
	}

	var summaryResults []SummaryEvaluationResult
	if len(opts.SummaryEvaluators) > 0 {
		var runExamples []RunExample
		for _, row := range experimentRows {
			runExamples = append(runExamples, RunExample{Run: row.Run, Example: row.Example})
		}
		for _, summaryEval := range opts.SummaryEvaluators {
			results, evalErr := summaryEval.EvaluateSummary(runExamples)
			if evalErr != nil {
				runErrors = append(runErrors, fmt.Errorf("summary evaluator: %w", evalErr))
				continue
			}
			for _, sr := range results {
				_, fbErr := client.CreateFeedback(ctx, langsmith.FeedbackCreate{
					Key:   sr.Key,
					Score: sr.Score,
					Value: sr.Value,
				})
				if fbErr != nil {
					runErrors = append(runErrors, fmt.Errorf("create summary feedback %q: %w", sr.Key, fbErr))
				}
			}
			summaryResults = append(summaryResults, results...)
		}
	}

	experimentResult := &ExperimentResults{
		ExperimentName: experimentName,
		ProjectID:      project.ID,
		DatasetID:      dataset.ID,
		DatasetName:    dataset.Name,
		CreatedAt:      time.Now(),
		Results:        experimentRows,
		SummaryResults: summaryResults,
		Metadata:       opts.Metadata,
	}

	if len(runErrors) > 0 {
		return experimentResult, fmt.Errorf("evaluate: %d error(s), first: %w", len(runErrors), runErrors[0])
	}

	return experimentResult, nil
}

// EvaluateExisting evaluates an existing experiment's runs.
func EvaluateExisting(
	ctx context.Context,
	client *langsmith.Client,
	projectName string,
	opts EvaluateOptions,
) (*ExperimentResults, error) {
	project, err := client.ReadProjectByName(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("evaluate existing: find project %q: %w", projectName, err)
	}

	runs, err := client.ListRuns(ctx, langsmith.ListRunsOptions{
		ProjectID: &project.ID,
		IsRoot:    langsmith.BoolPtr(true),
	})
	if err != nil {
		return nil, fmt.Errorf("evaluate existing: list runs: %w", err)
	}

	var experimentRows []ExperimentResultRow
	var runErrors []error
	for _, run := range runs {
		var example *langsmith.Example
		if run.ReferenceExampleID != nil {
			ex, readErr := client.ReadExample(ctx, *run.ReferenceExampleID)
			if readErr == nil {
				example = ex
			}
		}

		var evalResults []EvaluationResult
		for _, evaluator := range opts.Evaluators {
			evalResult, evalErr := evaluator.EvaluateRun(run, example)
			if evalErr != nil {
				runErrors = append(runErrors, fmt.Errorf("evaluator on run %s: %w", run.ID, evalErr))
				continue
			}
			if evalResult != nil {
				_, fbErr := client.CreateFeedback(ctx, langsmith.FeedbackCreate{
					RunID:   &run.ID,
					Key:     evalResult.Key,
					Score:   evalResult.Score,
					Value:   evalResult.Value,
					Comment: evalResult.Comment,
				})
				if fbErr != nil {
					runErrors = append(runErrors, fmt.Errorf("create feedback for run %s: %w", run.ID, fbErr))
				}
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

	experimentResult := &ExperimentResults{
		ExperimentName: projectName,
		ProjectID:      project.ID,
		CreatedAt:      time.Now(),
		Results:        experimentRows,
		Metadata:       opts.Metadata,
	}

	if len(runErrors) > 0 {
		return experimentResult, fmt.Errorf("evaluate existing: %d error(s), first: %w", len(runErrors), runErrors[0])
	}

	return experimentResult, nil
}

// listAllExamples paginates through all examples for a dataset.
func listAllExamples(ctx context.Context, client *langsmith.Client, datasetID string) ([]langsmith.Example, error) {
	const pageSize = 100
	var all []langsmith.Example
	offset := 0

	for {
		page, err := client.ListExamples(ctx, langsmith.ListExamplesOptions{
			DatasetID: &datasetID,
			Limit:     langsmith.IntPtr(pageSize),
			Offset:    offset,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
		offset += len(page)
	}
	return all, nil
}
