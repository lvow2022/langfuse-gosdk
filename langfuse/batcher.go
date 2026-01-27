package langfuse

import (
	"context"
	"log"
	"sync"
	"time"
)

// Batcher handles batching and async sending of events
type Batcher struct {
	client   *Client
	config   *Config
	queue    []Event
	mu       sync.Mutex
	ticker   *time.Ticker
	done     chan struct{}
	wg       sync.WaitGroup
	attempts map[string]int // Track retry attempts per event batch
}

// NewBatcher creates a new batcher
func NewBatcher(client *Client, config *Config) *Batcher {
	return &Batcher{
		client: client,
		config: config,
		queue:  make([]Event, 0, config.MaxQueueSize),
		done:   make(chan struct{}),
	}
}

// Start begins the background flush loop
func (b *Batcher) Start() {
	b.ticker = time.NewTicker(b.config.FlushInterval)
	b.wg.Add(1)

	go func() {
		defer b.wg.Done()
		for {
			select {
			case <-b.ticker.C:
				if err := b.Flush(context.Background()); err != nil {
					if b.config.Debug {
						log.Printf("[Langfuse] Error flushing events: %v", err)
					}
				}
			case <-b.done:
				b.ticker.Stop()
				return
			}
		}
	}()
}

// Add adds an event to the queue
func (b *Batcher) Add(event Event) error {
	// Record metrics if enabled
	if b.config.MetricsEnabled {
		b.client.metrics.RecordEnqueued(1)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if queue is full
	if len(b.queue) >= b.config.MaxQueueSize {
		if b.config.Debug {
			log.Printf("[Langfuse] Queue is full (%d events), dropping event", len(b.queue))
		}

		// Record dropped event
		if b.config.MetricsEnabled {
			b.client.metrics.RecordDropped(1)
		}

		// Call drop callback if provided
		if b.config.OnEventDropped != nil {
			go b.config.OnEventDropped(1)
		}

		return &QueueFullError{MaxSize: b.config.MaxQueueSize}
	}

	b.queue = append(b.queue, event)

	// Auto-flush if we've reached FlushAt threshold
	if len(b.queue) >= b.config.FlushAt {
		// Unlock before flushing to avoid deadlock
		b.mu.Unlock()
		if err := b.Flush(context.Background()); err != nil {
			if b.config.Debug {
				log.Printf("[Langfuse] Error auto-flushing: %v", err)
			}
		}
		b.mu.Lock()
	}

	return nil
}

// Flush sends all queued events immediately
func (b *Batcher) Flush(ctx context.Context) error {
	b.mu.Lock()

	if len(b.queue) == 0 {
		b.mu.Unlock()
		return nil
	}

	// Take all events from queue
	events := make([]Event, len(b.queue))
	copy(events, b.queue)
	b.queue = b.queue[:0] // Clear queue

	b.mu.Unlock()

	// Send events
	req := &IngestionRequest{
		Batch: events,
	}

	resp, err := b.client.sendIngestion(ctx, req)

	// Handle errors
	if err != nil {
		b.handleFlushError(events, err, resp)
		return err
	}

	// Record metrics
	successCount := 0
	errorCount := 0
	if resp != nil {
		successCount = len(resp.Successes)
		errorCount = len(resp.Errors)
	}

	if b.config.MetricsEnabled {
		b.client.metrics.RecordFlush(successCount, errorCount)
	}

	// Call flush callback if provided
	if b.config.OnEventFlushed != nil {
		go b.config.OnEventFlushed(successCount, errorCount)
	}

	// Log any errors from the API
	if resp != nil && len(resp.Errors) > 0 {
		if b.config.Debug {
			log.Printf("[Langfuse] API returned %d errors out of %d events", len(resp.Errors), len(events))
		}
	}

	return nil
}

// handleFlushError processes errors during flush
func (b *Batcher) handleFlushError(events []Event, err error, resp *IngestionResponse) {
	// Check if this is a retryable error
	if langfuseErr, ok := err.(*LangfuseError); ok && langfuseErr.IsRetryable() {
		if b.config.Debug {
			log.Printf("[Langfuse] Retryable error encountered: %v", err)
		}

		// Record retry attempt
		if b.config.MetricsEnabled {
			b.client.metrics.RecordRetry()
		}

		// Put events back at the front of the queue for retry
		b.mu.Lock()
		b.queue = append(events, b.queue...)
		b.mu.Unlock()
		return
	}

	// Non-retryable error - record and discard
	if b.config.Debug {
		log.Printf("[Langfuse] Non-retryable error, dropping %d events: %v", len(events), err)
	}

	// Record failed events for monitoring
	if b.config.MetricsEnabled {
		for _, e := range events {
			b.client.metrics.RecordFailedEvent(e, err, 0)
		}
	}
}

// Close stops the batcher and flushes remaining events
func (b *Batcher) Close(ctx context.Context) error {
	close(b.done)
	b.wg.Wait()

	return b.Flush(ctx)
}

// QueueFullError is returned when the event queue is full
type QueueFullError struct {
	MaxSize int
}

func (e *QueueFullError) Error() string {
	return "event queue is full"
}
