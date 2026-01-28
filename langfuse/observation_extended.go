package langfuse

import (
	"time"
)

// CreateAgent creates a new agent observation
func (t *Trace) CreateAgent(params AgentParams) (string, error) {
	return t.client.CreateAgent(t.id, params)
}

// CreateAgent creates a new agent observation
func (c *Client) CreateAgent(traceID string, params AgentParams) (string, error) {
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
		Type:      EventTypeAgentCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateTool creates a new tool observation
func (t *Trace) CreateTool(params ToolParams) (string, error) {
	return t.client.CreateTool(t.id, params)
}

// CreateTool creates a new tool observation
func (c *Client) CreateTool(traceID string, params ToolParams) (string, error) {
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
		Type:      EventTypeToolCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateChain creates a new chain observation
func (t *Trace) CreateChain(params ChainParams) (string, error) {
	return t.client.CreateChain(t.id, params)
}

// CreateChain creates a new chain observation
func (c *Client) CreateChain(traceID string, params ChainParams) (string, error) {
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
		Type:      EventTypeChainCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateRetriever creates a new retriever observation
func (t *Trace) CreateRetriever(params RetrieverParams) (string, error) {
	return t.client.CreateRetriever(t.id, params)
}

// CreateRetriever creates a new retriever observation
func (c *Client) CreateRetriever(traceID string, params RetrieverParams) (string, error) {
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
		Type:      EventTypeRetrieverCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateEvaluator creates a new evaluator observation
func (t *Trace) CreateEvaluator(params EvaluatorParams) (string, error) {
	return t.client.CreateEvaluator(t.id, params)
}

// CreateEvaluator creates a new evaluator observation
func (c *Client) CreateEvaluator(traceID string, params EvaluatorParams) (string, error) {
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
		Type:      EventTypeEvaluatorCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateEmbedding creates a new embedding observation
func (t *Trace) CreateEmbedding(params EmbeddingParams) (string, error) {
	return t.client.CreateEmbedding(t.id, params)
}

// CreateEmbedding creates a new embedding observation
func (c *Client) CreateEmbedding(traceID string, params EmbeddingParams) (string, error) {
	id := generateID()
	if params.ID != nil {
		id = *params.ID
	}

	params.TraceID = traceID
	body := observationToBody(params.ObservationParams, id)

	if params.EndTime != nil {
		body["endTime"] = params.EndTime.Format(time.RFC3339Nano)
	}

	if params.EmbeddingModel != nil {
		body["model"] = *params.EmbeddingModel
	}

	if params.EmbeddingModelParameters != nil {
		body["modelParameters"] = params.EmbeddingModelParameters
	}

	event := Event{
		ID:        generateID(),
		Type:      EventTypeEmbeddingCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateGuardrail creates a new guardrail observation
func (t *Trace) CreateGuardrail(params GuardrailParams) (string, error) {
	return t.client.CreateGuardrail(t.id, params)
}

// CreateGuardrail creates a new guardrail observation
func (c *Client) CreateGuardrail(traceID string, params GuardrailParams) (string, error) {
	id := generateID()
	if params.ID != nil {
		id = *params.ID
	}

	params.TraceID = traceID
	body := observationToBody(params.ObservationParams, id)

	event := Event{
		ID:        generateID(),
		Type:      EventTypeGuardrailCreate,
		Timestamp: time.Now(),
		Body:      body,
	}

	if err := c.enqueue(event); err != nil {
		return "", err
	}

	return id, nil
}

// CreateSdkLog creates a new SDK log event
func (c *Client) CreateSdkLog(params SdkLogParams) error {
	body := map[string]interface{}{
		"log": params.Log,
	}

	event := Event{
		ID:        generateID(),
		Type:      EventTypeSdkLog,
		Timestamp: time.Now(),
		Body:      body,
	}

	return c.enqueue(event)
}

// UpdateTool updates an existing tool observation
func (c *Client) UpdateTool(toolID string, params ToolParams) error {
	body := observationToBody(params.ObservationParams, toolID)

	if params.EndTime != nil {
		body["endTime"] = params.EndTime.Format(time.RFC3339Nano)
	}

	event := Event{
		ID:        generateID(),
		Type:      EventTypeSpanUpdate,  // Tool 是 Span 的一种，使用 span-update
		Timestamp: time.Now(),
		Body:      body,
	}

	return c.enqueue(event)
}
