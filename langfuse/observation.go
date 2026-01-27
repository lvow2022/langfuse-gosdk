package langfuse

import (
	"time"
)

// ObservationParams contains common parameters for observations
type ObservationParams struct {
	// ID is the unique identifier (auto-generated if not provided)
	ID *string

	// TraceID is the ID of the parent trace (required)
	TraceID string

	// ParentObservationID is the ID of the parent observation (for nesting)
	ParentObservationID *string

	// Name is the name of the observation
	Name *string

	// StartTime is when the observation started (defaults to now)
	StartTime *time.Time

	// Metadata is additional metadata
	Metadata map[string]interface{}

	// Input is the input data
	Input interface{}

	// Output is the output data
	Output interface{}

	// Level is the severity level
	Level *ObservationLevel

	// StatusMessage is a status message
	StatusMessage *string

	// Version is the version string
	Version *string

	// Environment is the environment name
	Environment *string
}

// SpanParams contains parameters for creating a span
type SpanParams struct {
	ObservationParams

	// EndTime is when the span ended
	EndTime *time.Time
}

// EventParams contains parameters for creating an event
type EventParams struct {
	ObservationParams
}

// GenerationParams contains parameters for creating a generation
type GenerationParams struct {
	SpanParams

	// Model is the model name/identifier
	Model *string

	// ModelParameters are parameters passed to the model
	ModelParameters map[string]interface{}

	// Usage contains token usage information
	Usage *Usage

	// PromptName is the name of the prompt used
	PromptName *string

	// PromptVersion is the version of the prompt
	PromptVersion *int

	// CompletionStartTime is when the completion started streaming
	CompletionStartTime *time.Time
}

// AgentParams contains parameters for creating an agent observation
type AgentParams struct {
	SpanParams
}

// ToolParams contains parameters for creating a tool observation
type ToolParams struct {
	SpanParams
}

// ChainParams contains parameters for creating a chain observation
type ChainParams struct {
	SpanParams
}

// RetrieverParams contains parameters for creating a retriever observation
type RetrieverParams struct {
	SpanParams
}

// EvaluatorParams contains parameters for creating an evaluator observation
type EvaluatorParams struct {
	SpanParams
}

// EmbeddingParams contains parameters for creating an embedding observation
type EmbeddingParams struct {
	SpanParams

	// EmbeddingModel is the embedding model name/identifier
	EmbeddingModel *string

	// EmbeddingModelParameters are parameters passed to the embedding model
	EmbeddingModelParameters map[string]interface{}
}

// GuardrailParams contains parameters for creating a guardrail observation
type GuardrailParams struct {
	ObservationParams
}

// SdkLogParams contains parameters for creating SDK log events
type SdkLogParams struct {
	// Log is the log data (any JSON value)
	Log interface{}
}

// CreateSpan creates a new span observation
func (t *Trace) CreateSpan(params SpanParams) (string, error) {
	return t.client.CreateSpan(t.id, params)
}

