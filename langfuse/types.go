package langfuse

import "time"

// EventType represents the type of event being tracked
type EventType string

const (
	EventTypeTraceCreate      EventType = "trace-create"
	EventTypeScoreCreate      EventType = "score-create"
	EventTypeEventCreate      EventType = "event-create"
	EventTypeSpanCreate       EventType = "span-create"
	EventTypeSpanUpdate       EventType = "span-update"
	EventTypeGenerationCreate EventType = "generation-create"
	EventTypeGenerationUpdate EventType = "generation-update"
	EventTypeAgentCreate      EventType = "agent-create"
	EventTypeToolCreate       EventType = "tool-create"
	EventTypeChainCreate      EventType = "chain-create"
	EventTypeRetrieverCreate  EventType = "retriever-create"
	EventTypeEvaluatorCreate  EventType = "evaluator-create"
	EventTypeEmbeddingCreate  EventType = "embedding-create"
	EventTypeGuardrailCreate  EventType = "guardrail-create"
	EventTypeSdkLog           EventType = "sdk-log"
)

// ObservationLevel represents the severity level of an observation
type ObservationLevel string

const (
	LevelDebug   ObservationLevel = "DEBUG"
	LevelDefault ObservationLevel = "DEFAULT"
	LevelWarning ObservationLevel = "WARNING"
	LevelError   ObservationLevel = "ERROR"
)

// Event represents a single event in the ingestion batch
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Body      map[string]interface{} `json:"body"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// IngestionRequest represents the batch ingestion request
type IngestionRequest struct {
	Batch    []Event                `json:"batch"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// IngestionResponse represents the response from ingestion API
type IngestionResponse struct {
	Successes []SuccessResult `json:"successes,omitempty"`
	Errors    []ErrorResult   `json:"errors,omitempty"`
}

// SuccessResult represents a successful event ingestion
type SuccessResult struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
}

// ErrorResult represents a failed event ingestion
type ErrorResult struct {
	ID      string `json:"id,omitempty"`
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Usage represents token usage information
type Usage struct {
	Input      *int    `json:"input,omitempty"`
	Output     *int    `json:"output,omitempty"`
	Total      *int    `json:"total,omitempty"`
	Unit       *string `json:"unit,omitempty"`
	InputCost  *float64 `json:"inputCost,omitempty"`
	OutputCost *float64 `json:"outputCost,omitempty"`
	TotalCost  *float64 `json:"totalCost,omitempty"`
}
