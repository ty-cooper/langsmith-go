package langsmith

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// CreateAnnotationQueue creates a new annotation queue.
func (c *Client) CreateAnnotationQueue(ctx context.Context, create AnnotationQueueCreate) (*AnnotationQueue, error) {
	var result AnnotationQueue
	if err := c.post(ctx, "/annotation-queues", create, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListAnnotationQueues lists annotation queues.
func (c *Client) ListAnnotationQueues(ctx context.Context, opts *ListAnnotationQueuesOptions) ([]AnnotationQueue, error) {
	q := url.Values{}
	if opts != nil {
		if opts.Name != nil {
			q.Set("name", *opts.Name)
		}
		if opts.Limit != nil {
			q.Set("limit", strconv.Itoa(*opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	var results []AnnotationQueue
	if err := c.get(ctx, "/annotation-queues", q, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// ListAnnotationQueuesOptions contains options for listing annotation queues.
type ListAnnotationQueuesOptions struct {
	Name   *string `json:"name,omitempty"`
	Limit  *int    `json:"limit,omitempty"`
	Offset int     `json:"offset,omitempty"`
}

// ReadAnnotationQueue retrieves an annotation queue by ID.
func (c *Client) ReadAnnotationQueue(ctx context.Context, queueID string) (*AnnotationQueue, error) {
	var result AnnotationQueue
	if err := c.get(ctx, fmt.Sprintf("/annotation-queues/%s", queueID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateAnnotationQueue updates an annotation queue.
func (c *Client) UpdateAnnotationQueue(ctx context.Context, queueID string, update AnnotationQueueCreate) (*AnnotationQueue, error) {
	var result AnnotationQueue
	if err := c.patch(ctx, fmt.Sprintf("/annotation-queues/%s", queueID), update, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteAnnotationQueue deletes an annotation queue.
func (c *Client) DeleteAnnotationQueue(ctx context.Context, queueID string) error {
	return c.delete(ctx, fmt.Sprintf("/annotation-queues/%s", queueID), nil)
}

// AddRunsToAnnotationQueue adds runs to an annotation queue.
func (c *Client) AddRunsToAnnotationQueue(ctx context.Context, queueID string, runIDs []string) error {
	body := runIDs
	return c.post(ctx, fmt.Sprintf("/annotation-queues/%s/runs", queueID), body, nil)
}

// GetRunFromAnnotationQueue gets the next run from an annotation queue.
func (c *Client) GetRunFromAnnotationQueue(ctx context.Context, queueID string) (*AnnotationQueueRunSchema, error) {
	var result AnnotationQueueRunSchema
	if err := c.get(ctx, fmt.Sprintf("/annotation-queues/%s/run", queueID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteRunFromAnnotationQueue removes a run from an annotation queue.
func (c *Client) DeleteRunFromAnnotationQueue(ctx context.Context, queueID, runID string) error {
	return c.delete(ctx, fmt.Sprintf("/annotation-queues/%s/runs/%s", queueID, runID), nil)
}
