package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lvow2022/langfuse-gosdk/langfuse"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func main() {
	fmt.Println("========================================")
	fmt.Println("  Langfuse Fetch Data Example")
	fmt.Println("========================================")

	config := langfuse.DefaultConfig()
	config.PublicKey = os.Getenv("LANGFUSE_PUBLIC_KEY")
	config.SecretKey = os.Getenv("LANGFUSE_SECRET_KEY")
	config.BaseURL = os.Getenv("LANGFUSE_BASE_URL")
	config.Debug = true

	client, err := langfuse.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create Langfuse client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// ==========================================
	// 示例 1: 获取单个 Trace
	// ==========================================
	// 替换为实际存在的 trace ID
	traceID := getEnv("TRACE_ID", "2f8e2eb5-4b69-47d2-91cf-975cac523b3b")
	if traceID != "" {
		fmt.Printf("=== Example 1: Fetching Trace by ID ===\n")
		fmt.Printf("Trace ID: %s\n\n", traceID)

		trace, err := client.GetTrace(ctx, langfuse.GetTraceParams{
			TraceID: traceID,
		})
		if err != nil {
			log.Printf("Failed to fetch trace: %v\n", err)
		} else {
			printTrace(trace)
		}
		fmt.Println()
	}

	// ==========================================
	// 示例 2: 获取 Session 下的所有 Traces
	// ==========================================
	sessionID := getEnv("SESSION_ID", "")
	if sessionID != "" {
		fmt.Printf("=== Example 2: Fetching Session ===\n")
		fmt.Printf("Session ID: %s\n\n", sessionID)

		session, err := client.GetSession(ctx, langfuse.GetSessionParams{
			SessionID: sessionID,
		})
		if err != nil {
			log.Printf("Failed to fetch session: %v\n", err)
		} else {
			printSession(session)
		}
		fmt.Println()
	}

	fmt.Println("\n========================================")
	fmt.Println("=== Examples Complete ===")
	fmt.Println("========================================")
	fmt.Printf("\nView your data in Langfuse UI:\n%s\n", config.BaseURL)

	if traceID != "" || sessionID != "" {
		fmt.Println("\nUsed IDs:")
		if traceID != "" {
			fmt.Printf("  Trace ID: %s\n", traceID)
		}
		if sessionID != "" {
			fmt.Printf("  Session ID: %s\n", sessionID)
		}
	}

	fmt.Println("\nUsage:")
	fmt.Println("  Set environment variables to fetch specific data:")
	fmt.Println("  export TRACE_ID=\"your-trace-id\"")
	fmt.Println("  export SESSION_ID=\"your-session-id\"")
}

