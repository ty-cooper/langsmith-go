package langsmith

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
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

// startTrace creates or nests a RunTree, posts it, and returns the new
// context. This is the shared setup for all Trace* functions.
func startTrace(ctx context.Context, name string, runType RunType, opts ...RunTreeOption) (context.Context, *RunTree) {
	parent := RunTreeFromContext(ctx)

	var rt *RunTree
	if parent != nil {
		rt = parent.CreateChild(name, runType, opts...)
	} else {
		rt = NewRunTree(name, runType, opts...)
	}

	rt.PostRun()
	return ContextWithRunTree(ctx, rt), rt
}

// endTrace finalizes a RunTree based on whether an error occurred.
func endTrace(rt *RunTree, err error, endOpts ...EndOption) {
	if err != nil {
		rt.End(WithEndError(err.Error()))
	} else {
		rt.End(endOpts...)
	}
}

// Trace executes fn within a new child run. If a RunTree exists in the context,
// the new run becomes its child; otherwise a new root run is created.
// The run is automatically posted before fn executes and patched after it completes.
//
// Usage:
//
//	err := langsmith.Trace(ctx, "my-step", langsmith.RunTypeChain, func(ctx context.Context) error {
//	    rt := langsmith.RunTreeFromContext(ctx)
//	    rt.SetInputs(map[string]any{"question": "hello"})
//	    rt.SetOutputs(map[string]any{"answer": "world"})
//	    return nil
//	})
func Trace(ctx context.Context, name string, runType RunType, fn func(ctx context.Context) error, opts ...RunTreeOption) error {
	childCtx, rt := startTrace(ctx, name, runType, opts...)
	err := fn(childCtx)
	endTrace(rt, err)
	return err
}

// TraceFunc is like Trace but for functions that return a value.
//
// Usage:
//
//	result, err := langsmith.TraceFunc(ctx, "my-step", langsmith.RunTypeLLM,
//	    func(ctx context.Context) (string, error) {
//	        return "hello world", nil
//	    },
//	)
func TraceFunc[T any](ctx context.Context, name string, runType RunType, fn func(ctx context.Context) (T, error), opts ...RunTreeOption) (T, error) {
	childCtx, rt := startTrace(ctx, name, runType, opts...)
	result, err := fn(childCtx)
	endTrace(rt, err, WithEndOutputs(map[string]any{"output": result}))
	return result, err
}

// TraceWithIO is like Trace but captures typed inputs and outputs.
//
// Usage:
//
//	output, err := langsmith.TraceWithIO(ctx, "classify", langsmith.RunTypeChain,
//	    map[string]any{"text": "hello"},
//	    func(ctx context.Context, input map[string]any) (map[string]any, error) {
//	        return map[string]any{"class": "greeting"}, nil
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
	childCtx, rt := startTrace(ctx, name, runType, opts...)
	rt.SetInputs(map[string]any{"input": input})
	result, err := fn(childCtx, input)
	endTrace(rt, err, WithEndOutputs(map[string]any{"output": result}))
	return result, err
}

// responseCapture wraps http.ResponseWriter to capture the status code.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rc *responseCapture) WriteHeader(code int) {
	if !rc.written {
		rc.statusCode = code
		rc.written = true
	}
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if !rc.written {
		rc.statusCode = http.StatusOK
		rc.written = true
	}
	return rc.ResponseWriter.Write(b)
}

// TracingMiddleware returns an HTTP middleware that creates a root RunTree
// for each incoming request and attaches it to the request context.
// It captures the response status code in the run outputs.
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

			rt.SetInputs(map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
				"query":  r.URL.RawQuery,
			})
			rt.PostRun()

			ctx := ContextWithRunTree(r.Context(), rt)
			rc := &responseCapture{ResponseWriter: w, statusCode: http.StatusOK}

			defer func() {
				if rec := recover(); rec != nil {
					errMsg := fmt.Sprintf("panic: %v", rec)
					rt.End(WithEndError(errMsg))
					panic(rec) // re-panic after recording the run
				}
			}()

			next.ServeHTTP(rc, r.WithContext(ctx))

			rt.End(WithEndOutputs(map[string]any{
				"status_code": rc.statusCode,
				"status":      strconv.Itoa(rc.statusCode) + " " + http.StatusText(rc.statusCode),
			}))
		})
	}
}
