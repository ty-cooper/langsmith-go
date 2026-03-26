# LangSmith Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/ty-cooper/langsmith-go.svg)](https://pkg.go.dev/github.com/ty-cooper/langsmith-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

This package provides a Go client for [LangSmith](https://smith.langchain.com/) — a platform for debugging, testing, evaluating, and monitoring LLM applications.

## Installation

```bash
go get github.com/ty-cooper/langsmith-go@v0.1.0
```

**Requirements:** Go 1.22+ | **Latest:** `v0.1.0`

## Quick Start

First, create an API key at [smith.langchain.com](https://smith.langchain.com/) and set it:

```bash
export LANGCHAIN_API_KEY="lsv2_pt_..."
export LANGCHAIN_TRACING_V2="true"
```

Then instrument your code:

```go
package main

import (
    "context"
    "fmt"
    "log"

    langsmith "github.com/ty-cooper/langsmith-go"
)

func main() {
    client, err := langsmith.NewClient()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Trace a function call
    err = langsmith.Trace(ctx, "my-chain", langsmith.RunTypeChain, func(ctx context.Context) error {
        rt := langsmith.RunTreeFromContext(ctx)
        rt.SetInputs(map[string]any{"question": "What is LangSmith?"})

        // Nest a child span
        answer, err := langsmith.TraceFunc(ctx, "llm-call", langsmith.RunTypeLLM,
            func(ctx context.Context) (string, error) {
                return "LangSmith is an observability platform for LLMs.", nil
            },
        )
        if err != nil {
            return err
        }

        rt.SetOutputs(map[string]any{"answer": answer})
        return nil
    }, langsmith.WithRunTreeClient(client))

    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Trace sent to LangSmith!")
}
```

## Configuration

The client reads configuration from environment variables by default:

| Variable | Description | Default |
|---|---|---|
| `LANGCHAIN_API_KEY` / `LANGSMITH_API_KEY` | API key for authentication | (required) |
| `LANGCHAIN_ENDPOINT` / `LANGSMITH_ENDPOINT` | API endpoint URL | `https://api.smith.langchain.com` |
| `LANGCHAIN_PROJECT` / `LANGSMITH_PROJECT` | Default project name | `default` |
| `LANGCHAIN_TRACING_V2` / `LANGSMITH_TRACING` | Enable tracing (`true`/`1`) | `false` |

All settings can be overridden via functional options:

```go
client, err := langsmith.NewClient(
    langsmith.WithAPIKey("lsv2_pt_..."),
    langsmith.WithEndpoint("https://api.smith.langchain.com"),
    langsmith.WithProject("my-project"),
    langsmith.WithMaxRetries(5),
    langsmith.WithTimeout(10 * time.Second),
    langsmith.WithLogger(slog.Default()),
    langsmith.WithOnBatchError(func(err error, count int) {
        slog.Error("batch flush failed", "error", err, "items_lost", count)
    }),
)
```

## Log Traces

### Using the Trace Helpers

The simplest way to trace is with the context-based helpers. They automatically manage parent-child relationships via `context.Context`:

```go
// Trace wraps a function and records it as a run
err := langsmith.Trace(ctx, "my-step", langsmith.RunTypeChain, func(ctx context.Context) error {
    // ctx carries the RunTree — child calls nest automatically
    return nil
}, langsmith.WithRunTreeClient(client))

// TraceFunc captures a typed return value
result, err := langsmith.TraceFunc(ctx, "generate", langsmith.RunTypeLLM,
    func(ctx context.Context) (string, error) {
        return callLLM("prompt"), nil
    },
    langsmith.WithRunTreeClient(client),
)

// TraceWithIO captures typed inputs and outputs
output, err := langsmith.TraceWithIO(ctx, "classify", langsmith.RunTypeChain,
    map[string]any{"text": "hello world"},
    func(ctx context.Context, input map[string]any) (map[string]any, error) {
        return map[string]any{"label": "greeting"}, nil
    },
    langsmith.WithRunTreeClient(client),
)
```

### Using RunTree Directly

For more control, create `RunTree` objects directly:

```go
// Create a root run
root := langsmith.NewRunTree("my-pipeline", langsmith.RunTypeChain,
    langsmith.WithRunTreeClient(client),
    langsmith.WithRunTreeInputs(map[string]any{"query": "hello"}),
)
root.PostRun()

// Create a child run
child := root.CreateChild("llm-call", langsmith.RunTypeLLM)
child.SetInputs(map[string]any{"prompt": "Answer: hello"})
child.PostRun()

// ... do work ...

child.End(langsmith.WithEndOutputs(map[string]any{"response": "Hi there!"}))
root.End(langsmith.WithEndOutputs(map[string]any{"result": "Hi there!"}))
```

### Distributed Tracing via HTTP Headers

Propagate trace context across service boundaries:

```go
// Client side — inject headers
headers := runTree.ToHeaders()
req.Header = headers

// Server side — extract headers
parent := langsmith.RunTreeFromHeaders(r.Header, client)
child := parent.CreateChild("downstream-op", langsmith.RunTypeChain)
```

### HTTP Server Middleware

Automatically trace all incoming requests:

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/chat", chatHandler)

traced := langsmith.TracingMiddleware(client, "my-server")(mux)
http.ListenAndServe(":8080", traced)
```

The middleware captures method, path, query, and response status code on every request.

## Create and Manage Datasets

```go
ctx := context.Background()

// Create a dataset
dataset, err := client.CreateDataset(ctx, langsmith.DatasetCreate{
    Name:        "my-qa-pairs",
    Description: langsmith.StringPtr("Question-answer pairs for eval"),
    DataType:    langsmith.DataTypeKV,
})

// Add examples
examples, err := client.CreateExamples(ctx, []langsmith.ExampleCreate{
    {
        DatasetID: dataset.ID,
        Inputs:    map[string]any{"question": "What is Go?"},
        Outputs:   map[string]any{"answer": "A programming language by Google."},
    },
    {
        DatasetID: dataset.ID,
        Inputs:    map[string]any{"question": "What is LangSmith?"},
        Outputs:   map[string]any{"answer": "An LLM observability platform."},
    },
})

// Upload from CSV
csvData, _ := os.ReadFile("data.csv")
dataset, err = client.UploadCSV(ctx, "csv-dataset", csvData, &langsmith.UploadCSVOptions{
    InputKeys:  []string{"question"},
    OutputKeys: []string{"answer"},
})
```

## Evaluate Runs

Run your application against a dataset and score the results with evaluators:

```go
import (
    langsmith "github.com/ty-cooper/langsmith-go"
    "github.com/ty-cooper/langsmith-go/evaluation"
)

results, err := evaluation.Evaluate(ctx, client, dataset.ID,
    func(inputs map[string]any) (map[string]any, error) {
        // Your application logic
        answer := myLLMChain(inputs["question"].(string))
        return map[string]any{"output": answer}, nil
    },
    evaluation.EvaluateOptions{
        Evaluators: []evaluation.RunEvaluator{
            evaluation.ExactMatch("correctness"),
        },
        ExperimentPrefix: "v1",
        MaxConcurrency:   10,
    },
)

fmt.Printf("Experiment: %s (%d results)\n", results.ExperimentName, len(results.Results))
```

### Custom Evaluators

Implement the `RunEvaluator` interface or use the function adapter:

```go
// Using the function adapter
myEval := evaluation.RunEvaluatorFunc(
    func(run langsmith.Run, example *langsmith.Example) (*evaluation.EvaluationResult, error) {
        output := run.Outputs["output"].(string)
        expected := example.Outputs["answer"].(string)

        score := 0.0
        if strings.Contains(output, expected) {
            score = 1.0
        }
        return &evaluation.EvaluationResult{
            Key:   "contains_answer",
            Score: &score,
        }, nil
    },
)

// Using StringEvaluator for string comparisons
lengthEval := &evaluation.StringEvaluator{
    Key: "length_check",
    EvalFunc: func(prediction, reference string) (*evaluation.EvaluationResult, error) {
        score := 0.0
        if len(prediction) > 0 && len(prediction) < 1000 {
            score = 1.0
        }
        return &evaluation.EvaluationResult{Score: &score}, nil
    },
}
```

### Evaluate Existing Experiments

Score runs from a previous experiment without re-running:

```go
results, err := evaluation.EvaluateExisting(ctx, client, "my-experiment-name",
    evaluation.EvaluateOptions{
        Evaluators: []evaluation.RunEvaluator{myEval},
    },
)
```

## Manage Feedback

Attach scores and comments to runs programmatically:

```go
feedback, err := client.CreateFeedback(ctx, langsmith.FeedbackCreate{
    RunID:   &runID,
    Key:     "user-rating",
    Score:   langsmith.Float64Ptr(0.9),
    Comment: langsmith.StringPtr("Great response!"),
})
```

## Prompt Management

Push, pull, and manage prompts in the LangSmith Prompt Hub:

```go
import "encoding/json"

// Push a prompt
manifest := json.RawMessage(`{"template": "Answer the following: {question}"}`)
commit, err := client.PushPrompt(ctx, "my-prompt", manifest, &langsmith.PushPromptOptions{
    CreateIfNotExists: true,
    Description:       langsmith.StringPtr("QA prompt template"),
    IsPublic:          langsmith.BoolPtr(false),
})

// Pull the latest version
commit, err = client.PullPrompt(ctx, "my-prompt", nil)

// Pull a specific version
commit, err = client.PullPrompt(ctx, "my-prompt", &langsmith.PullPromptOptions{
    CommitHash: langsmith.StringPtr("abc123"),
})
```

## Error Handling

The SDK provides typed errors for fine-grained handling:

```go
_, err := client.ReadDataset(ctx, "nonexistent-id")

// Check for not-found (API 404 or empty list result)
if langsmith.IsNotFound(err) {
    log.Println("dataset does not exist")
}

// Extract API error details
if apiErr := langsmith.AsAPIError(err); apiErr != nil {
    log.Printf("API error: status=%d message=%s", apiErr.StatusCode, apiErr.Message)
}

// Sentinel error for by-name lookups
_, err = client.ReadDatasetByName(ctx, "missing")
if errors.Is(err, langsmith.ErrNotFound) {
    log.Println("no dataset with that name")
}
```

## Concurrency and Batching

The client uses a background goroutine to batch run ingestion for efficiency:

- `CreateRunBatched` / `UpdateRunBatched` are non-blocking fire-and-forget methods
- They return `false` if the queue is full or the client has been closed
- Batches flush on a configurable interval (default 1s) or when batch size is reached (default 100)
- `client.Close()` blocks until all queued runs are flushed
- Configure error reporting with `WithOnBatchError` or `WithLogger`

```go
client, _ := langsmith.NewClient(
    langsmith.WithAPIKey("..."),
    langsmith.WithBatchSize(200),
    langsmith.WithBatchInterval(500 * time.Millisecond),
    langsmith.WithOnBatchError(func(err error, count int) {
        log.Printf("lost %d runs: %v", count, err)
    }),
)
defer client.Close() // flushes remaining items
```

## API Reference

### Client Methods

| Category | Methods |
|---|---|
| **Runs** | `CreateRun`, `ReadRun`, `ListRuns`, `UpdateRun`, `DeleteRun`, `BatchIngestRuns`, `CreateRunBatched`, `UpdateRunBatched`, `GetRunURL` |
| **Datasets** | `CreateDataset`, `ReadDataset`, `ReadDatasetByName`, `ListDatasets`, `UpdateDataset`, `DeleteDataset`, `CloneDataset`, `UploadCSV`, `DatasetDiff` |
| **Examples** | `CreateExample`, `CreateExamples`, `ReadExample`, `ListExamples`, `UpdateExample`, `UpdateExamples`, `DeleteExample`, `DeleteExamples` |
| **Feedback** | `CreateFeedback`, `ReadFeedback`, `ListFeedback`, `UpdateFeedback`, `DeleteFeedback` |
| **Projects** | `CreateProject`, `ReadProject`, `ReadProjectByName`, `ListProjects`, `UpdateProject`, `DeleteProject`, `DeleteProjectByName` |
| **Prompts** | `GetPrompt`, `ListPrompts`, `CreatePrompt`, `UpdatePrompt`, `DeletePrompt`, `PushPrompt`, `PullPrompt`, `LikePrompt`, `UnlikePrompt` |
| **Annotation Queues** | `CreateAnnotationQueue`, `ListAnnotationQueues`, `ReadAnnotationQueue`, `UpdateAnnotationQueue`, `DeleteAnnotationQueue`, `AddRunsToAnnotationQueue`, `GetRunFromAnnotationQueue`, `DeleteRunFromAnnotationQueue` |
| **Sharing** | `ShareRun`, `UnshareRun`, `ReadRunSharedLink`, `RunIsShared` |
| **Server** | `ServerInfo` |

### Tracing

| Function | Description |
|---|---|
| `Trace(ctx, name, runType, fn, opts...)` | Wrap a `func(ctx) error` in a traced run |
| `TraceFunc[T](ctx, name, runType, fn, opts...)` | Wrap a `func(ctx) (T, error)` with typed output |
| `TraceWithIO[I, O](ctx, name, runType, input, fn, opts...)` | Wrap with typed input and output |
| `TracingMiddleware(client, project)` | HTTP middleware for automatic request tracing |
| `NewRunTree(name, runType, opts...)` | Create a root RunTree manually |
| `RunTreeFromContext(ctx)` | Extract current RunTree from context |
| `ContextWithRunTree(ctx, rt)` | Inject RunTree into context |
| `RunTreeFromHeaders(headers, client)` | Reconstruct RunTree from HTTP headers |

### Evaluation

| Function | Description |
|---|---|
| `evaluation.Evaluate(ctx, client, datasetID, target, opts)` | Run a target function against a dataset |
| `evaluation.EvaluateExisting(ctx, client, projectName, opts)` | Evaluate runs from an existing experiment |
| `evaluation.ExactMatch(key)` | Built-in evaluator for exact string match |
| `evaluation.RunEvaluatorFunc(fn)` | Adapt a function into a `RunEvaluator` |

## Additional Documentation

- [LangSmith Documentation](https://docs.smith.langchain.com/)
- [LangSmith Python SDK](https://github.com/langchain-ai/langsmith-sdk) (reference implementation)
- [Go Package Documentation](https://pkg.go.dev/github.com/ty-cooper/langsmith-go)

## License

MIT — see [LICENSE](LICENSE) for details.
