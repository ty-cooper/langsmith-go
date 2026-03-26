package langsmith

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ty-cooper/langsmith-go/internal"
)

// RunTree represents a hierarchical execution trace.
// It manages parent-child relationships and can post/patch runs
// to LangSmith via a background batch worker.
//
// RunTree is safe for concurrent use from multiple goroutines.
type RunTree struct {
	mu sync.Mutex

	ID          string
	Name        string
	RunType     RunType
	StartTime   time.Time
	EndTime     *time.Time
	Inputs      map[string]any
	Outputs     map[string]any
	Error       *string
	Tags        []string
	Metadata    map[string]any
	Events      []map[string]any
	Extra       map[string]any
	Serialized  map[string]any
	ParentRunID *string
	TraceID     string
	DottedOrder string
	SessionName string
	SessionID   *string
	Attachments map[string]Attachment
	ReferenceExampleID *string

	client   *Client
	children []*RunTree
	posted   bool
}

// RunTreeOption configures a RunTree.
type RunTreeOption func(*RunTree)

// WithRunTreeInputs sets the initial inputs.
func WithRunTreeInputs(inputs map[string]any) RunTreeOption {
	return func(rt *RunTree) { rt.Inputs = inputs }
}

// WithRunTreeOutputs sets the initial outputs.
func WithRunTreeOutputs(outputs map[string]any) RunTreeOption {
	return func(rt *RunTree) { rt.Outputs = outputs }
}

// WithRunTreeMetadata sets metadata.
func WithRunTreeMetadata(metadata map[string]any) RunTreeOption {
	return func(rt *RunTree) { rt.Metadata = metadata }
}

// WithRunTreeTags sets tags.
func WithRunTreeTags(tags []string) RunTreeOption {
	return func(rt *RunTree) { rt.Tags = tags }
}

// WithRunTreeExtra sets extra data.
func WithRunTreeExtra(extra map[string]any) RunTreeOption {
	return func(rt *RunTree) { rt.Extra = extra }
}

// WithRunTreeSessionName sets the project/session name.
func WithRunTreeSessionName(name string) RunTreeOption {
	return func(rt *RunTree) { rt.SessionName = name }
}

// WithRunTreeClient sets the client for submitting runs.
func WithRunTreeClient(c *Client) RunTreeOption {
	return func(rt *RunTree) { rt.client = c }
}

// WithRunTreeReferenceExampleID sets the reference example ID.
func WithRunTreeReferenceExampleID(id string) RunTreeOption {
	return func(rt *RunTree) { rt.ReferenceExampleID = &id }
}

// NewRunTree creates a new root RunTree.
func NewRunTree(name string, runType RunType, opts ...RunTreeOption) *RunTree {
	id := internal.UUID7()
	now := time.Now().UTC()

	rt := &RunTree{
		ID:        id,
		Name:      name,
		RunType:   runType,
		StartTime: now,
		Metadata:  make(map[string]any),
	}

	for _, opt := range opts {
		opt(rt)
	}

	rt.TraceID = id
	rt.DottedOrder = internal.GenerateDottedOrder(now, id)

	if rt.SessionName == "" && rt.client != nil {
		rt.SessionName = rt.client.Project()
	}
	if rt.SessionName == "" {
		rt.SessionName = GetProject()
	}

	return rt
}

// CreateChild creates a child RunTree under this parent.
// Safe for concurrent use.
func (rt *RunTree) CreateChild(name string, runType RunType, opts ...RunTreeOption) *RunTree {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	id := internal.UUID7()
	now := time.Now().UTC()

	child := &RunTree{
		ID:          id,
		Name:        name,
		RunType:     runType,
		StartTime:   now,
		ParentRunID: &rt.ID,
		TraceID:     rt.TraceID,
		DottedOrder: internal.AppendDottedOrder(rt.DottedOrder, now, id),
		SessionName: rt.SessionName,
		SessionID:   rt.SessionID,
		Metadata:    make(map[string]any),
		client:      rt.client,
	}

	for _, opt := range opts {
		opt(child)
	}

	rt.children = append(rt.children, child)
	return child
}

// End finalizes the run with outputs and/or an error, then patches it.
func (rt *RunTree) End(opts ...EndOption) {
	rt.mu.Lock()
	now := time.Now().UTC()
	rt.EndTime = &now
	for _, opt := range opts {
		opt(rt)
	}
	rt.mu.Unlock()

	rt.PatchRun()
}

// EndOption configures the End call.
type EndOption func(*RunTree)

// WithEndOutputs sets the outputs when ending.
func WithEndOutputs(outputs map[string]any) EndOption {
	return func(rt *RunTree) { rt.Outputs = outputs }
}

// WithEndError sets the error when ending.
func WithEndError(err string) EndOption {
	return func(rt *RunTree) { rt.Error = &err }
}

// SetInputs sets the inputs on the run.
func (rt *RunTree) SetInputs(inputs map[string]any) {
	rt.mu.Lock()
	rt.Inputs = inputs
	rt.mu.Unlock()
}