// CreateSpan creates a new span observation
func (c *Client) CreateSpan(traceID string, params SpanParams) (string, error) {
	id := generateID()
	if params.ID != nil {
		id = *params.ID
	}

	params.TraceID = traceID

	body := observationToBody(params.ObservationParams, id)

	if params.EndTime != nil {
		body["endTime"] = params.EndTime.Format(time.RFC3339Nano)
	}

	event := Event{
		ID:        generateID(),
		Type:      EventTypeSpanCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateEvent creates a new event observation
func (t *Trace) CreateEvent(params EventParams) (string, error) {
	return t.client.CreateEvent(t.id, params)
}

// CreateEvent creates a new event observation
func (c *Client) CreateEvent(traceID string, params EventParams) (string, error) {
	id := generateID()
	if params.ID != nil {
		id = *params.ID
	}

	params.TraceID = traceID

	body := observationToBody(params.ObservationParams, id)

	event := Event{
		ID:        generateID(),
		Type:      EventTypeEventCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateGeneration creates a new generation observation
func (t *Trace) CreateGeneration(params GenerationParams) (string, error) {
	return t.client.CreateGeneration(t.id, params)
}

// CreateGeneration creates a new generation observation
func (c *Client) CreateGeneration(traceID string, params GenerationParams) (string, error) {
	id := generateID()
	if params.ID != nil {
		id = *params.ID
	}

	params.TraceID = traceID

	body := observationToBody(params.ObservationParams, id)

	if params.EndTime != nil {
		body["endTime"] = params.EndTime.Format(time.RFC3339Nano)
	}

	if params.Model != nil {
		body["model"] = *params.Model
	}

	if params.ModelParameters != nil {
		body["modelParameters"] = params.ModelParameters
	}

	if params.Usage != nil {
		body["usage"] = params.Usage
	}

	if params.PromptName != nil {
		body["promptName"] = *params.PromptName
	}

	if params.PromptVersion != nil {
		body["promptVersion"] = *params.PromptVersion
	}

	if params.CompletionStartTime != nil {
		body["completionStartTime"] = params.CompletionStartTime.Format(time.RFC3339Nano)
	}

	event := Event{
		ID:        generateID(),
		Type:      EventTypeGenerationCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// UpdateSpan updates an existing span
func (c *Client) UpdateSpan(spanID string, params SpanParams) error {
	body := observationToBody(params.ObservationParams, spanID)

	if params.EndTime != nil {
		body["endTime"] = params.EndTime.Format(time.RFC3339Nano)
	}

	event := Event{
		ID:        generateID(),
		Type:      EventTypeSpanUpdate,
		Timestamp: time.Now(),
		Body:      body,
	}

	return c.enqueue(event)
}

// UpdateGeneration updates an existing generation
func (c *Client) UpdateGeneration(generationID string, params GenerationParams) error {
	body := observationToBody(params.ObservationParams, generationID)

	if params.EndTime != nil {
		body["endTime"] = params.EndTime.Format(time.RFC3339Nano)
	}

	if params.Model != nil {
		body["model"] = *params.Model
	}

	if params.ModelParameters != nil {
		body["modelParameters"] = params.ModelParameters
	}

	if params.Usage != nil {
		body["usage"] = params.Usage
	}

	if params.PromptName != nil {
		body["promptName"] = *params.PromptName
	}

	if params.PromptVersion != nil {
		body["promptVersion"] = *params.PromptVersion
	}

	if params.CompletionStartTime != nil {
		body["completionStartTime"] = params.CompletionStartTime.Format(time.RFC3339Nano)
	}

	event := Event{
		ID:        generateID(),
		Type:      EventTypeGenerationUpdate,
		Timestamp: time.Now(),
		Body:      body,
	}

	return c.enqueue(event)
}

// observationToBody converts observation params to event body
func observationToBody(params ObservationParams, id string) map[string]interface{} {
	body := make(map[string]interface{})

	body["id"] = id
	if params.TraceID != "" {
		body["traceId"] = params.TraceID
	}

	if params.ParentObservationID != nil {
		body["parentObservationId"] = *params.ParentObservationID
	}

	if params.Name != nil {
		body["name"] = *params.Name
	}

	if params.StartTime != nil {
		body["startTime"] = params.StartTime.Format(time.RFC3339Nano)
	}

	if params.Metadata != nil {
		body["metadata"] = params.Metadata
	}

	if params.Input != nil {
		body["input"] = params.Input
	}

	if params.Output != nil {
		body["output"] = params.Output
	}

	if params.Level != nil {
		body["level"] = string(*params.Level)
	}

	if params.StatusMessage != nil {
		body["statusMessage"] = *params.StatusMessage
	}

	if params.Version != nil {
		body["version"] = *params.Version
	}

	if params.Environment != nil {
		body["environment"] = *params.Environment
	}

	return body
}
