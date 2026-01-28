package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	langfuse "github.com/lvow2022/langfuse-gosdk/langfuse"
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func ptr[T any](v T) *T {
	return &v
}

func main() {
	// 从环境变量加载配置
	publicKey := getEnv("LANGFUSE_PUBLIC_KEY", "pk-lf-613f327a-210d-492d-9b0e-1d15ff456dba")
	secretKey := getEnv("LANGFUSE_SECRET_KEY", "sk-lf-dd707fbc-d87d-497a-86c0-7a464af5cfca")
	baseURL := getEnv("LANGFUSE_BASE_URL", "http://192.168.0.55:3000")

	if publicKey == "" || secretKey == "" {
		log.Fatal("LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY must be set")
	}

	fmt.Println("========================================")
	fmt.Println("  Langfuse Fetch Data Example")
	fmt.Println("========================================")
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Println("----------------------------------------\n")

	// 初始化 Langfuse 客户端
	config := langfuse.DefaultConfig()
	config.PublicKey = publicKey
	config.SecretKey = secretKey
	config.BaseURL = baseURL
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
	fmt.Printf("\nView your data in Langfuse UI:\n%s\n", baseURL)

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

		// Input (show preview)
		if obs.Input != nil {
			fmt.Printf("    │  Input: ")
			switch v := obs.Input.(type) {
			case string:
				if len(v) > 100 {
					fmt.Printf("%.100s... (truncated)", v)
				} else {
					fmt.Printf("%s", v)
				}
			case map[string]interface{}:
				// Show key fields
				if msg, ok := v["message"]; ok {
					fmt.Printf("message: %v", truncateString(fmt.Sprintf("%v", msg), 100))
				} else if len(v) > 0 {
					// Show first few keys
					keys := make([]string, 0, min(3, len(v)))
					for k := range v {
						keys = append(keys, k)
						if len(keys) >= 3 {
							break
						}
					}
					fmt.Printf("keys: %v", keys)
				} else {
					fmt.Printf("%v", v)
				}
			case []interface{}:
				fmt.Printf("[%d items]", len(v))
			default:
				fmt.Printf("%v", v)
			}
			fmt.Println()
		}

		// Output (show preview)
		if obs.Output != nil {
			fmt.Printf("    │  Output: ")
			switch v := obs.Output.(type) {
			case string:
				if len(v) > 100 {
					fmt.Printf("%.100s... (truncated)", v)
				} else {
					fmt.Printf("%s", v)
				}
			case map[string]interface{}:
				// Show key fields
				if answer, ok := v["answer"]; ok {
					fmt.Printf("answer: %v", truncateString(fmt.Sprintf("%v", answer), 100))
				} else if result, ok := v["result"]; ok {
					fmt.Printf("result: %v", truncateString(fmt.Sprintf("%v", result), 100))
				} else if len(v) > 0 {
					// Show first few keys
					keys := make([]string, 0, min(3, len(v)))
					for k := range v {
						keys = append(keys, k)
						if len(keys) >= 3 {
							break
						}
					}
					fmt.Printf("keys: %v", keys)
				} else {
					fmt.Printf("%v", v)
				}
			case []interface{}:
				fmt.Printf("[%d items]", len(v))
			default:
				fmt.Printf("%v", v)
			}
			fmt.Println()
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
		fmt.Printf("\n  Trace %d:\n", i+1)
		fmt.Printf("    ID: %s\n", trace.ID)
		if trace.Name != nil {
			fmt.Printf("    Name: %s\n", *trace.Name)
		}
		fmt.Printf("    Timestamp: %s\n", trace.Timestamp)

		if trace.Input != nil {
			if inputMap, ok := trace.Input.(map[string]interface{}); ok {
				if msg, exists := inputMap["message"]; exists {
					fmt.Printf("    Input Message: %v\n", msg)
				}
			}
		}

		if trace.Output != nil {
			if outputMap, ok := trace.Output.(map[string]interface{}); ok {
				if answer, exists := outputMap["answer"]; exists {
					fmt.Printf("    Output Answer: %v\n", answer)
				}
			}
		}

		fmt.Printf("    Observations: %d\n", len(trace.Observations))
		fmt.Printf("    Scores: %d\n", len(trace.Scores))
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
