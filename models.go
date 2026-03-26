package langsmith

import (
	"encoding/json"
	"time"
)

// RunType represents the type of a run.
type RunType string

const (
	RunTypeTool      RunType = "tool"
	RunTypeChain     RunType = "chain"
	RunTypeLLM       RunType = "llm"
	RunTypeRetriever RunType = "retriever"
	RunTypeEmbedding RunType = "embedding"
	RunTypePrompt    RunType = "prompt"
	RunTypeParser    RunType = "parser"
)

// DataType represents the type of data in a dataset.
type DataType string

const (
	DataTypeKV   DataType = "kv"
	DataTypeLLM  DataType = "llm"
	DataTypeChat DataType = "chat"
)

// FeedbackSourceType represents the source of feedback.
type FeedbackSourceType string

const (
	FeedbackSourceAPI   FeedbackSourceType = "api"
	FeedbackSourceModel FeedbackSourceType = "model"
	FeedbackSourceApp   FeedbackSourceType = "app"
)

// Run represents a single execution span in LangSmith.
type Run struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	RunType         RunType                `json:"run_type"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Status          string                 `json:"status,omitempty"`
	Error           *string                `json:"error,omitempty"`
	Inputs          map[string]interface{} `json:"inputs,omitempty"`
	Outputs         map[string]interface{} `json:"outputs,omitempty"`
	ParentRunID     *string                `json:"parent_run_id,omitempty"`
	TraceID         string                 `json:"trace_id,omitempty"`
	DottedOrder     string                 `json:"dotted_order,omitempty"`
	SessionID       *string                `json:"session_id,omitempty"`
	SessionName     string                 `json:"session_name,omitempty"`
	ReferenceExampleID *string             `json:"reference_example_id,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Events          []map[string]interface{} `json:"events,omitempty"`
	Serialized      map[string]interface{} `json:"serialized,omitempty"`
	Extra           map[string]interface{} `json:"extra,omitempty"`
	PromptTokens    *int                   `json:"prompt_tokens,omitempty"`
	CompletionTokens *int                  `json:"completion_tokens,omitempty"`
	TotalTokens     *int                   `json:"total_tokens,omitempty"`
	PromptCost      *float64               `json:"prompt_cost,omitempty"`
	CompletionCost  *float64               `json:"completion_cost,omitempty"`
	TotalCost       *float64               `json:"total_cost,omitempty"`
	FirstTokenTime  *time.Time             `json:"first_token_time,omitempty"`
	FeedbackStats   map[string]interface{} `json:"feedback_stats,omitempty"`
	ChildRunIDs     []string               `json:"child_run_ids,omitempty"`
	ChildRuns       []Run                  `json:"child_runs,omitempty"`
	Attachments     map[string]Attachment   `json:"attachments,omitempty"`
}

// RunCreate represents the payload for creating a new run.
type RunCreate struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	RunType         RunType                `json:"run_type"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Error           *string                `json:"error,omitempty"`
	Inputs          map[string]interface{} `json:"inputs,omitempty"`
	Outputs         map[string]interface{} `json:"outputs,omitempty"`
	ParentRunID     *string                `json:"parent_run_id,omitempty"`
	TraceID         string                 `json:"trace_id,omitempty"`
	DottedOrder     string                 `json:"dotted_order,omitempty"`
	SessionID       *string                `json:"session_id,omitempty"`
	SessionName     string                 `json:"session_name,omitempty"`
	ReferenceExampleID *string             `json:"reference_example_id,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Events          []map[string]interface{} `json:"events,omitempty"`
	Serialized      map[string]interface{} `json:"serialized,omitempty"`
	Extra           map[string]interface{} `json:"extra,omitempty"`
	Attachments     map[string]Attachment   `json:"attachments,omitempty"`
}

