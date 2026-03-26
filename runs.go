package langsmith

import (
	"context"
	"fmt"
	"net/url"
)

// CreateRun creates a new run in LangSmith.
func (c *Client) CreateRun(ctx context.Context, run RunCreate) (*Run, error) {
	if run.SessionName == "" && run.SessionID == nil {
		run.SessionName = c.project
	}
	var result Run
	if err := c.post(ctx, "/runs", run, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRunBatched submits a run creation to the background batch worker.
// Returns false if the queue is full or the worker has been closed,
// in which case the item is dropped.
func (c *Client) CreateRunBatched(run RunCreate) bool {
	if run.SessionName == "" && run.SessionID == nil {
		run.SessionName = c.project
	}
	return c.submitBatch("post", run)
}

// UpdateRun updates an existing run.
func (c *Client) UpdateRun(ctx context.Context, runID string, update RunUpdate) error {
	return c.patch(ctx, idPath("/runs", runID), update, nil)
}

// UpdateRunBatched submits a run update to the background batch worker.
// Returns false if the queue is full or the worker has been closed,
// in which case the item is dropped.
func (c *Client) UpdateRunBatched(runID string, update RunUpdate) bool {
	payload := struct {
		RunUpdate
		ID string `json:"id"`
	}{
		RunUpdate: update,
		ID:        runID,
	}
	return c.submitBatch("patch", payload)
}

// ReadRun retrieves a single run by ID.
func (c *Client) ReadRun(ctx context.Context, runID string) (*Run, error) {
	var result Run
	if err := c.get(ctx, idPath("/runs", runID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRuns lists runs matching the given options.
func (c *Client) ListRuns(ctx context.Context, opts ListRunsOptions) ([]Run, error) {
	var result []Run
	if err := c.post(ctx, "/runs/query", opts, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// BatchIngestRuns sends a batch of run creates and updates.
func (c *Client) BatchIngestRuns(ctx context.Context, req BatchIngestRequest) error {
	return c.post(ctx, "/runs/batch", req, nil)
}

// GetRunURL returns the web URL for viewing a run in the LangSmith UI.
func (c *Client) GetRunURL(runID string, opts ...GetRunURLOption) string {
	cfg := &getRunURLConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	baseURL := c.endpoint
	// Convert API URL to app URL.
	if baseURL == "https://api.smith.langchain.com" {
		baseURL = "https://smith.langchain.com"
	}

	if cfg.projectID != "" {
		return fmt.Sprintf("%s/o/default/projects/p/%s/r/%s", baseURL, cfg.projectID, runID)
	}
	return fmt.Sprintf("%s/o/default/r/%s", baseURL, runID)
}

type getRunURLConfig struct {
	projectID string
}

// GetRunURLOption configures GetRunURL.
type GetRunURLOption func(*getRunURLConfig)

// WithRunURLProjectID sets the project ID for the run URL.
func WithRunURLProjectID(id string) GetRunURLOption {
	return func(c *getRunURLConfig) { c.projectID = id }
}

// DeleteRun deletes a run by ID.
func (c *Client) DeleteRun(ctx context.Context, runID string) error {
	return c.del(ctx, idPath("/runs", runID), nil)
}

// ShareRun creates a publicly accessible link for a run.
func (c *Client) ShareRun(ctx context.Context, runID string) (*SharedRunURL, error) {
	var result SharedRunURL
	body := map[string]string{"run_id": runID}
	if err := c.put(ctx, fmt.Sprintf("/runs/%s/share", runID), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UnshareRun removes the shared link for a run.
func (c *Client) UnshareRun(ctx context.Context, runID string) error {
	return c.del(ctx, fmt.Sprintf("/runs/%s/share", runID), nil)
}

// ReadRunSharedLink retrieves the shared link for a run.
func (c *Client) ReadRunSharedLink(ctx context.Context, runID string) (*SharedRunURL, error) {
	var result SharedRunURL
	if err := c.get(ctx, fmt.Sprintf("/runs/%s/share", runID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RunIsShared checks if a run has a shared link.
func (c *Client) RunIsShared(ctx context.Context, runID string) (bool, error) {
	_, err := c.ReadRunSharedLink(ctx, runID)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListRunsIterator returns an iterator for paginating through runs.
// Each call to Next uses the provided context for the underlying API call.
func (c *Client) ListRunsIterator(opts ListRunsOptions) *RunIterator {
	return &RunIterator{
		client: c,
		opts:   opts,
	}
}

// RunIterator provides pagination over runs.
type RunIterator struct {
	client *Client
	opts   ListRunsOptions
	buffer []Run
	offset int
	done   bool
}

// Next returns the next run. Returns nil, nil when iteration is complete.
func (it *RunIterator) Next(ctx context.Context) (*Run, error) {
	if len(it.buffer) == 0 {
		if it.done {
			return nil, nil
		}
		it.opts.Offset = it.offset
		if it.opts.Limit == nil {
			it.opts.Limit = IntPtr(100)
		}
		runs, err := it.client.ListRuns(ctx, it.opts)
		if err != nil {
			return nil, err
		}
		if len(runs) == 0 {
			it.done = true
			return nil, nil
		}
		it.buffer = runs
		it.offset += len(runs)
		if len(runs) < *it.opts.Limit {
			it.done = true
		}
	}

	run := it.buffer[0]
	it.buffer = it.buffer[1:]
	return &run, nil
}

// All collects all remaining runs into a slice.
func (it *RunIterator) All(ctx context.Context) ([]Run, error) {
	var all []Run
	for {
		run, err := it.Next(ctx)
		if err != nil {
			return nil, err
		}
		if run == nil {
			break
		}
		all = append(all, *run)
	}
	return all, nil
}

// idPath builds a URL path with the given ID safely escaped.
func idPath(base, id string) string {
	return fmt.Sprintf("%s/%s", base, url.PathEscape(id))
}
