package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	langfuse "github.com/lvow2022/langfuse-gosdk/langfuse"
	openai "github.com/sashabaranov/go-openai"
)

type WeatherArgs struct {
	City string `json:"city"`
}

func getWeather(city string) string {
	return fmt.Sprintf("The weather in %s is sunny, 26°C", city)
}

func main() {
	openaiKey := os.Getenv("OPENAI_API_KEY")
	openaiBaseURL := os.Getenv("OPENAI_BASE_URL")
	openaiModel := os.Getenv("OPENAI_MODEL")

	if openaiKey == "" {
		log.Fatal("OPENAI_API_KEY is required. Please set it in .env file")
	}

	langfuseConfig := langfuse.DefaultConfig()
	langfuseConfig.PublicKey = os.Getenv("LANGFUSE_PUBLIC_KEY")
	langfuseConfig.SecretKey = os.Getenv("LANGFUSE_SECRET_KEY")
	langfuseConfig.BaseURL = os.Getenv("LANGFUSE_BASE_URL")
	langfuseConfig.Debug = true

	langfuseClient, err := langfuse.NewClient(langfuseConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer langfuseClient.Close()

	// Initialize OpenAI client with config
	openaiConfig := openai.DefaultConfig(openaiKey)
	openaiConfig.BaseURL = openaiBaseURL
	client := openai.NewClientWithConfig(openaiConfig)
	ctx := context.Background()

	fmt.Printf("OpenAI Model: %s\n", openaiModel)
	fmt.Printf("OpenAI Base URL: %s\n", openaiBaseURL)
	fmt.Println("----------------------------------------")

	// Session monitoring
	sessionID := "session-weather-demo-" + time.Now().Format("20060102-150405")
	userID := "user-123"

	fmt.Println("========================================")
	fmt.Println("Session Monitoring Demo")
	fmt.Println("========================================")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("User ID: %s\n", userID)
	fmt.Println("----------------------------------------")

	// Create a trace for this conversation with SessionID
	trace, err := langfuseClient.CreateTrace(langfuse.TraceParams{
		Name:   langfuse.Ptr("weather-tool-call-demo"),
		UserID: &userID,
		Metadata: map[string]any{
			"model":           openaiModel,
			"session_type":    "weather_query",
			"conversation_id": "conv-001",
		},
		Tags: []string{"demo", "tool-calling", "weather", "session-tracked"},
	})
	if err != nil {
		log.Printf("Warning: failed to create trace: %v", err)
	}
	fmt.Printf("Trace ID: %s\n", trace.ID())

	// Step 1: user message
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "What's the weather in Beijing today?",
		},
	}

	// Step 2: define tool schema
	tools := []openai.Tool{
		{
			Type:     openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather by city name",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"city": { "type": "string" }
					},
					"required": ["city"]
				}`),
			},
		},
	}

	// Step 3: request model
	genStartTime := time.Now()
	genParams := langfuse.GenerationParams{
		SpanParams: langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name:      langfuse.Ptr("llm-generation"),
				StartTime: &genStartTime,
				Input: map[string]any{
					"messages": messages,
					"tools":    tools,
				},
			},
		},
	}
	genParams.Model = &openaiModel
	genParams.ModelParameters = map[string]any{"temperature": 0.7}

	genID, _ := trace.CreateGeneration(genParams)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    openaiModel,
		Messages: messages,
		Tools:    tools,
	})
	if err != nil {
		genEndTime := time.Now()
		level := langfuse.LevelError
		langfuseClient.UpdateGeneration(genID, langfuse.GenerationParams{
			SpanParams: langfuse.SpanParams{
				ObservationParams: langfuse.ObservationParams{
					StatusMessage: langfuse.Ptr(err.Error()),
					Level:         &level,
				},
				EndTime: &genEndTime,
			},
		})
		panic(err)
	}

	msg := resp.Choices[0].Message

	// Step 4: model decided to call tool?
	if len(msg.ToolCalls) > 0 {
		toolCall := msg.ToolCalls[0]

		fmt.Println("LLM wants to call tool:", toolCall.Function.Name)
		fmt.Println("Arguments:", toolCall.Function.Arguments)

		// parse tool args
		var args WeatherArgs
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

		// Create Tool observation for langfuse
		toolStartTime := time.Now()
		var argsMap map[string]any
		json.Unmarshal([]byte(toolCall.Function.Arguments), &argsMap)

		toolID, _ := trace.CreateTool(langfuse.ToolParams{
			SpanParams: langfuse.SpanParams{
				ObservationParams: langfuse.ObservationParams{
					Name:      langfuse.Ptr("tool-get_weather"),
					StartTime: &toolStartTime,
					Input: map[string]any{
						"tool_name": toolCall.Function.Name,
						"tool_id":   toolCall.ID,
						"arguments": argsMap,
					},
				},
			},
		})

		// Step 5: execute tool
		result := getWeather(args.City)
		toolEndTime := time.Now()

		// Update Tool observation with result
		langfuseClient.UpdateTool(toolID, langfuse.ToolParams{
			SpanParams: langfuse.SpanParams{
				ObservationParams: langfuse.ObservationParams{
					Output: map[string]any{
						"result": result,
					},
				},
				EndTime: &toolEndTime,
			},
		})

		// Step 6: append tool result to messages
		messages = append(messages, msg)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result,
			ToolCallID: toolCall.ID,
		})

		// Step 7: call model again to finalize response
		finalResp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    openaiModel,
			Messages: messages,
		})
		if err != nil {
			panic(err)
		}

		finalMsg := finalResp.Choices[0].Message
		finalEndTime := time.Now()

		// Update generation with complete information
		usage := langfuse.Usage{
			Input:  langfuse.Ptr(resp.Usage.PromptTokens + finalResp.Usage.PromptTokens),
			Output: langfuse.Ptr(resp.Usage.CompletionTokens + finalResp.Usage.CompletionTokens),
			Total:  langfuse.Ptr(resp.Usage.TotalTokens + finalResp.Usage.TotalTokens),
		}

		// Re-parse args for generation update
		json.Unmarshal([]byte(toolCall.Function.Arguments), &argsMap)

		langfuseClient.UpdateGeneration(genID, langfuse.GenerationParams{
			SpanParams: langfuse.SpanParams{
				ObservationParams: langfuse.ObservationParams{
					Output: map[string]any{
						"tool_calls": map[string]any{
							"tool_name": toolCall.Function.Name,
							"arguments": argsMap,
							"result":    result,
						},
						"final_response": finalMsg.Content,
					},
				},
				EndTime: &finalEndTime,
			},
			Usage: &usage,
		})

		// Update trace with output and session metadata
		trace.Update(langfuse.TraceParams{
			Output: map[string]any{
				"answer":       finalMsg.Content,
				"tool_used":    toolCall.Function.Name,
				"total_tokens": usage.Total,
			},
			Metadata: map[string]any{
				"success":          true,
				"tool_calls_used":  true,
				"response_time_ms": finalEndTime.Sub(genStartTime).Milliseconds(),
				"tokens_used":      *usage.Total,
				// Session 相关元数据
				"session_id":           sessionID,
				"session_round":        1,
				"session_total_rounds": 1,
			},
		})

		fmt.Println("\nFinal Answer:")
		fmt.Println(finalMsg.Content)
		fmt.Printf("\nTokens used: %d\n", *usage.Total)
		fmt.Printf("Response time: %v\n", finalEndTime.Sub(genStartTime))

		// Flush events to Langfuse
		fmt.Println("\nFlushing events to Langfuse...")
		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := langfuseClient.Flush(flushCtx); err != nil {
			log.Printf("Warning: failed to flush events: %v", err)
		}

		// Show metrics
		snapshot := langfuseClient.GetMetrics()
		fmt.Printf("Metrics: %s\n", snapshot.String())
		fmt.Printf("Success Rate: %.2f%%\n", snapshot.SuccessRate())

		sessionSummaryTrace, _ := langfuseClient.CreateTrace(langfuse.TraceParams{
			Name:      langfuse.Ptr("session-summary"),
			UserID:    &userID,
			SessionID: &sessionID,
			Metadata: map[string]any{
				"session_type":     "weather_query",
				"total_traces":     1,
				"session_duration": "single_query",
			},
			Tags: []string{"session", "summary", "monitoring"},
		})

		sessionSummaryTrace.Update(langfuse.TraceParams{
			Output: map[string]any{
				"session_id":      sessionID,
				"user_id":         userID,
				"total_traces":    1,
				"total_tokens":    *usage.Total,
				"tools_used":      []string{"get_weather"},
				"session_status":  "completed",
				"monitoring_type": "session_level",
			},
		})

		langfuseClient.Flush(context.Background())

		fmt.Println("\n========================================")
		fmt.Println("Session Summary Created")
		fmt.Println("========================================")
		fmt.Printf("Session Summary Trace ID: %s\n", sessionSummaryTrace.ID())
		fmt.Printf("Session ID: %s\n", sessionID)
		fmt.Printf("Total Traces in Session: 1\n")
		fmt.Printf("Total Tokens: %d\n", *usage.Total)
	}
}
