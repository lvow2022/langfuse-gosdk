package langfuse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Client is the main Langfuse client
type Client struct {
	config     *Config
	httpClient *http.Client
	batcher    *Batcher
	metrics    *Metrics
	mu         sync.Mutex
	closed     bool
}

// NewClient creates a new Langfuse client with the given configuration
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	client := &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		metrics: &Metrics{},
	}

	// Initialize batcher for async event sending
	if config.Enabled {
		client.batcher = NewBatcher(client, config)
		client.batcher.Start()
	}

	return client, nil
}

// makeAuthHeader creates the Basic Auth header
func (c *Client) makeAuthHeader() string {
	auth := c.config.PublicKey + ":" + c.config.SecretKey
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// sendIngestion sends an ingestion request to the Langfuse API
func (c *Client) sendIngestion(ctx context.Context, req *IngestionRequest) (*IngestionResponse, error) {
	if !c.config.Enabled {
		return &IngestionResponse{}, nil
	}

	url := c.config.BaseURL + "/api/public/ingestion"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.makeAuthHeader())
	httpReq.Header.Set("X-Langfuse-Sdk-Name", "langfuse-go")
	httpReq.Header.Set("X-Langfuse-Sdk-Version", c.config.SDKVersion)
	if c.config.SDKIntegration != "" {
		httpReq.Header.Set("X-Langfuse-Sdk-Integration", c.config.SDKIntegration)
	}

	if c.config.Debug {
		log.Printf("[Langfuse] Sending %d events to %s", len(req.Batch), url)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewNetworkError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewNetworkError(err)
	}

	// API returns 207 Multi-Status for batch requests
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMultiStatus {
		return nil, NewHTTPError(resp.StatusCode, string(respBody))
	}

	var ingestionResp IngestionResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &ingestionResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	if c.config.Debug {
		log.Printf("[Langfuse] Response: %d successes, %d errors", len(ingestionResp.Successes), len(ingestionResp.Errors))
		if len(ingestionResp.Errors) > 0 {
			for _, e := range ingestionResp.Errors {
				log.Printf("[Langfuse] Error: %s - %s", e.Error, e.Message)
			}
		}
	}

	return &ingestionResp, nil
}

// enqueue adds an event to the batch queue
func (c *Client) enqueue(event Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("client is closed")
	}

	if !c.config.Enabled {
		return nil
	}

	return c.batcher.Add(event)
}

// Flush forces all queued events to be sent immediately
func (c *Client) Flush(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	if c.batcher == nil {
		return nil
	}

	return c.batcher.Flush(ctx)
}

// Close stops the client and flushes all pending events
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	if c.batcher != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return c.batcher.Close(ctx)
	}

	return nil
}

// GetMetrics returns a snapshot of current metrics
func (c *Client) GetMetrics() MetricsSnapshot {
	return c.metrics.GetSnapshot()
}

// GetFailedEvents returns a copy of the failed events list
func (c *Client) GetFailedEvents() []FailedEvent {
	return c.metrics.GetFailedEvents()
}

// generateID generates a new UUID for events
func generateID() string {
	return uuid.New().String()
}

// Ptr is a helper function to get a pointer to a value
func Ptr[T any](v T) *T {
	return &v
}

// ptr is a helper function to get a pointer to a value (internal use)
func ptr[T any](v T) *T {
	return &v
}
