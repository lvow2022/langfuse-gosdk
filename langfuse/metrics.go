package langfuse

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks SDK operational metrics
type Metrics struct {
	mu sync.Mutex

	// Event counters
	eventsEnqueued  int64
	eventsFlushed   int64
	eventsSucceeded int64
	eventsFailed    int64
	eventsDropped   int64

	// Operation counters
	flushCount int64
	retryCount int64

	// Timing
	lastFlushTimeUnix int64 // Unix timestamp in nanoseconds

	// Failed events for monitoring (limited size)
	failedEvents []FailedEvent
}

// FailedEvent represents an event that failed to send
type FailedEvent struct {
	Event     Event
	Error     error
	Attempt   int
	Timestamp time.Time
}

// RecordEnqueued records that events were added to the queue
func (m *Metrics) RecordEnqueued(count int) {
	atomic.AddInt64(&m.eventsEnqueued, int64(count))
}

// RecordFlush records a flush operation with success and failure counts
func (m *Metrics) RecordFlush(success, failed int) {
	atomic.AddInt64(&m.eventsFlushed, int64(success+failed))
	atomic.AddInt64(&m.eventsSucceeded, int64(success))
	atomic.AddInt64(&m.eventsFailed, int64(failed))
	atomic.AddInt64(&m.flushCount, 1)
	atomic.StoreInt64(&m.lastFlushTimeUnix, time.Now().UnixNano())
}

// RecordDropped records that events were dropped due to a full queue
func (m *Metrics) RecordDropped(count int) {
	atomic.AddInt64(&m.eventsDropped, int64(count))
}

// RecordRetry records that a retry attempt was made
func (m *Metrics) RecordRetry() {
	atomic.AddInt64(&m.retryCount, 1)
}

// RecordFailedEvent records a failed event for monitoring
func (m *Metrics) RecordFailedEvent(event Event, err error, attempt int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failedEvents = append(m.failedEvents, FailedEvent{
		Event:     event,
		Error:     err,
		Attempt:   attempt,
		Timestamp: time.Now(),
	})

	// Limit the size to prevent unbounded growth
	if len(m.failedEvents) > 1000 {
		m.failedEvents = m.failedEvents[len(m.failedEvents)-1000:]
	}
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	lastFlushUnix := atomic.LoadInt64(&m.lastFlushTimeUnix)
	var lastFlush time.Time
	if lastFlushUnix > 0 {
		lastFlush = time.Unix(0, lastFlushUnix)
	}

	return MetricsSnapshot{
		EventsEnqueued:  atomic.LoadInt64(&m.eventsEnqueued),
		EventsFlushed:   atomic.LoadInt64(&m.eventsFlushed),
		EventsSucceeded: atomic.LoadInt64(&m.eventsSucceeded),
		EventsFailed:    atomic.LoadInt64(&m.eventsFailed),
		EventsDropped:   atomic.LoadInt64(&m.eventsDropped),
		FlushCount:      atomic.LoadInt64(&m.flushCount),
		RetryCount:      atomic.LoadInt64(&m.retryCount),
		LastFlushTime:   lastFlush,
		FailedEventCount: len(m.failedEvents),
	}
}

// GetFailedEvents returns a copy of the failed events list
func (m *Metrics) GetFailedEvents() []FailedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	events := make([]FailedEvent, len(m.failedEvents))
	copy(events, m.failedEvents)
	return events
}

// Reset clears all metrics (useful for testing)
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.eventsEnqueued, 0)
	atomic.StoreInt64(&m.eventsFlushed, 0)
	atomic.StoreInt64(&m.eventsSucceeded, 0)
	atomic.StoreInt64(&m.eventsFailed, 0)
	atomic.StoreInt64(&m.eventsDropped, 0)
	atomic.StoreInt64(&m.flushCount, 0)
	atomic.StoreInt64(&m.retryCount, 0)
	atomic.StoreInt64(&m.lastFlushTimeUnix, 0)

	m.mu.Lock()
	m.failedEvents = nil
	m.mu.Unlock()
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	EventsEnqueued   int64
	EventsFlushed    int64
	EventsSucceeded  int64
	EventsFailed     int64
	EventsDropped    int64
	FlushCount       int64
	RetryCount       int64
	LastFlushTime    time.Time
	FailedEventCount int
}

// String returns a formatted string representation of the snapshot
func (s MetricsSnapshot) String() string {
	lastFlush := "never"
	if !s.LastFlushTime.IsZero() {
		lastFlush = s.LastFlushTime.Format(time.RFC3339)
	}

	return fmt.Sprintf(
		"Enqueued: %d, Flushed: %d (Success: %d, Failed: %d), Dropped: %d, Retries: %d, Flushes: %d, LastFlush: %s",
		s.EventsEnqueued, s.EventsFlushed, s.EventsSucceeded, s.EventsFailed,
		s.EventsDropped, s.RetryCount, s.FlushCount, lastFlush,
	)
}

// SuccessRate returns the success rate as a percentage (0-100)
func (s MetricsSnapshot) SuccessRate() float64 {
	if s.EventsFlushed == 0 {
		return 100.0
	}
	return (float64(s.EventsSucceeded) / float64(s.EventsFlushed)) * 100.0
}

// DropRate returns the drop rate as a percentage (0-100)
func (s MetricsSnapshot) DropRate() float64 {
	if s.EventsEnqueued == 0 {
		return 0.0
	}
	return (float64(s.EventsDropped) / float64(s.EventsEnqueued)) * 100.0
}
