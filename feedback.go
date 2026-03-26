package langsmith

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// CreateFeedback creates new feedback on a run.
func (c *Client) CreateFeedback(ctx context.Context, create FeedbackCreate) (*Feedback, error) {
	var result Feedback
	if err := c.post(ctx, "/feedback", create, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadFeedback retrieves feedback by ID.
func (c *Client) ReadFeedback(ctx context.Context, feedbackID string) (*Feedback, error) {
	var result Feedback
	if err := c.get(ctx, fmt.Sprintf("/feedback/%s", feedbackID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListFeedback lists feedback matching the given options.
func (c *Client) ListFeedback(ctx context.Context, opts *ListFeedbackOptions) ([]Feedback, error) {
	q := url.Values{}
	if opts != nil {
		if opts.RunID != nil {
			q.Set("run", *opts.RunID)
		}
		for _, id := range opts.RunIDs {
			q.Add("run", id)
		}
		if opts.Key != nil {
			q.Set("key", *opts.Key)
		}
		if opts.Limit != nil {
			q.Set("limit", strconv.Itoa(*opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	var results []Feedback
	if err := c.get(ctx, "/feedback", q, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// ListFeedbackOptions contains options for listing feedback.
type ListFeedbackOptions struct {
	RunID  *string  `json:"run_id,omitempty"`
	RunIDs []string `json:"run_ids,omitempty"`
	Key    *string  `json:"key,omitempty"`
	Limit  *int     `json:"limit,omitempty"`
	Offset int      `json:"offset,omitempty"`
}

// UpdateFeedback updates existing feedback.
func (c *Client) UpdateFeedback(ctx context.Context, feedbackID string, update FeedbackUpdate) error {
	return c.patch(ctx, fmt.Sprintf("/feedback/%s", feedbackID), update, nil)
}

// DeleteFeedback deletes feedback by ID.
func (c *Client) DeleteFeedback(ctx context.Context, feedbackID string) error {
	return c.delete(ctx, fmt.Sprintf("/feedback/%s", feedbackID), nil)
}
