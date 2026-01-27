package langfuse

import (
	"time"
)

// ScoreParams contains parameters for creating a score
type ScoreParams struct {
	// ID is the unique identifier (auto-generated if not provided)
	ID *string

	// TraceID is the ID of the trace being scored (required if ObservationID not set)
	TraceID *string

	// ObservationID is the ID of the observation being scored (optional, for granular scoring)
	ObservationID *string

	// Name is the name/identifier of the score (required)
	Name string

	// Value is the numeric score value (required)
	Value float64

	// Comment is an optional comment about the score
	Comment *string

	// DataType is the type of score (default: "NUMERIC", can be "CATEGORICAL", "BOOLEAN")
	DataType *string

	// ConfigID links the score to a score config
	ConfigID *string
}

// CreateScore creates a new score for a trace or observation
func (c *Client) CreateScore(params ScoreParams) (string, error) {
	id := generateID()
	if params.ID != nil {
		id = *params.ID
	}

	body := scoreToBody(params, id)

	event := Event{
		ID:        generateID(),
		Type:      EventTypeScoreCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateScore creates a new score for this trace
func (t *Trace) CreateScore(params ScoreParams) (string, error) {
	params.TraceID = &t.id
	return t.client.CreateScore(params)
}

// scoreToBody converts score params to event body
func scoreToBody(params ScoreParams, id string) map[string]interface{} {
	body := make(map[string]interface{})

	body["id"] = id
	body["name"] = params.Name
	body["value"] = params.Value

	if params.TraceID != nil {
		body["traceId"] = *params.TraceID
	}

	if params.ObservationID != nil {
		body["observationId"] = *params.ObservationID
	}

	if params.Comment != nil {
		body["comment"] = *params.Comment
	}

	if params.DataType != nil {
		body["dataType"] = *params.DataType
	} else {
		body["dataType"] = "NUMERIC"
	}

	if params.ConfigID != nil {
		body["configId"] = *params.ConfigID
	}

	return body
}