// RunUpdate represents the payload for updating an existing run.
type RunUpdate struct {
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Error      *string                `json:"error,omitempty"`
	Outputs    map[string]interface{} `json:"outputs,omitempty"`
	Events     []map[string]interface{} `json:"events,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// Attachment represents a file attachment on a run.
type Attachment struct {
	MimeType string `json:"mime_type"`
	Data     []byte `json:"data,omitempty"`
	PresignedURL string `json:"presigned_url,omitempty"`
}

// ListRunsOptions contains options for listing runs.
type ListRunsOptions struct {
	ProjectID       *string   `json:"project_id,omitempty"`
	ProjectName     *string   `json:"project_name,omitempty"`
	RunType         *RunType  `json:"run_type,omitempty"`
	TraceID         *string   `json:"trace_id,omitempty"`
	ReferenceExampleID *string `json:"reference_example_id,omitempty"`
	ParentRunID     *string   `json:"parent_run_id,omitempty"`
	IsRoot          *bool     `json:"is_root,omitempty"`
	Error           *bool     `json:"error,omitempty"`
	ExecutionOrder  *int      `json:"execution_order,omitempty"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Filter          *string   `json:"filter,omitempty"`
	Query           *string   `json:"query,omitempty"`
	Limit           *int      `json:"limit,omitempty"`
	Offset          int       `json:"offset,omitempty"`
	OrderBy         *string   `json:"order_by,omitempty"`
	Tags            []string  `json:"tag,omitempty"`
	Select          []string  `json:"select,omitempty"`
}

// BatchIngestRequest represents a batch of runs to ingest.
type BatchIngestRequest struct {
	Post  []RunCreate `json:"post,omitempty"`
	Patch []RunUpdate `json:"patch,omitempty"`
}

// Dataset represents a collection of examples in LangSmith.
type Dataset struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      *string                `json:"description,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	ModifiedAt       *time.Time             `json:"modified_at,omitempty"`
	DataType         DataType               `json:"data_type,omitempty"`
	ExampleCount     int                    `json:"example_count,omitempty"`
	InputsSchema     map[string]interface{} `json:"inputs_schema,omitempty"`
	OutputsSchema    map[string]interface{} `json:"outputs_schema,omitempty"`
	Transformations  []interface{}          `json:"transformations,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	TenantID         string                 `json:"tenant_id,omitempty"`
	LastSessionID    *string                `json:"last_session_id,omitempty"`
}