// printTrace prints trace details in a readable format
func printTrace(trace *langfuse.TraceWithFullDetails) {
	fmt.Printf("Trace Details:\n")
	fmt.Printf("  ID: %s\n", trace.ID)
	if trace.Name != nil {
		fmt.Printf("  Name: %s\n", *trace.Name)
	}
	if trace.UserID != nil {
		fmt.Printf("  User ID: %s\n", *trace.UserID)
	}
	if trace.SessionID != nil {
		fmt.Printf("  Session ID: %s\n", *trace.SessionID)
	}
	fmt.Printf("  Timestamp: %s\n", trace.Timestamp)
	fmt.Printf("  Tags: %v\n", trace.Tags)

	// Input
	if trace.Input != nil {
		fmt.Println("  Input:")
		inputJSON, _ := json.MarshalIndent(trace.Input, "    ", "  ")
		fmt.Printf("    %s\n", inputJSON)
	}

	// Output
	if trace.Output != nil {
		fmt.Println("  Output:")
		outputJSON, _ := json.MarshalIndent(trace.Output, "    ", "  ")
		fmt.Printf("    %s\n", outputJSON)
	}

	// Metadata
	if len(trace.Metadata) > 0 {
		fmt.Println("  Metadata:")
		metadataJSON, _ := json.MarshalIndent(trace.Metadata, "    ", "  ")
		fmt.Printf("    %s\n", metadataJSON)
	}

	// Observations (Spans/Generations/Tools)
	fmt.Printf("  Observations: %d\n", len(trace.Observations))
	for i, obs := range trace.Observations {
		fmt.Printf("\n    ┌─ [%d] %s", i+1, obs.Type)
		if obs.Name != nil {
			fmt.Printf(" - %s", *obs.Name)
		}
		fmt.Println()

		// Time information
		fmt.Printf("    │  Time: %s", obs.StartTime)
		if obs.EndTime != nil {
			fmt.Printf(" → %s", *obs.EndTime)
			// Calculate duration
			if startTime, err := time.Parse(time.RFC3339Nano, obs.StartTime); err == nil {
				if endTime, err := time.Parse(time.RFC3339Nano, *obs.EndTime); err == nil {
					duration := endTime.Sub(startTime)
					fmt.Printf(" (duration: %v)", duration.Round(time.Millisecond))
				}
			}
		}
		fmt.Println()

		// Level and Status
		if obs.Level != nil || obs.StatusMessage != nil {
			fmt.Printf("    │  Status: ")
			if obs.Level != nil {
				fmt.Printf("%s", *obs.Level)
			}
			if obs.StatusMessage != nil {
				if obs.Level != nil {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", *obs.StatusMessage)
			}
			fmt.Println()
		}

		// Model & Usage (for GENERATION type)
		if obs.Type == "GENERATION" {
			if obs.Model != nil {
				fmt.Printf("    │  Model: %s\n", *obs.Model)
			}
			if obs.ModelParameters != nil && len(obs.ModelParameters) > 0 {
				fmt.Printf("    │  Model Parameters: %v\n", obs.ModelParameters)
			}
			if obs.Usage != nil {
				fmt.Printf("    │  Usage: input=%d, output=%d, total=%d\n",
					ptrToInt(obs.Usage.Input),
					ptrToInt(obs.Usage.Output),
					ptrToInt(obs.Usage.Total))
			}
		}

		// Input (show full content)
		if obs.Input != nil {
			fmt.Printf("    │  Input:\n")

			// Handle JSON string (API returns data as string)
			var inputData interface{}
			switch v := obs.Input.(type) {
			case string:
				// Try to parse as JSON
				if err := json.Unmarshal([]byte(v), &inputData); err == nil {
					// Successfully parsed
				} else {
					// Not JSON, use as-is
					inputData = v
				}
			default:
				inputData = v
			}

			inputJSON, _ := json.MarshalIndent(inputData, "      ", "  ")
			inputStr := string(inputJSON)
			if len(inputStr) > 500 {
				fmt.Printf("      %s... (truncated, %d chars)\n",
					truncateString(inputStr, 500), len(inputStr))
			} else {
				fmt.Printf("      %s\n", inputStr)
			}
		}

		// Output (show full content)
		if obs.Output != nil {
			fmt.Printf("    │  Output:\n")

			// Handle JSON string (API returns data as string)
			var outputData interface{}
			switch v := obs.Output.(type) {
			case string:
				// Try to parse as JSON
				if err := json.Unmarshal([]byte(v), &outputData); err == nil {
					// Successfully parsed
				} else {
					// Not JSON, use as-is
					outputData = v
				}
			default:
				outputData = v
			}

			outputJSON, _ := json.MarshalIndent(outputData, "      ", "  ")
			outputStr := string(outputJSON)
			if len(outputStr) > 500 {
				fmt.Printf("      %s... (truncated, %d chars)\n",
					truncateString(outputStr, 500), len(outputStr))
			} else {
				fmt.Printf("      %s\n", outputStr)
			}
		}

		// Parent observation ID
		if obs.ParentObservationID != nil {
			fmt.Printf("    │  Parent: %s\n", *obs.ParentObservationID)
		}

		fmt.Printf("    └─────────────────────────────────\n")
	}

	// Scores
	if len(trace.Scores) > 0 {
		fmt.Printf("  Scores: %d\n", len(trace.Scores))
		for i, score := range trace.Scores {
			fmt.Printf("    [%d] Name: %s, Value: %.2f, Type: %s\n",
				i+1, score.Name, score.Value, score.DataType)
		}
	}
}

// printSession prints session details
func printSession(session *langfuse.SessionWithTraces) {
	fmt.Printf("Session Details:\n")
	fmt.Printf("  ID: %s\n", session.ID)
	fmt.Printf("  Created At: %s\n", session.CreatedAt)
	fmt.Printf("  Total Traces: %d\n", len(session.Traces))

	for i, trace := range session.Traces {
		fmt.Printf("\n======== Trace %d ========\n", i+1)
		printTrace(&trace)
	}
}

// Helper function to convert *int to int
func ptrToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// truncateString truncates a string to max length and adds ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
