package langsmith

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
)

// CreateDataset creates a new dataset.
func (c *Client) CreateDataset(ctx context.Context, create DatasetCreate) (*Dataset, error) {
	var result Dataset
	if err := c.post(ctx, "/datasets", create, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadDataset retrieves a dataset by ID.
func (c *Client) ReadDataset(ctx context.Context, datasetID string) (*Dataset, error) {
	var result Dataset
	if err := c.get(ctx, idPath("/datasets", datasetID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadDatasetByName retrieves a dataset by name.
// Returns ErrNotFound if no dataset matches.
func (c *Client) ReadDatasetByName(ctx context.Context, name string) (*Dataset, error) {
	q := url.Values{}
	q.Set("name", name)
	var results []Dataset
	if err := c.get(ctx, "/datasets", q, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("dataset %q: %w", name, ErrNotFound)
	}
	return &results[0], nil
}

// ListDatasets lists datasets matching the given options.
func (c *Client) ListDatasets(ctx context.Context, opts *ListDatasetsOptions) ([]Dataset, error) {
	q := url.Values{}
	if opts != nil {
		if opts.Name != nil {
			q.Set("name", *opts.Name)
		}
		if opts.DataType != nil {
			q.Set("data_type", string(*opts.DataType))
		}
		if opts.Limit != nil {
			q.Set("limit", strconv.Itoa(*opts.Limit))
		}
		if opts.Offset > 0 {
			q.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	var results []Dataset
	if err := c.get(ctx, "/datasets", q, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// UpdateDataset updates a dataset.
func (c *Client) UpdateDataset(ctx context.Context, datasetID string, update DatasetUpdate) (*Dataset, error) {
	var result Dataset
	if err := c.patch(ctx, idPath("/datasets", datasetID), update, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteDataset deletes a dataset by ID.
func (c *Client) DeleteDataset(ctx context.Context, datasetID string) error {
	return c.del(ctx, idPath("/datasets", datasetID), nil)
}

// CloneDataset clones a dataset with a new name.
func (c *Client) CloneDataset(ctx context.Context, datasetID string, newName string) (*Dataset, error) {
	body := map[string]string{
		"source_dataset_id": datasetID,
		"name":              newName,
	}
	var result Dataset
	if err := c.post(ctx, "/datasets/clone", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UploadCSV uploads a CSV file to create a dataset.
func (c *Client) UploadCSV(ctx context.Context, datasetName string, csvData []byte, opts *UploadCSVOptions) (*Dataset, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if err := writer.WriteField("name", datasetName); err != nil {
		return nil, fmt.Errorf("upload csv: write name field: %w", err)
	}
	if opts != nil {
		if opts.Description != nil {
			if err := writer.WriteField("description", *opts.Description); err != nil {
				return nil, fmt.Errorf("upload csv: write description field: %w", err)
			}
		}
		for _, k := range opts.InputKeys {
			if err := writer.WriteField("input_keys", k); err != nil {
				return nil, fmt.Errorf("upload csv: write input_keys field: %w", err)
			}
		}
		for _, k := range opts.OutputKeys {
			if err := writer.WriteField("output_keys", k); err != nil {
				return nil, fmt.Errorf("upload csv: write output_keys field: %w", err)
			}
		}
	}

	part, err := writer.CreateFormFile("file", "data.csv")
	if err != nil {
		return nil, fmt.Errorf("upload csv: create form file: %w", err)
	}
	if _, err := part.Write(csvData); err != nil {
		return nil, fmt.Errorf("upload csv: write csv data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("upload csv: close multipart writer: %w", err)
	}

	fullURL := c.endpoint + "/datasets/upload"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("upload csv: create request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload csv: http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("upload csv: read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result Dataset
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("upload csv: decode response: %w", err)
	}
	return &result, nil
}

// DatasetDiff returns the diff between two versions of a dataset.
func (c *Client) DatasetDiff(ctx context.Context, datasetID string, fromVersion, toVersion string) (map[string]any, error) {
	q := url.Values{}
	q.Set("from_version", fromVersion)
	q.Set("to_version", toVersion)
	var result map[string]any
	if err := c.get(ctx, fmt.Sprintf("/datasets/%s/diff", datasetID), q, &result); err != nil {
		return nil, err
	}
	return result, nil
}
