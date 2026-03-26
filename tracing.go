package langsmith

import (
	"context"
	"net/http"
)

type contextKey string

const runTreeContextKey contextKey = "langsmith-run-tree"

// ContextWithRunTree returns a new context with the RunTree attached.
func ContextWithRunTree(ctx context.Context, rt *RunTree) context.Context {
	return context.WithValue(ctx, runTreeContextKey, rt)
}

// RunTreeFromContext extracts the current RunTree from the context.
// Returns nil if no RunTree is present.
func RunTreeFromContext(ctx context.Context) *RunTree {
	rt, _ := ctx.Value(runTreeContextKey).(*RunTree)
	return rt
}

// Trace executes fn within a new child run. If a RunTree exists in the context,
// the new run becomes its child; otherwise a new root run is created.
// The run is automatically posted before fn executes and patched after it completes.
//
// Usage:
//
//	err := langsmith.Trace(ctx, "my-step", langsmith.RunTypeChain, func(ctx context.Context) error {
//	    // ctx now contains the child RunTree
//	    rt := langsmith.RunTreeFromContext(ctx)
//	    rt.SetInputs(map[string]interface{}{"question": "hello"})
//	    // ... do work ...
//	    rt.SetOutputs(map[string]interface{}{"answer": "world"})
//	    return nil
//	})
func Trace(ctx context.Context, name string, runType RunType, fn func(ctx context.Context) error, opts ...RunTreeOption) error {
	parent := RunTreeFromContext(ctx)

	var rt *RunTree
	if parent != nil {
		rt = parent.CreateChild(name, runType, opts...)
	} else {
		rt = NewRunTree(name, runType, opts...)
	}

	rt.PostRun()
	childCtx := ContextWithRunTree(ctx, rt)

	err := fn(childCtx)
	if err != nil {
		errStr := err.Error()
		rt.End(WithEndError(errStr))
	} else {
		rt.End()
	}

	return err
}

// TraceFunc is like Trace but for functions that return a value.
// It uses generics to capture the output type.
//
// Usage:
//
//	result, err := langsmith.TraceFunc(ctx, "my-step", langsmith.RunTypeLLM,
//	    func(ctx context.Context) (string, error) {
//	        return "hello world", nil
//	    },
//	)
func TraceFunc[T any](ctx context.Context, name string, runType RunType, fn func(ctx context.Context) (T, error), opts ...RunTreeOption) (T, error) {
	parent := RunTreeFromContext(ctx)

	var rt *RunTree
	if parent != nil {
		rt = parent.CreateChild(name, runType, opts...)
	} else {
		rt = NewRunTree(name, runType, opts...)
	}

	rt.PostRun()
	childCtx := ContextWithRunTree(ctx, rt)

	result, err := fn(childCtx)
	if err != nil {
		errStr := err.Error()
		rt.End(WithEndError(errStr))
	} else {
		rt.End(WithEndOutputs(map[string]interface{}{"output": result}))
	}

	return result, err
}

// TraceWithIO is like Trace but captures typed inputs and outputs.
//
// Usage:
//
//	output, err := langsmith.TraceWithIO(ctx, "classify", langsmith.RunTypeChain,
//	    map[string]interface{}{"text": "hello"},
//	    func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
//	        return map[string]interface{}{"class": "greeting"}, nil
//	    },
//	)
func TraceWithIO[I any, O any](
	ctx context.Context,
	name string,
	runType RunType,
	input I,
	fn func(ctx context.Context, input I) (O, error),
	opts ...RunTreeOption,
) (O, error) {
	parent := RunTreeFromContext(ctx)

	var rt *RunTree
	if parent != nil {
		rt = parent.CreateChild(name, runType, opts...)
	} else {
		rt = NewRunTree(name, runType, opts...)
	}

	rt.SetInputs(map[string]interface{}{"input": input})
	rt.PostRun()
	childCtx := ContextWithRunTree(ctx, rt)

	result, err := fn(childCtx, input)
	if err != nil {
		errStr := err.Error()
		rt.End(WithEndError(errStr))
	} else {
		rt.End(WithEndOutputs(map[string]interface{}{"output": result}))
	}

	return result, err
}

// TracingMiddleware returns an HTTP middleware that creates a root RunTree
// for each incoming request and attaches it to the request context.
// This is useful for instrumenting HTTP servers.
//
// Usage:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/api", handler)
//	wrapped := langsmith.TracingMiddleware(client, "my-server")(mux)
//	http.ListenAndServe(":8080", wrapped)
func TracingMiddleware(client *Client, projectName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for incoming trace context from upstream.
			parent := RunTreeFromHeaders(r.Header, client)

			var rt *RunTree
			if parent != nil {
				rt = parent.CreateChild(r.Method+" "+r.URL.Path, RunTypeChain,
					WithRunTreeClient(client),
					WithRunTreeSessionName(projectName),
				)
			} else {
				rt = NewRunTree(r.Method+" "+r.URL.Path, RunTypeChain,
					WithRunTreeClient(client),
					WithRunTreeSessionName(projectName),
				)
			}

			rt.SetInputs(map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
				"query":  r.URL.RawQuery,
			})
			rt.PostRun()

			ctx := ContextWithRunTree(r.Context(), rt)
			next.ServeHTTP(w, r.WithContext(ctx))

			rt.End()
		})
	}
}