// DatasetCreate represents the payload for creating a dataset.
type DatasetCreate struct {
	Name          string                 `json:"name"`
	Description   *string                `json:"description,omitempty"`
	DataType      DataType               `json:"data_type,omitempty"`
	InputsSchema  map[string]interface{} `json:"inputs_schema,omitempty"`
	OutputsSchema map[string]interface{} `json:"outputs_schema,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// DatasetUpdate represents the payload for updating a dataset.
type DatasetUpdate struct {
	Name          *string                `json:"name,omitempty"`
	Description   *string                `json:"description,omitempty"`
	InputsSchema  map[string]interface{} `json:"inputs_schema,omitempty"`
	OutputsSchema map[string]interface{} `json:"outputs_schema,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Example represents a single example in a dataset.
type Example struct {
	ID           string                 `json:"id"`
	DatasetID    string                 `json:"dataset_id"`
	CreatedAt    time.Time              `json:"created_at"`
	ModifiedAt   *time.Time             `json:"modified_at,omitempty"`
	Inputs       map[string]interface{} `json:"inputs"`
	Outputs      map[string]interface{} `json:"outputs,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SourceRunID  *string                `json:"source_run_id,omitempty"`
	Attachments  map[string]Attachment   `json:"attachments,omitempty"`
}

// ExampleCreate represents the payload for creating an example.
type ExampleCreate struct {
	DatasetID    string                 `json:"dataset_id"`
	Inputs       map[string]interface{} `json:"inputs"`
	Outputs      map[string]interface{} `json:"outputs,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SourceRunID  *string                `json:"source_run_id,omitempty"`
	ID           *string                `json:"id,omitempty"`
	CreatedAt    *time.Time             `json:"created_at,omitempty"`
}

// ExampleUpdate represents the payload for updating an example.
type ExampleUpdate struct {
	Inputs   map[string]interface{} `json:"inputs,omitempty"`
	Outputs  map[string]interface{} `json:"outputs,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ListExamplesOptions contains options for listing examples.
type ListExamplesOptions struct {
	DatasetID   *string    `json:"dataset_id,omitempty"`
	DatasetName *string    `json:"dataset_name,omitempty"`
	AsOf        *time.Time `json:"as_of,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Filter      *string    `json:"filter,omitempty"`
	Limit       *int       `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// FeedbackSource represents the source of feedback.
type FeedbackSource struct {
	Type     FeedbackSourceType `json:"type"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Feedback represents feedback on a run.
type Feedback struct {
	ID             string                 `json:"id"`
	RunID          *string                `json:"run_id,omitempty"`
	TraceID        *string                `json:"trace_id,omitempty"`
	Key            string                 `json:"key"`
	Score          *float64               `json:"score,omitempty"`
	Value          interface{}            `json:"value,omitempty"`
	Comment        *string                `json:"comment,omitempty"`
	Correction     interface{}            `json:"correction,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	ModifiedAt     *time.Time             `json:"modified_at,omitempty"`
	FeedbackSource *FeedbackSource        `json:"feedback_source,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

// FeedbackCreate represents the payload for creating feedback.
type FeedbackCreate struct {
	RunID          *string         `json:"run_id,omitempty"`
	TraceID        *string         `json:"trace_id,omitempty"`
	Key            string          `json:"key"`
	Score          *float64        `json:"score,omitempty"`
	Value          interface{}     `json:"value,omitempty"`
	Comment        *string         `json:"comment,omitempty"`
	Correction     interface{}     `json:"correction,omitempty"`
	FeedbackSource *FeedbackSource `json:"feedback_source,omitempty"`
	ID             *string         `json:"id,omitempty"`
}

// FeedbackUpdate represents the payload for updating feedback.
type FeedbackUpdate struct {
	Score      *float64    `json:"score,omitempty"`
	Value      interface{} `json:"value,omitempty"`
	Comment    *string     `json:"comment,omitempty"`
	Correction interface{} `json:"correction,omitempty"`
}

// TracerSession represents a project/session in LangSmith.
type TracerSession struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	StartTime          time.Time  `json:"start_time"`
	EndTime            *time.Time `json:"end_time,omitempty"`
	TenantID           string     `json:"tenant_id,omitempty"`
	ReferenceDatasetID *string    `json:"reference_dataset_id,omitempty"`
	ExtraJSON          map[string]interface{} `json:"extra,omitempty"`
	RunCount           *int       `json:"run_count,omitempty"`
	LatencyP50         *float64   `json:"latency_p50,omitempty"`
	LatencyP99         *float64   `json:"latency_p99,omitempty"`
	ErrorRate          *float64   `json:"error_rate,omitempty"`
	FeedbackStats      map[string]interface{} `json:"feedback_stats,omitempty"`
	LastRunStartTime   *time.Time `json:"last_run_start_time,omitempty"`
}

// TracerSessionCreate represents the payload for creating a project.
type TracerSessionCreate struct {
	Name               string                 `json:"name"`
	Description        *string                `json:"description,omitempty"`
	ReferenceDatasetID *string                `json:"reference_dataset_id,omitempty"`
	Extra              map[string]interface{} `json:"extra,omitempty"`
}

// TracerSessionUpdate represents the payload for updating a project.
type TracerSessionUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Prompt represents a prompt in the LangSmith prompt hub.
type Prompt struct {
	RepoHandle  string    `json:"repo_handle"`
	Description *string   `json:"description,omitempty"`
	Readme      *string   `json:"readme,omitempty"`
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsPublic    bool      `json:"is_public"`
	IsArchived  bool      `json:"is_archived"`
	Tags        []string  `json:"tags,omitempty"`
	FullName    string    `json:"full_name,omitempty"`
	NumLikes    int       `json:"num_likes,omitempty"`
	NumDownloads int      `json:"num_downloads,omitempty"`
	NumViews    int       `json:"num_views,omitempty"`
	LikedByAuthUser bool `json:"liked_by_auth_user,omitempty"`
	LastCommitHash *string `json:"last_commit_hash,omitempty"`
}

// PromptCommit represents a specific commit/version of a prompt.
type PromptCommit struct {
	Owner      string          `json:"owner"`
	Repo       string          `json:"repo"`
	CommitHash string          `json:"commit_hash"`
	Manifest   json.RawMessage `json:"manifest,omitempty"`
	Examples   []interface{}   `json:"examples,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

// PromptSortField represents the field to sort prompts by.
type PromptSortField string

const (
	PromptSortByNumDownloads PromptSortField = "num_downloads"
	PromptSortByNumViews     PromptSortField = "num_views"
	PromptSortByUpdatedAt    PromptSortField = "updated_at"
	PromptSortByNumLikes     PromptSortField = "num_likes"
)

// ListPromptsOptions contains options for listing prompts.
type ListPromptsOptions struct {
	IsPublic   *bool            `json:"is_public,omitempty"`
	IsArchived *bool            `json:"is_archived,omitempty"`
	SortField  *PromptSortField `json:"sort_field,omitempty"`
	SortDirection *string       `json:"sort_direction,omitempty"`
	Query      *string          `json:"query,omitempty"`
	Tags       []string         `json:"tags,omitempty"`
	Limit      *int             `json:"limit,omitempty"`
	Offset     int              `json:"offset,omitempty"`
}

// ListPromptsResponse is the response from listing prompts.
type ListPromptsResponse struct {
	Repos []Prompt `json:"repos"`
	Total int      `json:"total"`
}

// AnnotationQueue represents an annotation queue.
type AnnotationQueue struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TenantID    string    `json:"tenant_id,omitempty"`
}

// AnnotationQueueCreate represents the payload for creating an annotation queue.
type AnnotationQueueCreate struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// AnnotationQueueRunSchema represents a run in an annotation queue.
type AnnotationQueueRunSchema struct {
	QueueID       string    `json:"queue_id"`
	RunID         string    `json:"run_id"`
	AddedAt       time.Time `json:"added_at,omitempty"`
	LastReviewedTime *time.Time `json:"last_reviewed_time,omitempty"`
}

// SharedRunURL represents a shared link for a run.
type SharedRunURL struct {
	RunID    string `json:"run_id"`
	ShareToken string `json:"share_token"`
}

// ServerInfo represents the LangSmith server info.
type ServerInfo struct {
	Version        string `json:"version,omitempty"`
	LicenseExpiration *time.Time `json:"license_expiration_time,omitempty"`
	BatchIngestConfig *BatchIngestConfig `json:"batch_ingest_config,omitempty"`
}

// BatchIngestConfig contains server configuration for batch ingestion.
type BatchIngestConfig struct {
	SizeLimit       int `json:"size_limit,omitempty"`
	SizeLimitBytes  *int `json:"size_limit_bytes,omitempty"`
	ScaleUpQSizeLimit int `json:"scale_up_qsize_limit,omitempty"`
	ScaleUpNThreadsLimit int `json:"scale_up_nthreads_limit,omitempty"`
	ScaleDownNemptyTrigger int `json:"scale_down_nempty_trigger,omitempty"`
}

// PaginatedResponse is a generic paginated response.
type PaginatedResponse[T any] struct {
	Items   []T  `json:"items,omitempty"`
	Total   *int `json:"total,omitempty"`
	Cursors map[string]string `json:"cursors,omitempty"`
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string { return &s }

// IntPtr returns a pointer to the given int.
func IntPtr(i int) *int { return &i }

// Float64Ptr returns a pointer to the given float64.
func Float64Ptr(f float64) *float64 { return &f }

// BoolPtr returns a pointer to the given bool.
func BoolPtr(b bool) *bool { return &b }

// TimePtr returns a pointer to the given time.
func TimePtr(t time.Time) *time.Time { return &t }
