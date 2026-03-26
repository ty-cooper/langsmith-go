package langsmith

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// CreateProject creates a new project (tracer session).
func (c *Client) CreateProject(ctx context.Context, create TracerSessionCreate) (*TracerSession, error) {
	var result TracerSession
	if err := c.post(ctx, "/sessions", create, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadProject retrieves a project by ID.
func (c *Client) ReadProject(ctx context.Context, projectID string) (*TracerSession, error) {
	var result TracerSession
	if err := c.get(ctx, fmt.Sprintf("/sessions/%s", projectID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadProjectByName retrieves a project by name.
func (c *Client) ReadProjectByName(ctx context.Context, name string) (*TracerSession, error) {
	q := url.Values{}
	q.Set("name", name)
	var results []TracerSession
	if err := c.get(ctx, "/sessions", q, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, &APIError{StatusCode: 404, Message: fmt.Sprintf("project %q not found", name)}
	}
	return &results[0], nil
}

// ListProjects lists all projects.
func (c *Client) ListProjects(ctx context.Context, opts *ListProjectsOptions) ([]TracerSession, error) {
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
		if opts.ReferenceDatasetID != nil {
			q.Set("reference_dataset", *opts.ReferenceDatasetID)
		}
	}
	var results []TracerSession
	if err := c.get(ctx, "/sessions", q, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// ListProjectsOptions contains options for listing projects.
type ListProjectsOptions struct {
	Name               *string `json:"name,omitempty"`
	ReferenceDatasetID *string `json:"reference_dataset_id,omitempty"`
	Limit              *int    `json:"limit,omitempty"`
	Offset             int     `json:"offset,omitempty"`
}

// UpdateProject updates a project.
func (c *Client) UpdateProject(ctx context.Context, projectID string, update TracerSessionUpdate) (*TracerSession, error) {
	var result TracerSession
	if err := c.patch(ctx, fmt.Sprintf("/sessions/%s", projectID), update, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteProject deletes a project by ID.
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	return c.delete(ctx, fmt.Sprintf("/sessions/%s", projectID), nil)
}

// DeleteProjectByName deletes a project by name.
func (c *Client) DeleteProjectByName(ctx context.Context, name string) error {
	project, err := c.ReadProjectByName(ctx, name)
	if err != nil {
		return err
	}
	return c.DeleteProject(ctx, project.ID)
}