// SetOutputs sets the outputs on the run.
func (rt *RunTree) SetOutputs(outputs map[string]any) {
	rt.mu.Lock()
	rt.Outputs = outputs
	rt.mu.Unlock()
}

// AddMetadata adds key-value pairs to the run metadata.
func (rt *RunTree) AddMetadata(kv map[string]any) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.Metadata == nil {
		rt.Metadata = make(map[string]any)
	}
	for k, v := range kv {
		rt.Metadata[k] = v
	}
}

// AddTags appends tags to the run.
func (rt *RunTree) AddTags(tags ...string) {
	rt.mu.Lock()
	rt.Tags = append(rt.Tags, tags...)
	rt.mu.Unlock()
}

// AddEvent appends an event to the run.
func (rt *RunTree) AddEvent(event map[string]any) {
	rt.mu.Lock()
	rt.Events = append(rt.Events, event)
	rt.mu.Unlock()
}

// PostRun submits the run creation to the batch worker.
func (rt *RunTree) PostRun() {
	rt.mu.Lock()
	if rt.client == nil || rt.posted {
		rt.mu.Unlock()
		return
	}
	rt.posted = true
	create := rt.buildCreateLocked()
	rt.mu.Unlock()

	rt.client.CreateRunBatched(create)
}

// PatchRun submits a run update to the batch worker.
func (rt *RunTree) PatchRun() {
	rt.mu.Lock()
	if rt.client == nil {
		rt.mu.Unlock()
		return
	}
	if !rt.posted {
		rt.posted = true
		create := rt.buildCreateLocked()
		rt.mu.Unlock()
		rt.client.CreateRunBatched(create)
		return
	}
	update := rt.buildUpdateLocked()
	id := rt.ID
	rt.mu.Unlock()

	rt.client.UpdateRunBatched(id, update)
}

// Children returns a copy of the child run trees.
func (rt *RunTree) Children() []*RunTree {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	out := make([]*RunTree, len(rt.children))
	copy(out, rt.children)
	return out
}

func (rt *RunTree) buildCreateLocked() RunCreate {
	return RunCreate{
		ID:                 rt.ID,
		Name:               rt.Name,
		RunType:            rt.RunType,
		StartTime:          rt.StartTime,
		EndTime:            rt.EndTime,
		Error:              rt.Error,
		Inputs:             rt.Inputs,
		Outputs:            rt.Outputs,
		ParentRunID:        rt.ParentRunID,
		TraceID:            rt.TraceID,
		DottedOrder:        rt.DottedOrder,
		SessionName:        rt.SessionName,
		SessionID:          rt.SessionID,
		ReferenceExampleID: rt.ReferenceExampleID,
		Tags:               rt.Tags,
		Metadata:           rt.Metadata,
		Events:             rt.Events,
		Serialized:         rt.Serialized,
		Extra:              rt.Extra,
		Attachments:        rt.Attachments,
	}
}

func (rt *RunTree) buildUpdateLocked() RunUpdate {
	return RunUpdate{
		EndTime:  rt.EndTime,
		Error:    rt.Error,
		Outputs:  rt.Outputs,
		Events:   rt.Events,
		Tags:     rt.Tags,
		Metadata: rt.Metadata,
		Extra:    rt.Extra,
	}
}

// --- Header Serialization ---
// These allow propagating trace context across service boundaries via HTTP headers.

const (
	// HeaderParentID is the header key for the parent run ID.
	HeaderParentID = "langsmith-trace"
	// HeaderBaggage carries additional trace context.
	HeaderBaggage = "baggage"
)

// ToHeaders serializes the RunTree trace context into HTTP headers.
func (rt *RunTree) ToHeaders() http.Header {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	h := http.Header{}
	h.Set(HeaderParentID, rt.ID)

	parts := []string{
		"langsmith-trace_id=" + rt.TraceID,
		"langsmith-dotted_order=" + rt.DottedOrder,
	}
	if rt.SessionName != "" {
		parts = append(parts, "langsmith-session_name="+rt.SessionName)
	}
	if rt.SessionID != nil {
		parts = append(parts, "langsmith-session_id="+*rt.SessionID)
	}
	h.Set(HeaderBaggage, strings.Join(parts, ","))

	return h
}

// RunTreeFromHeaders reconstructs minimal RunTree context from HTTP headers.
// The returned RunTree can be used as a parent for CreateChild.
func RunTreeFromHeaders(headers http.Header, client *Client) *RunTree {
	parentID := headers.Get(HeaderParentID)
	if parentID == "" {
		return nil
	}

	rt := &RunTree{
		ID:     parentID,
		client: client,
	}

	baggage := headers.Get(HeaderBaggage)
	if baggage != "" {
		for _, part := range strings.Split(baggage, ",") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) != 2 {
				continue
			}
			switch kv[0] {
			case "langsmith-trace_id":
				rt.TraceID = kv[1]
			case "langsmith-dotted_order":
				rt.DottedOrder = kv[1]
			case "langsmith-session_name":
				rt.SessionName = kv[1]
			case "langsmith-session_id":
				rt.SessionID = &kv[1]
			}
		}
	}

	return rt
}
