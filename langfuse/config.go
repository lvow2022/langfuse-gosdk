package langfuse

import (
	"time"
)

// Config holds the configuration for the Langfuse client
type Config struct {
	// PublicKey is the Langfuse project public key
	PublicKey string

	// SecretKey is the Langfuse project secret key
	SecretKey string

	// BaseURL is the Langfuse API base URL (default: https://cloud.langfuse.com)
	BaseURL string

	// FlushInterval is how often to flush events to the server (default: 1 second)
	FlushInterval time.Duration

	// FlushAt is the number of events to batch before flushing (default: 15)
	FlushAt int

	// MaxQueueSize is the maximum number of events to queue before dropping (default: 1000)
	MaxQueueSize int

	// Timeout is the HTTP request timeout (default: 10 seconds)
	Timeout time.Duration

	// SDKIntegration identifies the SDK integration (optional)
	SDKIntegration string

	// SDKVersion is the version of this SDK
	SDKVersion string

	// Enabled controls whether the SDK is active (default: true)
	Enabled bool

	// Debug enables debug logging (default: false)
	Debug bool

	// MaxRetryAttempts is the maximum number of retry attempts for retryable errors (default: 5)
	MaxRetryAttempts int

	// RetryBaseDelay is the base delay for retry backoff (default: 5 seconds)
	RetryBaseDelay time.Duration

	// RetryMaxDelay is the maximum delay for retry backoff (default: 30 seconds)
	RetryMaxDelay time.Duration

	// MetricsEnabled enables metrics collection (default: false)
	MetricsEnabled bool

	// OnEventFlushed is called after each flush with success and error counts
	OnEventFlushed func(successCount, errorCount int)

	// OnEventDropped is called when events are dropped due to a full queue
	OnEventDropped func(count int)
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		BaseURL:          "https://cloud.langfuse.com",
		FlushInterval:    1 * time.Second,
		FlushAt:          15,
		MaxQueueSize:     1000,
		Timeout:          10 * time.Second,
		SDKVersion:       "0.2.0",
		Enabled:          true,
		Debug:            false,
		MaxRetryAttempts: 5,
		RetryBaseDelay:   5 * time.Second,
		RetryMaxDelay:    30 * time.Second,
		MetricsEnabled:   false,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.PublicKey == "" {
		return &ConfigError{Field: "PublicKey", Message: "public key is required"}
	}
	if c.SecretKey == "" {
		return &ConfigError{Field: "SecretKey", Message: "secret key is required"}
	}
	if c.BaseURL == "" {
		return &ConfigError{Field: "BaseURL", Message: "base URL is required"}
	}
	if c.FlushAt <= 0 {
		return &ConfigError{Field: "FlushAt", Message: "flush at must be positive"}
	}
	if c.MaxQueueSize <= 0 {
		return &ConfigError{Field: "MaxQueueSize", Message: "max queue size must be positive"}
	}
	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + ": " + e.Message
}
