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
	path := fmt.Sprintf("/repos/%s", promptIdentifier)
	if err := c.get(ctx, path, nil, &result); err != nil {
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
func (c *Client) CreatePrompt(ctx context.Context, repoHandle string, opts *CreatePromptOptions) (*Prompt, error) {
	body := map[string]interface{}{
		"repo_handle": repoHandle,
	}
	if opts != nil {
		if opts.Description != nil {
			body["description"] = *opts.Description
		}
		if opts.Readme != nil {
			body["readme"] = *opts.Readme
		}
		if opts.IsPublic != nil {
			body["is_public"] = *opts.IsPublic
		}
		if opts.Tags != nil {
			body["tags"] = opts.Tags
		}
	}
	var result Prompt
	if err := c.post(ctx, "/repos", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePromptOptions contains options for creating a prompt.
type CreatePromptOptions struct {
	Description *string  `json:"description,omitempty"`
	Readme      *string  `json:"readme,omitempty"`
	IsPublic    *bool    `json:"is_public,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdatePrompt updates an existing prompt repo.
func (c *Client) UpdatePrompt(ctx context.Context, repoHandle string, opts UpdatePromptOptions) (*Prompt, error) {
	var result Prompt
	if err := c.patch(ctx, fmt.Sprintf("/repos/%s", repoHandle), opts, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePromptOptions contains options for updating a prompt.
type UpdatePromptOptions struct {
	Description *string  `json:"description,omitempty"`
	Readme      *string  `json:"readme,omitempty"`
	IsPublic    *bool    `json:"is_public,omitempty"`
	IsArchived  *bool    `json:"is_archived,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// DeletePrompt deletes a prompt repo.
func (c *Client) DeletePrompt(ctx context.Context, repoHandle string) error {
	return c.delete(ctx, fmt.Sprintf("/repos/%s", repoHandle), nil)
}

// PushPrompt pushes a new commit to a prompt repo. Creates the repo if it doesn't exist.
func (c *Client) PushPrompt(ctx context.Context, repoHandle string, manifest json.RawMessage, opts *PushPromptOptions) (*PromptCommit, error) {
	// Ensure repo exists.
	if opts != nil && opts.CreateIfNotExists {
		_, err := c.GetPrompt(ctx, repoHandle)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
				createOpts := &CreatePromptOptions{
					Description: opts.Description,
					IsPublic:    opts.IsPublic,
					Tags:        opts.Tags,
				}
				if _, err := c.CreatePrompt(ctx, repoHandle, createOpts); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	body := map[string]interface{}{
		"manifest": manifest,
	}
	if opts != nil && opts.ParentCommitHash != nil {
		body["parent_commit"] = *opts.ParentCommitHash
	}

	var result PromptCommit
	if err := c.post(ctx, fmt.Sprintf("/commits/%s", repoHandle), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PushPromptOptions contains options for pushing a prompt.
type PushPromptOptions struct {
	CreateIfNotExists bool     `json:"-"`
	Description       *string  `json:"description,omitempty"`
	IsPublic          *bool    `json:"is_public,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	ParentCommitHash  *string  `json:"parent_commit,omitempty"`
}

// PullPrompt pulls the latest commit from a prompt repo.
func (c *Client) PullPrompt(ctx context.Context, repoHandle string, opts *PullPromptOptions) (*PromptCommit, error) {
	path := fmt.Sprintf("/commits/%s/latest", repoHandle)
	q := url.Values{}
	if opts != nil && opts.CommitHash != nil {
		path = fmt.Sprintf("/commits/%s/%s", repoHandle, *opts.CommitHash)
	}

	var result PromptCommit
	if err := c.get(ctx, path, q, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PullPromptOptions contains options for pulling a prompt.
type PullPromptOptions struct {
	CommitHash *string `json:"commit_hash,omitempty"`
}

// LikePrompt likes a prompt repo.
func (c *Client) LikePrompt(ctx context.Context, repoHandle string) error {
	return c.post(ctx, fmt.Sprintf("/likes/%s", repoHandle), nil, nil)
}

// UnlikePrompt removes a like from a prompt repo.
func (c *Client) UnlikePrompt(ctx context.Context, repoHandle string) error {
	return c.delete(ctx, fmt.Sprintf("/likes/%s", repoHandle), nil)
}
