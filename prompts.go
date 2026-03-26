package langsmith

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GetPrompt retrieves a prompt by name (format: "owner/repo" or just "repo").
func (c *Client) GetPrompt(ctx context.Context, promptIdentifier string) (*Prompt, error) {
	var result Prompt
	if err := c.get(ctx, fmt.Sprintf("/repos/%s", promptIdentifier), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListPrompts lists prompts in the hub.
func (c *Client) ListPrompts(ctx context.Context, opts *ListPromptsOptions) (*ListPromptsResponse, error) {
	q := url.Values{}
	if opts != nil {
		if opts.IsPublic != nil {
			q.Set("is_public", strconv.FormatBool(*opts.IsPublic))
		}
		if opts.IsArchived != nil {
			q.Set("is_archived", strconv.FormatBool(*opts.IsArchived))
		}
		if opts.SortField != nil {
			q.Set("sort_field", string(*opts.SortField))
		}
		if opts.SortDirection != nil {
			q.Set("sort_direction", *opts.SortDirection)
		}
		if opts.Query != nil {
			q.Set("query", *opts.Query)
		}
		for _, tag := range opts.Tags {
			q.Add("tags", tag)
		}
		if opts.Limit != nil {
			q.Set("limit", strconv.Itoa(*opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	var result ListPromptsResponse
	if err := c.get(ctx, "/repos", q, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePrompt creates a new prompt repo.
func (c *Client) CreatePrompt(ctx context.Context, req CreatePromptRequest) (*Prompt, error) {
	var result Prompt
	if err := c.post(ctx, "/repos", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePrompt updates an existing prompt repo.
func (c *Client) UpdatePrompt(ctx context.Context, repoHandle string, opts UpdatePromptOptions) (*Prompt, error) {
	var result Prompt
	if err := c.patch(ctx, fmt.Sprintf("/repos/%s", repoHandle), opts, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePrompt deletes a prompt repo.
func (c *Client) DeletePrompt(ctx context.Context, repoHandle string) error {
	return c.del(ctx, fmt.Sprintf("/repos/%s", repoHandle), nil)
}

// PushPrompt pushes a new commit to a prompt repo. Creates the repo if
// opts.CreateIfNotExists is true.
func (c *Client) PushPrompt(ctx context.Context, repoHandle string, manifest json.RawMessage, opts *PushPromptOptions) (*PromptCommit, error) {
	if opts != nil && opts.CreateIfNotExists {
		_, err := c.GetPrompt(ctx, repoHandle)
		if err != nil {
			if IsNotFound(err) {
				req := CreatePromptRequest{
					RepoHandle:  repoHandle,
					Description: opts.Description,
					IsPublic:    opts.IsPublic,
					Tags:        opts.Tags,
				}
				if _, err := c.CreatePrompt(ctx, req); err != nil {
					return nil, fmt.Errorf("push prompt: create repo: %w", err)
				}
			} else {
				return nil, fmt.Errorf("push prompt: check repo: %w", err)
			}
		}
	}

	body := map[string]any{"manifest": manifest}
	if opts != nil && opts.ParentCommitHash != nil {
		body["parent_commit"] = *opts.ParentCommitHash
	}

	var result PromptCommit
	if err := c.post(ctx, fmt.Sprintf("/commits/%s", repoHandle), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PullPrompt pulls the latest commit from a prompt repo.
func (c *Client) PullPrompt(ctx context.Context, repoHandle string, opts *PullPromptOptions) (*PromptCommit, error) {
	path := fmt.Sprintf("/commits/%s/latest", repoHandle)
	if opts != nil && opts.CommitHash != nil {
		path = fmt.Sprintf("/commits/%s/%s", repoHandle, *opts.CommitHash)
	}

	var result PromptCommit
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// LikePrompt likes a prompt repo.
func (c *Client) LikePrompt(ctx context.Context, repoHandle string) error {
	return c.post(ctx, fmt.Sprintf("/likes/%s", repoHandle), nil, nil)
}

// UnlikePrompt removes a like from a prompt repo.
func (c *Client) UnlikePrompt(ctx context.Context, repoHandle string) error {
	return c.del(ctx, fmt.Sprintf("/likes/%s", repoHandle), nil)
}
