package langsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// CreateExample creates a new example in a dataset.
func (c *Client) CreateExample(ctx context.Context, create ExampleCreate) (*Example, error) {
	var result Example
	if err := c.post(ctx, "/examples", create, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateExamples creates multiple examples in a dataset.
func (c *Client) CreateExamples(ctx context.Context, creates []ExampleCreate) ([]Example, error) {
	var results []Example
	if err := c.post(ctx, "/examples/bulk", creates, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// ReadExample retrieves an example by ID.
func (c *Client) ReadExample(ctx context.Context, exampleID string) (*Example, error) {
	var result Example
	if err := c.get(ctx, idPath("/examples", exampleID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListExamples lists examples matching the given options.
func (c *Client) ListExamples(ctx context.Context, opts ListExamplesOptions) ([]Example, error) {
	q := url.Values{}
	if opts.DatasetID != nil {
		q.Set("dataset", *opts.DatasetID)
	}
	if opts.Limit != nil {
		q.Set("limit", strconv.Itoa(*opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	if opts.Metadata != nil {
		metaJSON, err := json.Marshal(opts.Metadata)
		if err != nil {
			return nil, fmt.Errorf("list examples: marshal metadata: %w", err)
		}
		q.Set("metadata", string(metaJSON))
	}
	if opts.Filter != nil {
		q.Set("filter", *opts.Filter)
	}
	var results []Example
	if err := c.get(ctx, "/examples", q, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// UpdateExample updates an existing example.
func (c *Client) UpdateExample(ctx context.Context, exampleID string, update ExampleUpdate) (*Example, error) {
	var result Example
	if err := c.patch(ctx, idPath("/examples", exampleID), update, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateExamples updates multiple examples.
func (c *Client) UpdateExamples(ctx context.Context, updates map[string]ExampleUpdate) error {
	var payload []map[string]any
	for id, update := range updates {
		item := map[string]any{"id": id}
		if update.Inputs != nil {
			item["inputs"] = update.Inputs
		}
		if update.Outputs != nil {
			item["outputs"] = update.Outputs
		}
		if update.Metadata != nil {
			item["metadata"] = update.Metadata
		}
		payload = append(payload, item)
	}
	return c.patch(ctx, "/examples/bulk", payload, nil)
}

// DeleteExample deletes an example by ID.
func (c *Client) DeleteExample(ctx context.Context, exampleID string) error {
	return c.del(ctx, idPath("/examples", exampleID), nil)
}

// DeleteExamples deletes multiple examples by their IDs.
func (c *Client) DeleteExamples(ctx context.Context, exampleIDs []string) error {
	q := url.Values{}
	for _, id := range exampleIDs {
		q.Add("id", id)
	}
	return c.del(ctx, "/examples", q)
}
