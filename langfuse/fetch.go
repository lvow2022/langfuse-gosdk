package langfuse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// TraceWithFullDetails represents a trace with all nested observations
type TraceWithFullDetails struct {
	ID           string                `json:"id"`
	Name         *string               `json:"name,omitempty"`
	UserID       *string               `json:"userId,omitempty"`
	SessionID    *string               `json:"sessionId,omitempty"`
	Timestamp    string                `json:"timestamp"`
	Input        interface{}           `json:"input,omitempty"`
	Output       interface{}           `json:"output,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Tags         []string              `json:"tags,omitempty"`
	Observations []ObservationDetails  `json:"observations,omitempty"`
	Scores       []ScoreData           `json:"scores,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for TraceWithFullDetails
// to handle cases where observations might be a string, null, or array
func (t *TraceWithFullDetails) UnmarshalJSON(data []byte) error {
	// Define a local type to avoid infinite recursion
	type Alias TraceWithFullDetails
	aux := &struct {
		Observations json.RawMessage `json:"observations"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle observations field
	if len(aux.Observations) > 0 && string(aux.Observations) != "null" {
		// Try to unmarshal as array first
		var obsArray []ObservationDetails
		if err := json.Unmarshal(aux.Observations, &obsArray); err != nil {
			// If that fails, it might be a string or other format
			// For now, just leave it empty
			t.Observations = nil
		} else {
			t.Observations = obsArray
		}
	}

	return nil
}

// ScoreData represents a score retrieved from API
type ScoreData struct {
	ID            string   `json:"id"`
	TraceID       string   `json:"traceId"`
	ObservationID *string  `json:"observationId,omitempty"`
	Name          string   `json:"name"`
	Value         float64  `json:"value"`
	Comment       *string  `json:"comment,omitempty"`
	DataType      string   `json:"dataType"`
	ConfigID      *string  `json:"configId,omitempty"`
	Timestamp     string   `json:"timestamp"`
}

// ObservationDetails represents an observation (span, generation, event, tool)
type ObservationDetails struct {
	ID                string         `json:"id"`
	TraceID           string         `json:"traceId"`
	Type              string         `json:"type"` // SPAN, GENERATION, EVENT, TOOL
	Name              *string        `json:"name,omitempty"`
	StartTime         string         `json:"startTime"`
	EndTime           *string        `json:"endTime,omitempty"`
	CompletionStartTime *string      `json:"completionStartTime,omitempty"`
	Input             interface{}    `json:"input,omitempty"`
	Output            interface{}    `json:"output,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Level             *string        `json:"level,omitempty"`
	StatusMessage     *string        `json:"statusMessage,omitempty"`
	ParentObservationID *string      `json:"parentObservationId,omitempty"`
	Version           *string        `json:"version,omitempty"`
	Model             *string        `json:"model,omitempty"`
	ModelParameters   map[string]interface{} `json:"modelParameters,omitempty"`
	Usage             *Usage         `json:"usage,omitempty"`
}

// SessionWithTraces represents a session with its traces
type SessionWithTraces struct {
	ID        string                 `json:"id"`
	CreatedAt string                 `json:"createdAt"`
	Traces    []TraceWithFullDetails `json:"traces"`
}

// PaginatedTraces represents paginated trace list response
type PaginatedTraces struct {
	Data       []TraceWithFullDetails `json:"data"`
	Meta       PaginationMeta         `json:"meta"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int   `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

// GetTraceParams represents parameters for fetching a single trace
type GetTraceParams struct {
	TraceID string
}

// ListTracesParams represents parameters for listing traces
type ListTracesParams struct {
	Page      *int
	Limit     *int
	UserID    *string
	Name      *string
	SessionID *string
	FromTimestamp *string
	ToTimestamp   *string
	Tags      []string
}

// GetSessionParams represents parameters for fetching a session
type GetSessionParams struct {
	SessionID string
}

// GetTrace retrieves a single trace by ID with all its observations
func (c *Client) GetTrace(ctx context.Context, params GetTraceParams) (*TraceWithFullDetails, error) {
	if !c.config.Enabled {
		return nil, fmt.Errorf("client is disabled")
	}

	if params.TraceID == "" {
		return nil, fmt.Errorf("traceID is required")
	}

	url := fmt.Sprintf("%s/api/public/traces/%s", c.config.BaseURL, params.TraceID)

	trace, err := c.fetchJSON(ctx, url, &TraceWithFullDetails{})
	if err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	return trace.(*TraceWithFullDetails), nil
}

// ListTraces retrieves a paginated list of traces
func (c *Client) ListTraces(ctx context.Context, params ListTracesParams) (*PaginatedTraces, error) {
	if !c.config.Enabled {
		return nil, fmt.Errorf("client is disabled")
	}

	baseURL := fmt.Sprintf("%s/api/public/traces", c.config.BaseURL)
	queryParams := url.Values{}

	if params.Page != nil {
		queryParams.Set("page", strconv.Itoa(*params.Page))
	}
	if params.Limit != nil {
		queryParams.Set("limit", strconv.Itoa(*params.Limit))
	}
	if params.UserID != nil {
		queryParams.Set("userId", *params.UserID)
	}
	if params.Name != nil {
		queryParams.Set("name", *params.Name)
	}
	if params.SessionID != nil {
		queryParams.Set("sessionId", *params.SessionID)
	}
	if params.FromTimestamp != nil {
		queryParams.Set("fromTimestamp", *params.FromTimestamp)
	}
	if params.ToTimestamp != nil {
		queryParams.Set("toTimestamp", *params.ToTimestamp)
	}
	for _, tag := range params.Tags {
		queryParams.Add("tags", tag)
	}

	fullURL := baseURL
	if len(queryParams) > 0 {
		fullURL = baseURL + "?" + queryParams.Encode()
	}

	traces, err := c.fetchJSON(ctx, fullURL, &PaginatedTraces{})
	if err != nil {
		return nil, fmt.Errorf("failed to list traces: %w", err)
	}

	return traces.(*PaginatedTraces), nil
}

// GetSession retrieves a session with all its traces
func (c *Client) GetSession(ctx context.Context, params GetSessionParams) (*SessionWithTraces, error) {
	if !c.config.Enabled {
		return nil, fmt.Errorf("client is disabled")
	}

	if params.SessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}

	url := fmt.Sprintf("%s/api/public/sessions/%s", c.config.BaseURL, params.SessionID)

	session, err := c.fetchJSON(ctx, url, &SessionWithTraces{})
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session.(*SessionWithTraces), nil
}

// fetchJSON is a helper method to make GET requests and parse JSON responses
func (c *Client) fetchJSON(ctx context.Context, url string, target interface{}) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.makeAuthHeader())
	req.Header.Set("Accept", "application/json")

	if c.config.Debug {
		fmt.Printf("[Langfuse] GET %s\n", url)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewNetworkError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewHTTPError(resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if c.config.Debug {
		fmt.Printf("[Langfuse] Successfully fetched data from %s\n", url)
	}

	return target, nil
}
