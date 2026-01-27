package main

import (
	"context"
	"fmt"
	"log"
	"time"

	langfuse "github.com/langfuse/langfuse-go/langfuse"
)

func ptr[T any](v T) *T { return &v }

func main() {
	fmt.Println("========================================")
	fmt.Println("  Langfuse Go SDK - Advanced Features")
	fmt.Println("========================================")

	// Create client with metrics and callbacks enabled
	config := langfuse.DefaultConfig()
	config.PublicKey = "pk-lf-your-key"
	config.SecretKey = "sk-lf-your-key"
	config.BaseURL = "http://localhost:3000"
	config.Debug = true
	config.MetricsEnabled = true

	// Set up callbacks
	config.OnEventFlushed = func(successCount, errorCount int) {
		fmt.Printf(">>> Flush completed: %d succeeded, %d failed\n", successCount, errorCount)
	}

	config.OnEventDropped = func(count int) {
		log.Printf("WARNING: %d events were dropped due to full queue\n", count)
	}

	client, err := langfuse.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a trace
	ctx := context.Background()
	trace, err := client.CreateTrace(langfuse.TraceParams{
		Name:      ptr("advanced-features-demo"),
		UserID:    ptr("user-456"),
		SessionID: ptr("session-advanced-001"),
		Metadata: map[string]interface{}{
			"demo":  "advanced-features",
			"build": "v0.2.0",
		},
	})
	if err != nil {
		log.Printf("Warning: failed to create trace: %v\n", err)
	}

	// 1. Test all new observation types
	fmt.Println("\n--- Testing New Observation Types ---")

	// Agent observation
	agentID, err := trace.CreateAgent(langfuse.AgentParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name: ptr("AI Agent"),
				Input: map[string]interface{}{
					"task": "Process user request",
				},
			},
			EndTime: ptr(time.Now().Add(2 * time.Second)),
		},
	})
	if err != nil {
		log.Printf("Agent error: %v\n", err)
	} else {
		fmt.Printf("✓ Agent created: %s\n", agentID)
	}

	// Tool observation
	toolID, err := trace.CreateTool(langfuse.ToolParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name: ptr("Database Query"),
				Input: map[string]interface{}{
					"query": "SELECT * FROM users",
				},
				Output: map[string]interface{}{
					"rows": 100,
				},
			},
			EndTime: ptr(time.Now().Add(500 * time.Millisecond)),
		},
	})
	if err != nil {
		log.Printf("Tool error: %v\n", err)
	} else {
		fmt.Printf("✓ Tool created: %s\n", toolID)
	}

	// Chain observation
	chainID, err := trace.CreateChain(langfuse.ChainParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name: ptr("Processing Pipeline"),
				Input: map[string]interface{}{
					"steps": []string{"validate", "transform", "save"},
				},
			},
			EndTime: ptr(time.Now().Add(3 * time.Second)),
		},
	})
	if err != nil {
		log.Printf("Chain error: %v\n", err)
	} else {
		fmt.Printf("✓ Chain created: %s\n", chainID)
	}

	// Retriever observation
	retrieverID, err := trace.CreateRetriever(langfuse.RetrieverParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name: ptr("Vector Search"),
				Input: map[string]interface{}{
					"query":  "machine learning",
					"top_k":  10,
				},
				Output: map[string]interface{}{
					"results": 10,
				},
			},
			EndTime: ptr(time.Now().Add(1 * time.Second)),
		},
	})
	if err != nil {
		log.Printf("Retriever error: %v\n", err)
	} else {
		fmt.Printf("✓ Retriever created: %s\n", retrieverID)
	}

	// Evaluator observation
	evaluatorID, err := trace.CreateEvaluator(langfuse.EvaluatorParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name: ptr("Quality Check"),
				Input: map[string]interface{}{
					"criteria": "accuracy",
				},
				Output: map[string]interface{}{
					"score": 0.95,
				},
			},
			EndTime: ptr(time.Now().Add(500 * time.Millisecond)),
		},
	})
	if err != nil {
		log.Printf("Evaluator error: %v\n", err)
	} else {
		fmt.Printf("✓ Evaluator created: %s\n", evaluatorID)
	}

	// Embedding observation
	embeddingID, err := trace.CreateEmbedding(langfuse.EmbeddingParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name: ptr("Text Embedding"),
				Input: map[string]interface{}{
					"text": "Hello, world!",
				},
			},
			EndTime: ptr(time.Now().Add(300 * time.Millisecond)),
		},
		EmbeddingModel: ptr("text-embedding-ada-002"),
		EmbeddingModelParameters: map[string]interface{}{
			"dimensions": 1536,
		},
	})
	if err != nil {
		log.Printf("Embedding error: %v\n", err)
	} else {
		fmt.Printf("✓ Embedding created: %s\n", embeddingID)
	}

	// Guardrail observation
	guardrailID, err := trace.CreateGuardrail(langfuse.GuardrailParams{
		ObservationParams: langfuse.ObservationParams{
			Name: ptr("Content Safety Check"),
			Input: map[string]interface{}{
				"content": "user input text",
			},
			Output: map[string]interface{}{
				"passed": true,
			},
		},
	})
	if err != nil {
		log.Printf("Guardrail error: %v\n", err)
	} else {
		fmt.Printf("✓ Guardrail created: %s\n", guardrailID)
	}

	// SDK Log
	err = client.CreateSdkLog(langfuse.SdkLogParams{
		Log: map[string]interface{}{
			"level":   "info",
			"message": "SDK initialized successfully",
			"version": "0.2.0",
		},
	})
	if err != nil {
		log.Printf("SDK Log error: %v\n", err)
	} else {
		fmt.Println("✓ SDK Log created")
	}

	// Update trace with final output
	trace.Update(langfuse.TraceParams{
		Output: map[string]interface{}{
			"status":           "completed",
			"observations_count": 8,
		},
	})

	// Wait a moment for async processing
	time.Sleep(2 * time.Second)

	// Flush and get metrics
	fmt.Println("\n--- Flushing Events ---")
	flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Flush(flushCtx); err != nil {
		log.Printf("Flush error: %v\n", err)
	}

	// 2. Display metrics
	fmt.Println("\n--- Metrics Snapshot ---")
	snapshot := client.GetMetrics()
	fmt.Printf("%s\n", snapshot.String())

	fmt.Printf("\nSuccess Rate: %.2f%%\n", snapshot.SuccessRate())
	fmt.Printf("Drop Rate: %.2f%%\n", snapshot.DropRate())

	// 3. Check for failed events
	fmt.Println("\n--- Failed Events ---")
	failedEvents := client.GetFailedEvents()
	if len(failedEvents) > 0 {
		fmt.Printf("Found %d failed events:\n", len(failedEvents))
		for i, fe := range failedEvents {
			if i >= 5 { // Show only first 5
				fmt.Printf("... and %d more\n", len(failedEvents)-5)
				break
			}
			fmt.Printf("  [%d] Type: %s, Error: %v\n", i+1, fe.Event.Type, fe.Error)
		}
	} else {
		fmt.Println("✓ No failed events")
	}

	fmt.Println("\n========================================")
	fmt.Println("Test completed!")
	fmt.Println("========================================")
}
