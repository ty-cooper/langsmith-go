package langsmith

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ty-cooper/langsmith-go/internal"
)

// uuidPattern matches UUID v4/v7 format strings.
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// dottedOrderSegment matches a single dotted-order segment: YYYYMMDDTHHMMSSffffffZ<uuid>.
var dottedOrderSegment = regexp.MustCompile(`^\d{8}T\d{6}\d{6}Z[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// isValidDottedOrder checks that s is a well-formed dotted order string
// consisting of one or more dot-separated segments.
func isValidDottedOrder(s string) bool {
	if s == "" || len(s) > 1024 {
		return false
	}
	for _, seg := range strings.Split(s, ".") {
		if !dottedOrderSegment.MatchString(seg) {
			return false
		}
	}
	return true
}

// RunTree represents a hierarchical execution trace.
// It manages parent-child relationships and can post/patch runs
// to LangSmith via a background batch worker.
//
// The exported methods (SetInputs, SetOutputs, AddMetadata, etc.) are safe
// for concurrent use. However, direct field access (e.g. rt.Outputs["k"] = v)
// is NOT protected by the mutex — use the provided methods instead.
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
		Inputs:             copyMap(rt.Inputs),
		Outputs:            copyMap(rt.Outputs),
		ParentRunID:        rt.ParentRunID,
		TraceID:            rt.TraceID,
		DottedOrder:        rt.DottedOrder,
		SessionName:        rt.SessionName,
		SessionID:          rt.SessionID,
		ReferenceExampleID: rt.ReferenceExampleID,
		Tags:               copySlice(rt.Tags),
		Metadata:           copyMap(rt.Metadata),
		Events:             copyEvents(rt.Events),
		Serialized:         copyMap(rt.Serialized),
		Extra:              copyMap(rt.Extra),
		Attachments:        copyAttachments(rt.Attachments),
	}
}

func (rt *RunTree) buildUpdateLocked() RunUpdate {
	return RunUpdate{
		EndTime:  rt.EndTime,
		Error:    rt.Error,
		Outputs:  copyMap(rt.Outputs),
		Events:   copyEvents(rt.Events),
		Tags:     copySlice(rt.Tags),
		Metadata: copyMap(rt.Metadata),
		Extra:    copyMap(rt.Extra),
	}
}

// copyMap returns a shallow copy of a map[string]any. Returns nil for nil input.
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// copySlice returns a copy of a string slice. Returns nil for nil input.
func copySlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}

// copyEvents returns a copy of an events slice. Returns nil for nil input.
func copyEvents(events []map[string]any) []map[string]any {
	if events == nil {
		return nil
	}
	out := make([]map[string]any, len(events))
	for i, e := range events {
		out[i] = copyMap(e)
	}
	return out
}

// copyAttachments returns a copy of an attachments map. Returns nil for nil input.
func copyAttachments(m map[string]Attachment) map[string]Attachment {
	if m == nil {
		return nil
	}
	out := make(map[string]Attachment, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
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
// Values are percent-encoded per the W3C Baggage specification.
func (rt *RunTree) ToHeaders() http.Header {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	h := http.Header{}
	h.Set(HeaderParentID, rt.ID)

	parts := []string{
		"langsmith-trace_id=" + url.QueryEscape(rt.TraceID),
		"langsmith-dotted_order=" + url.QueryEscape(rt.DottedOrder),
	}
	if rt.SessionName != "" {
		parts = append(parts, "langsmith-session_name="+url.QueryEscape(rt.SessionName))
	}
	if rt.SessionID != nil {
		parts = append(parts, "langsmith-session_id="+url.QueryEscape(*rt.SessionID))
	}
	h.Set(HeaderBaggage, strings.Join(parts, ","))

	return h
}

// RunTreeFromHeaders reconstructs minimal RunTree context from HTTP headers.
// The returned RunTree can be used as a parent for CreateChild.
// Returns nil if the headers are missing or contain invalid trace IDs.
func RunTreeFromHeaders(headers http.Header, client *Client) *RunTree {
	parentID := headers.Get(HeaderParentID)
	if parentID == "" {
		return nil
	}
	if !isValidUUID(parentID) {
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
			val, err := url.QueryUnescape(kv[1])
			if err != nil {
				continue
			}
			switch kv[0] {
			case "langsmith-trace_id":
				if isValidUUID(val) {
					rt.TraceID = val
				}
			case "langsmith-dotted_order":
				if isValidDottedOrder(val) {
					rt.DottedOrder = val
				}
			case "langsmith-session_name":
				rt.SessionName = val
			case "langsmith-session_id":
				if isValidUUID(val) {
					rt.SessionID = &val
				}
			}
		}
	}

	// If trace ID wasn't provided or was invalid, fall back to parent ID.
	if rt.TraceID == "" {
		rt.TraceID = parentID
	}

	return rt
}

// isValidUUID checks that s matches the standard UUID format.
func isValidUUID(s string) bool {
	return uuidPattern.MatchString(s)
}
