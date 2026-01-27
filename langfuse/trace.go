package langfuse

import (
	"time"
)

// TraceParams contains parameters for creating a trace
type TraceParams struct {
	// ID is the unique identifier for the trace (auto-generated if not provided)
	ID *string

	// Name is the name of the trace
	Name *string

	// Timestamp is when the trace started (defaults to now)
	Timestamp *time.Time

	// Input is the input data for the trace
	Input interface{}

	// Output is the output data for the trace
	Output interface{}

	// Metadata is additional metadata for the trace
	Metadata map[string]interface{}

	// UserID is the user identifier
	UserID *string

	// SessionID is the session identifier
	SessionID *string

	// Environment is the environment name (default: "default")
	Environment *string

	// Version is the version string
	Version *string

	// Release is the release string
	Release *string

	// Tags are tags for categorization
	Tags []string

	// Public indicates if the trace is publicly accessible
	Public *bool
}

// Trace represents a trace object
type Trace struct {
	client *Client
	id     string
	params TraceParams
}

// CreateTrace creates a new trace
func (c *Client) CreateTrace(params TraceParams) (*Trace, error) {
	// Generate ID if not provided
	id := generateID()
	if params.ID != nil {
		id = *params.ID
	}

	trace := &Trace{
		client: c,
		id:     id,
		params: params,
	}

	// Create trace event
	event := Event{
		ID:        generateID(),
		Type:      EventTypeTraceCreate,
		Timestamp: time.Now(),
		Body:      trace.toBody(),
	}

	if err := c.enqueue(event); err != nil {
		return nil, err
	}

	return trace, nil
}

// toBody converts trace params to event body
func (t *Trace) toBody() map[string]interface{} {
	body := make(map[string]interface{})

	body["id"] = t.id

	if t.params.Name != nil {
		body["name"] = *t.params.Name
	}

	if t.params.Timestamp != nil {
		body["timestamp"] = t.params.Timestamp.Format(time.RFC3339Nano)
	}

	if t.params.Input != nil {
		body["input"] = t.params.Input
	}

	if t.params.Output != nil {
		body["output"] = t.params.Output
	}

	if t.params.Metadata != nil {
		body["metadata"] = t.params.Metadata
	}

	if t.params.UserID != nil {
		body["userId"] = *t.params.UserID
	}

	if t.params.SessionID != nil {
		body["sessionId"] = *t.params.SessionID
	}

	if t.params.Environment != nil {
		body["environment"] = *t.params.Environment
	}

	if t.params.Version != nil {
		body["version"] = *t.params.Version
	}

	if t.params.Release != nil {
		body["release"] = *t.params.Release
	}

	if t.params.Tags != nil && len(t.params.Tags) > 0 {
		body["tags"] = t.params.Tags
	}

	if t.params.Public != nil {
		body["public"] = *t.params.Public
	}

	return body
}

// ID returns the trace ID
func (t *Trace) ID() string {
	return t.id
}

// Update updates the trace with new parameters
func (t *Trace) Update(params TraceParams) error {
	// Merge params
	if params.Name != nil {
		t.params.Name = params.Name
	}
	if params.Input != nil {
		t.params.Input = params.Input
	}
	if params.Output != nil {
		t.params.Output = params.Output
	}
	if params.Metadata != nil {
		if t.params.Metadata == nil {
			t.params.Metadata = make(map[string]interface{})
		}
		for k, v := range params.Metadata {
			t.params.Metadata[k] = v
		}
	}
	if params.UserID != nil {
		t.params.UserID = params.UserID
	}
	if params.SessionID != nil {
		t.params.SessionID = params.SessionID
	}
	if params.Tags != nil {
		t.params.Tags = params.Tags
	}
	if params.Public != nil {
		t.params.Public = params.Public
	}

	// Send updated trace event
	event := Event{
		ID:        generateID(),
		Type:      EventTypeTraceCreate,
		Timestamp: time.Now(),
		Body:      t.toBody(),
	}

	return t.client.enqueue(event)
}
