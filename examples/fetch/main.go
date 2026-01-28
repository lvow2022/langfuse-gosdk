package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

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

	// ==========================================
	// 示例 3: 列出 Traces (分页查询)
	// ==========================================
	fmt.Println("=== Example 3: Listing Traces (Paginated) ===")

	page := 1
	limit := 5
	params := langfuse.ListTracesParams{
		Page:  &page,
		Limit: &limit,
	}

	// 可选: 添加过滤条件
	userID := getEnv("USER_ID", "")
	if userID != "" {
		params.UserID = &userID
		fmt.Printf("Filtering by User ID: %s\n", userID)
	}

	if sessionID != "" {
		params.SessionID = &sessionID
		fmt.Printf("Filtering by Session ID: %s\n", sessionID)
	}

	tags := getEnv("TAGS", "")
	if tags != "" {
		params.Tags = []string{tags}
		fmt.Printf("Filtering by Tag: %s\n", tags)
	}

	fmt.Println()

	traces, err := client.ListTraces(ctx, params)
	if err != nil {
		log.Printf("Failed to list traces: %v\n", err)
	} else {
		printTracesList(traces)
	}

	// ==========================================
	// 示例 4: 从 Trace 重建对话上下文
	// ==========================================
	if traceID != "" {
		fmt.Println("\n=== Example 4: Replay Context Reconstruction ===")

		trace, err := client.GetTrace(ctx, langfuse.GetTraceParams{
			TraceID: traceID,
		})
		if err != nil {
			log.Printf("Failed to fetch trace: %v\n", err)
		} else {
			reconstructContext(trace)
		}
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
	fmt.Println("  export USER_ID=\"your-user-id\"")
	fmt.Println("  export TAGS=\"your-tag\"")
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

	// Observations
	fmt.Printf("  Observations: %d\n", len(trace.Observations))
	for i, obs := range trace.Observations {
		fmt.Printf("    [%d] Type: %s", i+1, obs.Type)
		if obs.Name != nil {
			fmt.Printf(", Name: %s", *obs.Name)
		}
		if obs.Model != nil {
			fmt.Printf(", Model: %s", *obs.Model)
		}
		if obs.Usage != nil {
			fmt.Printf(", Usage: input=%d, output=%d, total=%d",
				ptrToInt(obs.Usage.Input),
				ptrToInt(obs.Usage.Output),
				ptrToInt(obs.Usage.Total))
		}
		fmt.Println()
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

// printTracesList prints paginated traces list
func printTracesList(traces *langfuse.PaginatedTraces) {
	fmt.Printf("Pagination Info:\n")
	fmt.Printf("  Page: %d/%d\n", traces.Meta.Page, traces.Meta.TotalPages)
	fmt.Printf("  Total Items: %d\n", traces.Meta.TotalItems)
	fmt.Printf("  Items in this page: %d\n\n", len(traces.Data))

	for i, trace := range traces.Data {
		fmt.Printf("[%d] Trace:\n", i+1)
		fmt.Printf("    ID: %s\n", trace.ID)
		if trace.Name != nil {
			fmt.Printf("    Name: %s\n", *trace.Name)
		}
		if trace.UserID != nil {
			fmt.Printf("    User ID: %s\n", *trace.UserID)
		}
		if trace.SessionID != nil {
			fmt.Printf("    Session ID: %s\n", *trace.SessionID)
		}
		fmt.Printf("    Tags: %v\n", trace.Tags)
		fmt.Printf("    Observations: %d\n", len(trace.Observations))
		fmt.Printf("    Scores: %d\n", len(trace.Scores))
		fmt.Println()
	}
}

// reconstructContext demonstrates how to rebuild conversation from trace
func reconstructContext(trace *langfuse.TraceWithFullDetails) {
	fmt.Println("Reconstructing conversation from trace data:\n")

	// 从 Input 提取用户消息
	if trace.Input != nil {
		if inputMap, ok := trace.Input.(map[string]interface{}); ok {
			if userMsg, exists := inputMap["message"]; exists {
				fmt.Printf("User Input:\n  %v\n\n", userMsg)
			}
		}
	}

	// 从 Output 提取助手回复
	if trace.Output != nil {
		if outputMap, ok := trace.Output.(map[string]interface{}); ok {
			if assistantMsg, exists := outputMap["answer"]; exists {
				fmt.Printf("Assistant Output:\n  %v\n\n", assistantMsg)
			}
		}
	}

	// 从 Observations 提取详细信息
	fmt.Println("Observations Details:")
	for i, obs := range trace.Observations {
		fmt.Printf("\n[%d] %s", i+1, obs.Type)
		if obs.Name != nil {
			fmt.Printf(" - %s", *obs.Name)
		}
		fmt.Println()

		if obs.Type == "GENERATION" {
			// LLM Generation 信息
			if obs.Input != nil {
				fmt.Println("  Input:")
				inputJSON, _ := json.MarshalIndent(obs.Input, "    ", "  ")
				fmt.Printf("    %s\n", inputJSON)
			}

			if obs.Output != nil {
				fmt.Println("  Output:")
				outputJSON, _ := json.MarshalIndent(obs.Output, "    ", "  ")
				fmt.Printf("    %s\n", outputJSON)
			}

			if obs.Usage != nil {
				fmt.Printf("  Token Usage: input=%d, output=%d, total=%d\n",
					ptrToInt(obs.Usage.Input),
					ptrToInt(obs.Usage.Output),
					ptrToInt(obs.Usage.Total))
			}

			if obs.Model != nil {
				fmt.Printf("  Model: %s\n", *obs.Model)
			}
		} else if obs.Type == "SPAN" {
			// Span 信息 (如 RAG retrieval)
			if obs.Input != nil {
				fmt.Println("  Input:")
				inputJSON, _ := json.MarshalIndent(obs.Input, "    ", "  ")
				fmt.Printf("    %s\n", inputJSON)
			}

			if obs.Output != nil {
				fmt.Println("  Output:")
				outputJSON, _ := json.MarshalIndent(obs.Output, "    ", "  ")
				fmt.Printf("    %s\n", outputJSON)
			}
		} else if obs.Type == "TOOL" {
			// Tool 调用信息
			if obs.Input != nil {
				fmt.Println("  Input:")
				inputJSON, _ := json.MarshalIndent(obs.Input, "    ", "  ")
				fmt.Printf("    %s\n", inputJSON)
			}

			if obs.Output != nil {
				fmt.Println("  Output:")
				outputJSON, _ := json.MarshalIndent(obs.Output, "    ", "  ")
				fmt.Printf("    %s\n", outputJSON)
			}
		}
	}

	fmt.Println("\nContext Reconstruction Summary:")
	fmt.Println("This trace data can be used to:")
	fmt.Println("  - Replay the exact conversation")
	fmt.Println("  - Analyze LLM behavior and token usage")
	fmt.Println("  - Debug issues in production")
	fmt.Println("  - Generate test datasets")
	fmt.Println("  - Build conversation memory systems")
}

// Helper function to convert *int to int
func ptrToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
