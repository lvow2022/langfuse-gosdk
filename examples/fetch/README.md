# Langfuse Go SDK - Fetch Data Example

This example demonstrates how to fetch and retrieve data from Langfuse using the Go SDK.

## Features Demonstrated

1. **Fetch Single Trace** - Retrieve a specific trace by ID with all observations
2. **Fetch Session** - Get all traces within a session
3. **List Traces** - Paginated trace listing with filters
4. **Replay Context** - Reconstruct conversation from fetched trace data

## Prerequisites

- Go 1.24 or higher
- Langfuse instance with existing data (traces/sessions)
- Valid Langfuse API credentials

## Installation

```bash
cd examples/fetch
go mod init fetch-example
go mod edit -replace github.com/lvow2022/langfuse-gosdk=../..
go mod tidy
```

## Configuration

Set the following environment variables:

```bash
# Required
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
export LANGFUSE_BASE_URL="http://localhost:3000"  # or https://cloud.langfuse.com

# Optional - to fetch specific data
export TRACE_ID="your-trace-id"           # Fetch specific trace
export SESSION_ID="your-session-id"       # Fetch specific session
export USER_ID="your-user-id"             # Filter traces by user
export TAGS="your-tag"                    # Filter traces by tag
```

**Note**: You need existing traces/sessions in your Langfuse instance. Get IDs from the Langfuse UI.

## Running the Example

```bash
go run main.go
```

## What This Example Does

### 1. Fetch by Trace ID

```go
fetchedTrace, err := client.GetTrace(ctx, langfuse.GetTraceParams{
    TraceID: traceID,
})
```

Returns:
- Trace metadata (ID, name, user, session, timestamps)
- Input and output data
- All nested observations (spans, generations, events, tools)
- Scores and evaluations

### 2. Fetch by Session ID

```go
session, err := client.GetSession(ctx, langfuse.GetSessionParams{
    SessionID: sessionID,
})
```

Returns:
- Session metadata
- All traces in the session
- Complete nested structure

### 3. List Traces with Filters

```go
tracesList, err := client.ListTraces(ctx, langfuse.ListTracesParams{
    Page:      &page,
    Limit:     &limit,
    UserID:    &userID,
    SessionID: &sessionID,
    Tags:      []string{"fetch-example"},
})
```

Supports:
- Pagination (page, limit)
- Filtering by user, session, name
- Time range filtering
- Tag filtering

### 4. Replay Context Reconstruction

The example shows how to:
- Extract conversation messages from trace data
- Reconstruct LLM input/output from generations
- Access token usage and metadata
- Rebuild conversation for replay or analysis

## API Methods

### GetTrace

Fetch a single trace with full details:

```go
trace, err := client.GetTrace(ctx, langfuse.GetTraceParams{
    TraceID: "trace-id-here",
})
```

### GetSession

Fetch a session with all its traces:

```go
session, err := client.GetSession(ctx, langfuse.GetSessionParams{
    SessionID: "session-id-here",
})
```

### ListTraces

List traces with pagination and filters:

```go
page := 1
limit := 10
userID := "user-123"

traces, err := client.ListTraces(ctx, langfuse.ListTracesParams{
    Page:      &page,
    Limit:     &limit,
    UserID:    &userID,
    Tags:      []string{"production"},
})

fmt.Printf("Page %d/%d, Total: %d\n",
    traces.Meta.Page,
    traces.Meta.TotalPages,
    traces.Meta.TotalItems)

for _, trace := range traces.Data {
    fmt.Printf("Trace: %s\n", trace.ID)
}
```

## Use Cases

### 1. Debugging and Analysis

Fetch traces to analyze:
- LLM performance and behavior
- Token usage patterns
- Error rates and types
- Latency bottlenecks

### 2. Replay and Testing

Reconstruct conversations:
- Replay user sessions for testing
- Generate test datasets from production
- Validate model behavior consistency

### 3. Data Export

Extract data for:
- Custom analytics
- External dashboards
- Compliance and auditing
- ML training datasets

### 4. Context Management

Retrieve conversation history:
- Build context windows for LLM calls
- Implement memory systems
- Personalization features

## Response Structure

### TraceWithFullDetails

```go
type TraceWithFullDetails struct {
    ID           string
    Name         *string
    UserID       *string
    SessionID    *string
    Timestamp    string
    Input        map[string]interface{}
    Output       map[string]interface{}
    Metadata     map[string]interface{}
    Tags         []string
    Observations []ObservationDetails
    Scores       []Score
}
```

### ObservationDetails

```go
type ObservationDetails struct {
    ID                  string
    TraceID             string
    Type                string  // SPAN, GENERATION, EVENT, TOOL
    Name                *string
    StartTime           string
    EndTime             *string
    Input               map[string]interface{}
    Output              map[string]interface{}
    Metadata            map[string]interface{}
    Level               *string
    StatusMessage       *string
    ParentObservationID *string
    Model               *string
    ModelParameters     map[string]interface{}
    Usage               *Usage
}
```

### SessionWithTraces

```go
type SessionWithTraces struct {
    ID        string
    CreatedAt string
    Traces    []TraceWithFullDetails
}
```

## Error Handling

```go
trace, err := client.GetTrace(ctx, langfuse.GetTraceParams{
    TraceID: traceID,
})
if err != nil {
    switch e := err.(type) {
    case *langfuse.HTTPError:
        if e.StatusCode == 404 {
            fmt.Println("Trace not found")
        }
    case *langfuse.NetworkError:
        fmt.Println("Network error:", e)
    default:
        fmt.Println("Error:", err)
    }
}
```

## Performance Considerations

1. **Pagination**: Use appropriate page sizes (10-100) to avoid overwhelming the API
2. **Filters**: Apply filters (user, session, tags) to reduce data transfer
3. **Time Range**: Use fromTimestamp/toTimestamp for large datasets
4. **Caching**: Consider caching fetched data locally for repeated access

## Related Examples

- `/examples/simple` - Basic trace creation and streaming
- `/examples/fetch` - This example (data retrieval)

## References

- [Langfuse API Documentation](https://langfuse.com/docs/api)
- [OpenAPI Specification](https://cloud.langfuse.com/generated/api/openapi.yml)
- [Langfuse Go SDK](https://github.com/lvow2022/langfuse-gosdk)
