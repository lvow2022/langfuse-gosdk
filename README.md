# Langfuse Go SDK

Official Go SDK for [Langfuse](https://langfuse.com) - the open-source LLM engineering platform.

## Features

- ✅ **Complete Event Type Support** (16 event types)
  - Trace & Score
  - 10 Observation types: Span, Event, Generation, Agent, Tool, Chain, Retriever, Evaluator, Embedding, Guardrail
  - SDK Log events

- 🔄 **Retry Mechanism**
  - Automatic exponential backoff for retryable errors (429, 5xx)
  - Configurable retry attempts and delays
  - Error classification (retryable vs non-retryable)

- 📊 **Metrics Collection**
  - Event tracking (enqueued, flushed, succeeded, failed, dropped)
  - Flush monitoring with callbacks
  - Failed events tracking

- 🔄 **Replay Context**
  - Complete conversation context storage
  - One-click OpenAI message conversion
  - Session-based multi-trace tracking
  - Tool call execution details

## Installation

```bash
go get github.com/langfuse/langfuse-go
```

## Quick Start

```go
package main

import (
    "context"
    langfuse "github.com/langfuse/langfuse-go/langfuse"
)

func main() {
    // Initialize client
    config := langfuse.DefaultConfig()
    config.PublicKey = "pk-lf-..."
    config.SecretKey = "sk-lf-..."
    config.BaseURL = "http://localhost:3000" // or "https://cloud.langfuse.com"

    client, err := langfuse.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // Create a trace
    trace, _ := client.CreateTrace(langfuse.TraceParams{
        Name:   ptr("my-first-trace"),
        UserID: ptr("user-123"),
    })

    // Create a generation
    genID, _ := trace.CreateGeneration(langfuse.GenerationParams{
        Model: ptr("gpt-4"),
    })

    // Update generation with results
    client.UpdateGeneration(genID, langfuse.GenerationParams{
        SpanParams: langfuse.SpanParams{
            ObservationParams: langfuse.ObservationParams{
                Output: map[string]any{"content": "Hello, world!"},
            },
        },
    })

    // Flush events
    client.Flush(context.Background())
}
```

## Documentation

See [examples/simple](examples/simple) for a complete chat demo with tool calls and replay context.

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `PublicKey` | string | - | Langfuse project public key |
| `SecretKey` | string | - | Langfuse project secret key |
| `BaseURL` | string | `https://cloud.langfuse.com` | Langfuse API base URL |
| `FlushInterval` | duration | 1s | How often to flush events |
| `FlushAt` | int | 15 | Batch size before auto-flush |
| `MaxQueueSize` | int | 1000 | Maximum queue size |
| `Timeout` | duration | 10s | HTTP request timeout |
| `MaxRetryAttempts` | int | 5 | Maximum retry attempts |
| `RetryBaseDelay` | duration | 5s | Base delay for retries |
| `RetryMaxDelay` | duration | 30s | Maximum delay for retries |
| `MetricsEnabled` | bool | false | Enable metrics collection |
| `Debug` | bool | false | Enable debug logging |

### Callbacks

```go
config.OnEventFlushed = func(successCount, errorCount int) {
    fmt.Printf("Flushed: %d succeeded, %d failed\n", successCount, errorCount)
}

config.OnEventDropped = func(count int) {
    log.Printf("WARNING: %d events dropped\n", count)
}
```

## Metrics

```go
snapshot := client.GetMetrics()
fmt.Printf("Success Rate: %.2f%%\n", snapshot.SuccessRate())
fmt.Printf("Drop Rate: %.2f%%\n", snapshot.DropRate())
```

## Replay Context

The SDK supports storing complete conversation context for replay functionality:

```go
// From Langfuse UI, get the replay_context JSON
var replayCtx ReplayContext
json.Unmarshal(replayContextJSON, &replayCtx)

// Convert to OpenAI messages
messages := replayCtx.ToOpenAIMessages()

// Replay the conversation
resp := openaiClient.CreateChatCompletion(ctx, messages)
```

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
