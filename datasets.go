package langsmith

import (
	"bytes"
	"context"
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
	if err := c.get(ctx, fmt.Sprintf("/datasets/%s", datasetID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadDatasetByName retrieves a dataset by name.
func (c *Client) ReadDatasetByName(ctx context.Context, name string) (*Dataset, error) {
	q := url.Values{}
	q.Set("name", name)
	var results []Dataset
	if err := c.get(ctx, "/datasets", q, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, &APIError{StatusCode: 404, Message: fmt.Sprintf("dataset %q not found", name)}
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

// ListDatasetsOptions contains options for listing datasets.
type ListDatasetsOptions struct {
	Name     *string   `json:"name,omitempty"`
	DataType *DataType `json:"data_type,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Limit    *int      `json:"limit,omitempty"`
	Offset   int       `json:"offset,omitempty"`
}

// UpdateDataset updates a dataset.
func (c *Client) UpdateDataset(ctx context.Context, datasetID string, update DatasetUpdate) (*Dataset, error) {
	var result Dataset
	if err := c.patch(ctx, fmt.Sprintf("/datasets/%s", datasetID), update, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteDataset deletes a dataset by ID.
func (c *Client) DeleteDataset(ctx context.Context, datasetID string) error {
	return c.delete(ctx, fmt.Sprintf("/datasets/%s", datasetID), nil)
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

	_ = writer.WriteField("name", datasetName)
	if opts != nil {
		if opts.Description != "" {
			_ = writer.WriteField("description", opts.Description)
		}
		if opts.InputKeys != nil {
			for _, k := range opts.InputKeys {
				_ = writer.WriteField("input_keys", k)
			}
		}
		if opts.OutputKeys != nil {
			for _, k := range opts.OutputKeys {
				_ = writer.WriteField("output_keys", k)
			}
		}
	}

	part, err := writer.CreateFormFile("file", "data.csv")
	if err != nil {
		return nil, &LangSmithError{Message: "failed to create form file", Err: err}
	}
	if _, err := part.Write(csvData); err != nil {
		return nil, &LangSmithError{Message: "failed to write CSV data", Err: err}
	}
	writer.Close()

	fullURL := c.endpoint + "/datasets/upload"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, &buf)
	if err != nil {
		return nil, &LangSmithError{Message: "failed to create request", Err: err}
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &LangSmithError{Message: "upload request failed", Err: err}
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result Dataset
	if err := decodeJSON(respBody, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UploadCSVOptions contains options for uploading a CSV.
type UploadCSVOptions struct {
	Description string
	InputKeys   []string
	OutputKeys  []string
}

// DatasetDiff returns the diff between two versions of a dataset.
func (c *Client) DatasetDiff(ctx context.Context, datasetID string, fromVersion, toVersion string) (map[string]interface{}, error) {
	q := url.Values{}
	q.Set("from_version", fromVersion)
	q.Set("to_version", toVersion)
	var result map[string]interface{}
	if err := c.get(ctx, fmt.Sprintf("/datasets/%s/diff", datasetID), q, &result); err != nil {
		return nil, err
	}
	return result, nil
}
