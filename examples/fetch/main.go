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
	publicKey := getEnv("LANGFUSE_PUBLIC_KEY", "")
	secretKey := getEnv("LANGFUSE_SECRET_KEY", "")
	baseURL := getEnv("LANGFUSE_BASE_URL", "http://localhost:3000")

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
	// 示例 1: 创建一些测试数据
	// ==========================================
	fmt.Println("=== Creating Test Data ===")

	userID := "test-user-123"
	sessionID := fmt.Sprintf("test-session-%d", time.Now().Unix())

	// 创建第一个 trace
	trace1, err := client.CreateTrace(langfuse.TraceParams{
		Name:      ptr("test-trace-1"),
		UserID:    &userID,
		SessionID: &sessionID,
		Input: map[string]interface{}{
			"message": "What is the weather in Beijing?",
		},
		Metadata: map[string]interface{}{
			"source": "fetch-example",
			"version": "1.0",
		},
		Tags: []string{"test", "fetch-example", "weather"},
	})
	if err != nil {
		log.Fatalf("Failed to create trace 1: %v", err)
	}
	fmt.Printf("Created Trace 1: %s\n", trace1.ID())

	// 为 trace1 添加一个 span
	spanStartTime := time.Now()
	span1ID, err := trace1.CreateSpan(langfuse.SpanParams{
		ObservationParams: langfuse.ObservationParams{
			Name:      ptr("rag-retrieval"),
			StartTime: &spanStartTime,
			Input: map[string]interface{}{
				"query": "weather Beijing",
			},
			Metadata: map[string]interface{}{
				"retrieval_method": "vector_search",
			},
		},
	})
	if err != nil {
		log.Printf("Warning: Failed to create span: %v", err)
	} else {
		fmt.Printf("Created Span: %s\n", span1ID)

		// 更新 span
		spanEndTime := time.Now()
		client.UpdateSpan(span1ID, langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Output: map[string]interface{}{
					"documents_found": 5,
					"top_score": 0.95,
				},
			},
			EndTime: &spanEndTime,
		})
	}

	// 为 trace1 添加一个 generation
	genStartTime := time.Now()
	genID, err := trace1.CreateGeneration(langfuse.GenerationParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name:      ptr("llm-generation"),
				StartTime: &genStartTime,
				Input: []map[string]interface{}{
					{"role": "user", "content": "What is the weather in Beijing?"},
				},
			},
		},
		Model: ptr("gpt-4"),
		ModelParameters: map[string]interface{}{
			"temperature": 0.7,
		},
	})
	if err != nil {
		log.Printf("Warning: Failed to create generation: %v", err)
	} else {
		fmt.Printf("Created Generation: %s\n", genID)

		// 更新 generation
		genEndTime := time.Now()
		usage := langfuse.Usage{
			Input:  ptr(50),
			Output: ptr(100),
			Total:  ptr(150),
		}
		client.UpdateGeneration(genID, langfuse.GenerationParams{
			SpanParams: langfuse.SpanParams{
				ObservationParams: langfuse.ObservationParams{
					Output: map[string]interface{}{
						"content": "The weather in Beijing is sunny, 22°C.",
					},
				},
				EndTime: &genEndTime,
			},
			Usage: &usage,
		})
	}

	// 更新 trace1
	trace1.Update(langfuse.TraceParams{
		Output: map[string]interface{}{
			"answer": "The weather in Beijing is sunny, 22°C.",
		},
	})

	// 创建第二个 trace (同一个 session)
	trace2, err := client.CreateTrace(langfuse.TraceParams{
		Name:      ptr("test-trace-2"),
		UserID:    &userID,
		SessionID: &sessionID,
		Input: map[string]interface{}{
			"message": "What about Shanghai?",
		},
		Metadata: map[string]interface{}{
			"source": "fetch-example",
			"version": "1.0",
		},
		Tags: []string{"test", "fetch-example", "weather"},
	})
	if err != nil {
		log.Fatalf("Failed to create trace 2: %v", err)
	}
	fmt.Printf("Created Trace 2: %s\n", trace2.ID())

	trace2.Update(langfuse.TraceParams{
		Output: map[string]interface{}{
			"answer": "Shanghai is cloudy, 26°C.",
		},
	})

	// 刷新数据到服务器
	fmt.Println("\nFlushing data to Langfuse...")
	flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Flush(flushCtx); err != nil {
		log.Fatalf("Failed to flush: %v", err)
	}

	// 等待一下让服务器处理数据
	fmt.Println("Waiting for server to process data...")
	time.Sleep(2 * time.Second)

	fmt.Println("\n========================================")
	fmt.Println("=== Fetching Data ===")
	fmt.Println("========================================\n")

	// ==========================================
	// 示例 2: 获取单个 Trace
	// ==========================================
	fmt.Printf("--- Fetching Trace by ID: %s ---\n", trace1.ID())
	fetchedTrace, err := client.GetTrace(ctx, langfuse.GetTraceParams{
		TraceID: trace1.ID(),
	})
	if err != nil {
		log.Printf("Failed to fetch trace: %v", err)
	} else {
		fmt.Printf("Trace ID: %s\n", fetchedTrace.ID)
		if fetchedTrace.Name != nil {
			fmt.Printf("Name: %s\n", *fetchedTrace.Name)
		}
		if fetchedTrace.UserID != nil {
			fmt.Printf("User ID: %s\n", *fetchedTrace.UserID)
		}
		if fetchedTrace.SessionID != nil {
			fmt.Printf("Session ID: %s\n", *fetchedTrace.SessionID)
		}
		fmt.Printf("Timestamp: %s\n", fetchedTrace.Timestamp)
		fmt.Printf("Tags: %v\n", fetchedTrace.Tags)

		// 显示 Input
		if len(fetchedTrace.Input) > 0 {
			inputJSON, _ := json.MarshalIndent(fetchedTrace.Input, "  ", "  ")
			fmt.Printf("Input:\n  %s\n", inputJSON)
		}

		// 显示 Output
		if len(fetchedTrace.Output) > 0 {
			outputJSON, _ := json.MarshalIndent(fetchedTrace.Output, "  ", "  ")
			fmt.Printf("Output:\n  %s\n", outputJSON)
		}

		// 显示 Metadata
		if len(fetchedTrace.Metadata) > 0 {
			metadataJSON, _ := json.MarshalIndent(fetchedTrace.Metadata, "  ", "  ")
			fmt.Printf("Metadata:\n  %s\n", metadataJSON)
		}

		// 显示 Observations
		fmt.Printf("Observations: %d\n", len(fetchedTrace.Observations))
		for i, obs := range fetchedTrace.Observations {
			fmt.Printf("  [%d] Type: %s", i+1, obs.Type)
			if obs.Name != nil {
				fmt.Printf(", Name: %s", *obs.Name)
			}
			if obs.Model != nil {
				fmt.Printf(", Model: %s", *obs.Model)
			}
			if obs.Usage != nil {
				fmt.Printf(", Usage: input=%d, output=%d",
					ptrToInt(obs.Usage.Input),
					ptrToInt(obs.Usage.Output))
			}
			fmt.Println()
		}

		// 显示 Scores
		fmt.Printf("Scores: %d\n", len(fetchedTrace.Scores))
	}

	// ==========================================
	// 示例 3: 获取 Session 下的所有 Traces
	// ==========================================
	fmt.Printf("\n--- Fetching Session: %s ---\n", sessionID)
	session, err := client.GetSession(ctx, langfuse.GetSessionParams{
		SessionID: sessionID,
	})
	if err != nil {
		log.Printf("Failed to fetch session: %v", err)
	} else {
		fmt.Printf("Session ID: %s\n", session.ID)
		fmt.Printf("Created At: %s\n", session.CreatedAt)
		fmt.Printf("Total Traces: %d\n", len(session.Traces))

		for i, trace := range session.Traces {
			fmt.Printf("\n  Trace %d:\n", i+1)
			fmt.Printf("    ID: %s\n", trace.ID)
			if trace.Name != nil {
				fmt.Printf("    Name: %s\n", *trace.Name)
			}
			fmt.Printf("    Timestamp: %s\n", trace.Timestamp)
			if len(trace.Input) > 0 {
				inputMsg := trace.Input["message"]
				fmt.Printf("    Input Message: %v\n", inputMsg)
			}
			if len(trace.Output) > 0 {
				outputAnswer := trace.Output["answer"]
				fmt.Printf("    Output Answer: %v\n", outputAnswer)
			}
			fmt.Printf("    Observations: %d\n", len(trace.Observations))
		}
	}

	// ==========================================
	// 示例 4: 列出 Traces (分页查询)
	// ==========================================
	fmt.Println("\n--- Listing Traces (Paginated) ---")
	page := 1
	limit := 10
	tracesList, err := client.ListTraces(ctx, langfuse.ListTracesParams{
		Page:      &page,
		Limit:     &limit,
		UserID:    &userID,
		SessionID: &sessionID,
		Tags:      []string{"fetch-example"},
	})
	if err != nil {
		log.Printf("Failed to list traces: %v", err)
	} else {
		fmt.Printf("Page: %d/%d\n", tracesList.Meta.Page, tracesList.Meta.TotalPages)
		fmt.Printf("Total Items: %d\n", tracesList.Meta.TotalItems)
		fmt.Printf("Items in this page: %d\n", len(tracesList.Data))

		for i, trace := range tracesList.Data {
			fmt.Printf("\n  [%d] Trace ID: %s\n", i+1, trace.ID)
			if trace.Name != nil {
				fmt.Printf("      Name: %s\n", *trace.Name)
			}
			if trace.UserID != nil {
				fmt.Printf("      User ID: %s\n", *trace.UserID)
			}
			fmt.Printf("      Tags: %v\n", trace.Tags)
			fmt.Printf("      Observations: %d\n", len(trace.Observations))
		}
	}

	// ==========================================
	// 示例 5: 使用 Trace 数据进行 Replay
	// ==========================================
	fmt.Println("\n========================================")
	fmt.Println("=== Replay Context Example ===")
	fmt.Println("========================================\n")

	if fetchedTrace != nil {
		fmt.Println("Reconstructing conversation from fetched trace:")

		// 从 trace 重建对话上下文
		messages := []map[string]interface{}{}

		// 从 Input 提取用户消息
		if userMsg, ok := fetchedTrace.Input["message"].(string); ok {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": userMsg,
			})
		}

		// 从 Output 提取助手回复
		if assistantMsg, ok := fetchedTrace.Output["answer"].(string); ok {
			messages = append(messages, map[string]interface{}{
				"role":    "assistant",
				"content": assistantMsg,
			})
		}

		// 从 Observations 提取更多信息
		for _, obs := range fetchedTrace.Observations {
			if obs.Type == "GENERATION" {
				// Generation 包含实际的 LLM 输入输出
				if obs.Input != nil {
					fmt.Printf("LLM Input (from Generation):\n")
					inputJSON, _ := json.MarshalIndent(obs.Input, "  ", "  ")
					fmt.Printf("  %s\n", inputJSON)
				}
				if obs.Output != nil {
					fmt.Printf("LLM Output (from Generation):\n")
					outputJSON, _ := json.MarshalIndent(obs.Output, "  ", "  ")
					fmt.Printf("  %s\n", outputJSON)
				}
				if obs.Usage != nil {
					fmt.Printf("Token Usage: input=%d, output=%d, total=%d\n",
						ptrToInt(obs.Usage.Input),
						ptrToInt(obs.Usage.Output),
						ptrToInt(obs.Usage.Total))
				}
			}
		}

		fmt.Println("\nReconstructed Messages:")
		for i, msg := range messages {
			msgJSON, _ := json.MarshalIndent(msg, "  ", "  ")
			fmt.Printf("Message %d:\n  %s\n", i+1, msgJSON)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("=== Example Complete ===")
	fmt.Println("========================================")
	fmt.Printf("\nView your data in Langfuse UI:\n%s\n", baseURL)
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("Trace 1 ID: %s\n", trace1.ID())
	fmt.Printf("Trace 2 ID: %s\n", trace2.ID())
}

// Helper function to convert *int to int
func ptrToInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
