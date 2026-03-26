package langsmith

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
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
func (c *Client) CreateRunBatched(run RunCreate) {
	if run.SessionName == "" && run.SessionID == nil {
		run.SessionName = c.project
	}
	c.submitBatch("post", run)
}

// UpdateRun updates an existing run.
func (c *Client) UpdateRun(ctx context.Context, runID string, update RunUpdate) error {
	return c.patch(ctx, fmt.Sprintf("/runs/%s", runID), update, nil)
}

// UpdateRunBatched submits a run update to the background batch worker.
func (c *Client) UpdateRunBatched(runID string, update RunUpdate) {
	payload := struct {
		RunUpdate
		ID string `json:"id"`
	}{
		RunUpdate: update,
		ID:        runID,
	}
	c.submitBatch("patch", payload)
}

// ReadRun retrieves a single run by ID.
func (c *Client) ReadRun(ctx context.Context, runID string) (*Run, error) {
	var result Run
	if err := c.get(ctx, fmt.Sprintf("/runs/%s", runID), nil, &result); err != nil {
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
	// Convert API URL to app URL
	if baseURL == "https://api.smith.langchain.com" {
		baseURL = "https://smith.langchain.com"
	}

	q := url.Values{}
	if cfg.projectID != "" {
		return fmt.Sprintf("%s/o/default/projects/p/%s/r/%s?%s", baseURL, cfg.projectID, runID, q.Encode())
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
	return func(c *getRunURLConfig) {
		c.projectID = id
	}
}

// DeleteRun deletes a run by ID.
func (c *Client) DeleteRun(ctx context.Context, runID string) error {
	return c.delete(ctx, fmt.Sprintf("/runs/%s", runID), nil)
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
	return c.delete(ctx, fmt.Sprintf("/runs/%s/share", runID), nil)
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
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListRunsIterator returns an iterator for paginating through runs.
func (c *Client) ListRunsIterator(ctx context.Context, opts ListRunsOptions) *RunIterator {
	return &RunIterator{
		client: c,
		ctx:    ctx,
		opts:   opts,
	}
}

// RunIterator provides pagination over runs.
type RunIterator struct {
	client  *Client
	ctx     context.Context
	opts    ListRunsOptions
	buffer  []Run
	offset  int
	done    bool
}

// Next returns the next run. Returns nil, nil when iteration is complete.
func (it *RunIterator) Next() (*Run, error) {
	if len(it.buffer) == 0 {
		if it.done {
			return nil, nil
		}
		it.opts.Offset = it.offset
		if it.opts.Limit == nil {
			it.opts.Limit = IntPtr(100)
		}
		runs, err := it.client.ListRuns(it.ctx, it.opts)
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
func (it *RunIterator) All() ([]Run, error) {
	var all []Run
	for {
		run, err := it.Next()
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

// buildRunQuery builds query parameters from ListRunsOptions for GET-style endpoints.
func buildRunQuery(opts ListRunsOptions) url.Values {
	q := url.Values{}
	if opts.ProjectID != nil {
		q.Set("project_id", *opts.ProjectID)
	}
	if opts.ProjectName != nil {
		q.Set("project_name", *opts.ProjectName)
	}
	if opts.RunType != nil {
		q.Set("run_type", string(*opts.RunType))
	}
	if opts.Limit != nil {
		q.Set("limit", strconv.Itoa(*opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	return q
}
