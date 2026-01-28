package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	langfuse "github.com/langfuse/langfuse-go/langfuse"
	"github.com/sashabaranov/go-openai"
)

func ptr[T any](v T) *T { return &v }

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

type Config struct {
	LangfusePublicKey string
	LangfuseSecretKey string
	LangfuseBaseURL   string
	OpenAIKey         string
	OpenAIBaseURL     string
	OpenAIModel       string
}

func LoadConfig() *Config {
	return &Config{
		LangfusePublicKey: getEnv("LANGFUSE_PUBLIC_KEY", "pk-lf-613f327a-210d-492d-9b0e-1d15ff456dba"),
		LangfuseSecretKey: getEnv("LANGFUSE_SECRET_KEY", "sk-lf-dd707fbc-d87d-497a-86c0-7a464af5cfca"),
		LangfuseBaseURL:   getEnv("LANGFUSE_BASE_URL", "http://192.168.0.55:3000"),
		OpenAIKey:         getEnv("OPENAI_API_KEY", "sk-04c945475f0b481faab28d27521b9192"),
		OpenAIBaseURL:     getEnv("OPENAI_BASE_URL", "https://api.deepseek.com/v1"),
		OpenAIModel:       getEnv("OPENAI_MODEL", "deepseek-chat"),
	}
}

// ============================================
// Replay Context - 用于逆向重建完整上下文
// ============================================

// ReplayContext 存储完整会话上下文，支持 replay 功能
type ReplayContext struct {
	// 会话标识
	SessionID   string    `json:"session_id"`
	UserID      string    `json:"user_id"`
	TraceID     string    `json:"trace_id"`
	Timestamp   time.Time `json:"timestamp"`

	// 模型配置
	ModelConfig ModelConfig `json:"model_config"`

	// 系统提示
	SystemPrompt SystemPrompt `json:"system_prompt"`

	// 工具定义
	Tools []ToolDefinition `json:"tools"`

	// 对话历史（按时间顺序）
	ConversationHistory []ConversationTurn `json:"conversation_history"`

	// 会话元数据
	Metadata SessionMetadata `json:"metadata"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Model           string                 `json:"model"`
	BaseURL         string                 `json:"base_url"`
	Temperature     float64                `json:"temperature"`
	MaxTokens       int                    `json:"max_tokens"`
	TopP            float64                `json:"top_p"`
	FrequencyPenalty float64               `json:"frequency_penalty"`
	PresencePenalty  float64               `json:"presence_penalty"`
	ExtraParams     map[string]interface{} `json:"extra_params,omitempty"`
}

// SystemPrompt 系统提示
type SystemPrompt struct {
	Content   string            `json:"content"`
	Role      string            `json:"role"` // system, user, etc.
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string                 `json:"type"` // function
	Function map[string]interface{} `json:"function"`
}

// ConversationTurn 对话轮次
type ConversationTurn struct {
	// 轮次信息
	Round       int    `json:"round"`
	Timestamp   string `json:"timestamp"`
	TurnID      string `json:"turn_id"` // trace ID for this turn

	// 用户输入
	UserInput UserMessage `json:"user_input"`

	// LLM 响应
	LLMResponse LLMResponse `json:"llm_response"`

	// 工具调用（如果有）
	ToolCalls []ToolCallExecution `json:"tool_calls,omitempty"`

	// Token 使用
	TokenUsage TokenUsage `json:"token_usage"`
}

// UserMessage 用户消息
type UserMessage struct {
	Role    string `json:"role"` // user
	Content string `json:"content"`
}

// LLMResponse LLM 响应
type LLMResponse struct {
	Role         string `json:"role"` // assistant
	Content      string `json:"content"`
	ToolCalls    bool   `json:"tool_calls"`
	Reasoning    string `json:"reasoning,omitempty"`    // 模型的推理过程（如果有）
	FinishReason string `json:"finish_reason,omitempty"` // stop, tool_calls, length, etc.
}

// ToolCallExecution 工具调用执行
type ToolCallExecution struct {
	ToolName  string                 `json:"tool_name"`
	ToolID    string                 `json:"tool_id"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result"`
	StartTime string                 `json:"start_time"`
	EndTime   string                 `json:"end_time"`
	DurationMs int64                 `json:"duration_ms"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

// TokenUsage Token 使用统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SessionMetadata 会话元数据
type SessionMetadata struct {
	Environment         string            `json:"environment"`
	Tags                []string          `json:"tags"`
	CustomFields        map[string]any    `json:"custom_fields,omitempty"`
	ResponseTimeMs      int64             `json:"response_time_ms"`
	TotalCost           float64           `json:"total_cost,omitempty"`
	AdditionalInfo      map[string]any    `json:"additional_info,omitempty"`
}

// ============================================
// 上下文构建器
// ============================================

// ContextBuilder 用于构建 replay context
type ContextBuilder struct {
	sessionID       string
	userID          string
	modelConfig     ModelConfig
	systemPrompt    SystemPrompt
	tools           []ToolDefinition
	conversationCtx []ConversationTurn
	baseURL         string
}

// NewContextBuilder 创建新的上下文构建器
func NewContextBuilder(userID, baseURL, model string) *ContextBuilder {
	return &ContextBuilder{
		userID:      userID,
		baseURL:     baseURL,
		modelConfig: ModelConfig{
			Model:       model,
			BaseURL:     baseURL,
			Temperature: 0.7,
			MaxTokens:   2000,
		},
		conversationCtx: make([]ConversationTurn, 0),
	}
}

// SetSessionID 设置会话 ID
func (b *ContextBuilder) SetSessionID(sessionID string) *ContextBuilder {
	b.sessionID = sessionID
	return b
}

// SetSystemPrompt 设置系统提示
func (b *ContextBuilder) SetSystemPrompt(content string, metadata map[string]any) *ContextBuilder {
	b.systemPrompt = SystemPrompt{
		Content:  content,
		Role:     "system",
		Metadata: metadata,
	}
	return b
}

// SetTools 设置工具定义
func (b *ContextBuilder) SetTools(tools []openai.Tool) *ContextBuilder {
	b.tools = make([]ToolDefinition, len(tools))
	for i, tool := range tools {
		b.tools[i] = ToolDefinition{
			Type:     string(tool.Type),
			Function: map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		}
	}
	return b
}

// SetModelParams 设置模型参数
func (b *ContextBuilder) SetModelParams(temperature float64, maxTokens int) *ContextBuilder {
	b.modelConfig.Temperature = temperature
	b.modelConfig.MaxTokens = maxTokens
	return b
}

// AddTurn 添加一个对话轮次
func (b *ContextBuilder) AddTurn(round int, userMsg string, assistantResp string, toolCalls []ToolCallExecution, tokenUsage TokenUsage, traceID string) *ContextBuilder {
	turn := ConversationTurn{
		Round:     round,
		Timestamp: time.Now().Format(time.RFC3339Nano),
		TurnID:    traceID,
		UserInput: UserMessage{
			Role:    "user",
			Content: userMsg,
		},
		LLMResponse: LLMResponse{
			Role:      "assistant",
			Content:   assistantResp,
			ToolCalls: len(toolCalls) > 0,
		},
		ToolCalls:  toolCalls,
		TokenUsage: tokenUsage,
	}
	b.conversationCtx = append(b.conversationCtx, turn)
	return b
}

// Build 构建 ReplayContext
func (b *ContextBuilder) Build(traceID string) ReplayContext {
	return ReplayContext{
		SessionID:           b.sessionID,
		UserID:              b.userID,
		TraceID:             traceID,
		Timestamp:           time.Now(),
		ModelConfig:         b.modelConfig,
		SystemPrompt:        b.systemPrompt,
		Tools:               b.tools,
		ConversationHistory: b.conversationCtx,
		Metadata: SessionMetadata{
			Tags: []string{"replay", "chat"},
		},
	}
}

// ToOpenAIMessages 将上下文转换为 OpenAI 消息格式（用于 replay）
func (r *ReplayContext) ToOpenAIMessages() []openai.ChatCompletionMessage {
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: r.SystemPrompt.Content,
		},
	}

	for _, turn := range r.ConversationHistory {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: turn.UserInput.Content,
		})

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: turn.LLMResponse.Content,
		})

		// 添加工具调用消息
		for _, tc := range turn.ToolCalls {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleTool,
				Content: tc.Result,
				ToolCallID: tc.ToolID,
			})
		}
	}

	return messages
}

// ToJSON 序列化为 JSON
func (r *ReplayContext) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ============================================
// 主程序
// ============================================

func main() {
	cfg := LoadConfig()
	if cfg.LangfusePublicKey == "" || cfg.LangfuseSecretKey == "" || cfg.OpenAIKey == "" {
		log.Fatal("LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY and OPENAI_API_KEY must be set")
	}

	// Init Langfuse with replay context enabled
	langfuseConfig := langfuse.DefaultConfig()
	langfuseConfig.PublicKey = cfg.LangfusePublicKey
	langfuseConfig.SecretKey = cfg.LangfuseSecretKey
	langfuseConfig.BaseURL = cfg.LangfuseBaseURL
	langfuseConfig.Debug = true
	langfuseConfig.MetricsEnabled = true

	langfuseClient, err := langfuse.NewClient(langfuseConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer langfuseClient.Close()

	// Init OpenAI
	openaiConfig := openai.DefaultConfig(cfg.OpenAIKey)
	openaiConfig.BaseURL = cfg.OpenAIBaseURL
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	// 初始化上下文构建器
	ctx := context.Background()
	userID := "user-replay-demo-123"
	sessionID := uuid.New().String()
	conversationStartTime := time.Now()

	contextBuilder := NewContextBuilder(userID, cfg.OpenAIBaseURL, cfg.OpenAIModel)
	contextBuilder.SetSessionID(sessionID)
	contextBuilder.SetSystemPrompt(
		"Use get_weather for weather queries, calculator for math.",
		map[string]any{"version": "1.0", "purpose": "tool-use-demo"},
	)

	fmt.Println("========================================")
	fmt.Println("  Replay-Enabled Chat Demo")
	fmt.Println("========================================")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Println("----------------------------------------")

	// 定义工具
	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "get_weather",
				Description: "Get weather for a city",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"city": map[string]any{"type": "string", "description": "City name"},
					},
					"required": []string{"city"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "calculator",
				Description: "Calculate: add, subtract, multiply, divide",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"operation": map[string]any{"type": "string", "enum": []string{"add", "subtract", "multiply", "divide"}},
						"a":         map[string]any{"type": "number"},
						"b":         map[string]any{"type": "number"},
					},
					"required": []string{"operation", "a", "b"},
				},
			},
		},
	}
	contextBuilder.SetTools(tools)

	// 会话状态
	messages := []openai.ChatCompletionMessage{{
		Role:    openai.ChatMessageRoleSystem,
		Content: "Use get_weather for weather queries, calculator for math.",
	}}

	// 模拟 RAG 检索函数
	simulateRAGRetrieval := func(query, category string) []map[string]interface{} {
		docs := []map[string]interface{}{}

		switch category {
		case "weather":
			docs = append(docs, map[string]interface{}{
				"doc_id":       "weather-001",
				"content":      "北京天气数据：实时温度22°C，晴朗，湿度45%，风向北风2级",
				"source":       "weather-api",
				"score":        0.95,
				"retrieved_at": time.Now().Format(time.RFC3339),
			})
			docs = append(docs, map[string]interface{}{
				"doc_id":       "weather-002",
				"content":      "上海天气数据：实时温度26°C，多云，湿度60%，风向南风3级",
				"source":       "weather-api",
				"score":        0.87,
				"retrieved_at": time.Now().Format(time.RFC3339),
			})
		case "math":
			docs = append(docs, map[string]interface{}{
				"doc_id":       "math-001",
				"content":      "基础数学运算：加法规则，1+1=2，2+2=4，等等",
				"source":       "math-kb",
				"score":        0.98,
				"retrieved_at": time.Now().Format(time.RFC3339),
			})
			docs = append(docs, map[string]interface{}{
				"doc_id":       "math-002",
				"content":      "计算器工具使用说明：支持加、减、乘、除四则运算",
				"source":       "tool-docs",
				"score":        0.85,
				"retrieved_at": time.Now().Format(time.RFC3339),
			})
		default:
			docs = append(docs, map[string]interface{}{
				"doc_id":       "general-001",
				"content":      "通用知识库：关于时间、日期和一般性问题回答指南",
				"source":       "general-kb",
				"score":        0.75,
				"retrieved_at": time.Now().Format(time.RFC3339),
			})
		}

		return docs
	}

	// 工具执行函数
	executeTool := func(toolCall openai.ToolCall) string {
		var args map[string]any
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

		switch toolCall.Function.Name {
		case "get_weather":
			city, _ := args["city"].(string)
			weatherData := map[string]string{
				"北京":   "22°C, 晴天",
				"上海":   "26°C, 多云",
				"Tokyo":  "20°C, Rainy",
			}
			if w, ok := weatherData[city]; ok {
				return fmt.Sprintf("Weather in %s: %s", city, w)
			}
			return fmt.Sprintf("Weather for %s: unavailable", city)

		case "calculator":
			op, _ := args["operation"].(string)
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			var result float64
			switch op {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return "Error: division by zero"
				}
				result = a / b
			}
			return fmt.Sprintf("%.2f", result)

		default:
			return fmt.Sprintf("Unknown tool: %s", toolCall.Function.Name)
		}
	}

	// 测试问题
	questions := []struct {
		question string
		category string
	}{
		{"北京的天气怎么样？", "weather"},
		{"1加1等于多少？", "math"},
		{"今年是哪一年？", "general"},
	}

	// 处理每个问题
	for i, qa := range questions {
		fmt.Printf("\n[Round %d] Question: %s\n", i+1, qa.question)

		traceStartTime := time.Now()

		// 创建 trace，包含 replay_context
		trace, err := langfuseClient.CreateTrace(langfuse.TraceParams{
			Name:      ptr(fmt.Sprintf("chat-round-%d-%s", i+1, qa.category)),
			UserID:    &userID,
			SessionID: &sessionID,
			Metadata: map[string]any{
				"model":      cfg.OpenAIModel,
				"round":      i + 1,
				"category":   qa.category,
				"has_context": true, // 标记此 trace 包含 replay 上下文
			},
			Tags: []string{"chat", qa.category, "replay-enabled"},
		})
		if err != nil {
			log.Printf("Warning: failed to create trace: %v", err)
			continue
		}

		// 添加用户消息
		messages = append(messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser, Content: qa.question,
		})

		// ========== 模拟 RAG 检索 Span ==========
		ragStartTime := time.Now()
		ragSpanID, _ := trace.CreateSpan(langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Name:      ptr(fmt.Sprintf("rag-retrieval-round-%d", i+1)),
				Input:     map[string]any{"query": qa.question, "retriever": "vector-store"},
				StartTime: &ragStartTime,
				Metadata: map[string]any{
					"retrieval_method":    "semantic_search",
					"top_k":               3,
					"index_name":          "knowledge-base-v1",
					"retrieval_duration_ms": 100, // 预期检索耗时
				},
			},
		})

		// 模拟 RAG 检索延迟 (100ms)
		time.Sleep(100 * time.Millisecond)

		// 模拟检索结果
		rerankedDocs := simulateRAGRetrieval(qa.question, qa.category)

		ragEndTime := time.Now()
		ragDuration := ragEndTime.Sub(ragStartTime)
		langfuseClient.UpdateSpan(ragSpanID, langfuse.SpanParams{
			ObservationParams: langfuse.ObservationParams{
				Output: map[string]any{
					"retrieved_documents": rerankedDocs,
					"total_docs_found":    len(rerankedDocs),
					"query_rewrite":       qa.question, // 假设进行了查询重写
				},
			},
			EndTime: &ragEndTime,
		})

		fmt.Printf("[Round %d] RAG: Retrieved %d documents in %v\n", i+1, len(rerankedDocs), ragDuration)
		// =========================================

		// 创建 generation，input 只包含实际的 LLM messages（保持纯净，便于 replay）
		genStartTime := time.Now()
		genParams := langfuse.GenerationParams{
			SpanParams: langfuse.SpanParams{
				ObservationParams: langfuse.ObservationParams{
					Name:      ptr(fmt.Sprintf("llm-generation-round-%d", i+1)),
					StartTime: &genStartTime,
					// 只包含实际传给 LLM 的输入，便于后续 replay 直接使用
					Input:     messages,
				},
			},
		}
		genParams.Model = &cfg.OpenAIModel
		genParams.ModelParameters = map[string]any{"temperature": 0.7, "max_tokens": 2000}

		genID, _ := trace.CreateGeneration(genParams)

		// 调用 OpenAI
		resp, err := openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: cfg.OpenAIModel, Messages: messages, Tools: tools,
		})
		if err != nil {
			genEndTime := time.Now()
			updateParams := langfuse.GenerationParams{
				SpanParams: langfuse.SpanParams{
					ObservationParams: langfuse.ObservationParams{
						StatusMessage: ptr(err.Error()),
						Level:         ptr(langfuse.LevelError),
					},
					EndTime: &genEndTime,
				},
			}
			langfuseClient.UpdateGeneration(genID, updateParams)
			log.Printf("Error: %v\n", err)
			continue
		}

		assistantMsg := resp.Choices[0].Message
		messages = append(messages, assistantMsg)
		usage := resp.Usage

		// 处理工具调用
		var finalAnswer string
		toolCallsUsed := false
		var toolExecutions []ToolCallExecution
		var toolCallsDetailed []map[string]any
		var toolResults map[string]string

		if len(assistantMsg.ToolCalls) > 0 {
			toolCallsUsed = true
			toolResults = make(map[string]string)
			toolCallsDetailed = make([]map[string]any, 0, len(assistantMsg.ToolCalls))
			toolExecutions = make([]ToolCallExecution, 0, len(assistantMsg.ToolCalls))

			for j, tc := range assistantMsg.ToolCalls {
				toolStartTime := time.Now()
				result := executeTool(tc)
				toolEndTime := time.Now()

				// 解析参数
				var argsMap map[string]any
				json.Unmarshal([]byte(tc.Function.Arguments), &argsMap)

				// 创建 Tool 观测
				trace.CreateTool(langfuse.ToolParams{
					SpanParams: langfuse.SpanParams{
						ObservationParams: langfuse.ObservationParams{
							Name: ptr(fmt.Sprintf("tool-%s", tc.Function.Name)),
							Input: map[string]any{
								"tool_name": tc.Function.Name,
								"tool_id":   tc.ID,
								"arguments": argsMap,
							},
							Output: map[string]any{
								"result": result,
							},
							StartTime: &toolStartTime,
							Metadata: map[string]any{
								"tool_index":   j,
								"round":        i + 1,
								"tool_call_id": tc.ID,
							},
						},
						EndTime: &toolEndTime,
					},
				})

				// 记录工具执行（用于 replay）
				toolExecutions = append(toolExecutions, ToolCallExecution{
					ToolName:  tc.Function.Name,
					ToolID:    tc.ID,
					Arguments: argsMap,
					Result:    result,
					StartTime: toolStartTime.Format(time.RFC3339Nano),
					EndTime:   toolEndTime.Format(time.RFC3339Nano),
					DurationMs: toolEndTime.Sub(toolStartTime).Milliseconds(),
					Success:   true,
				})

				toolCallsDetailed = append(toolCallsDetailed, map[string]any{
					"tool":      tc.Function.Name,
					"tool_id":   tc.ID,
					"arguments": argsMap,
					"result":    result,
					"index":     j,
				})

				toolResults[tc.Function.Name] = result
				messages = append(messages, openai.ChatCompletionMessage{
					Role: openai.ChatMessageRoleTool, Content: result, ToolCallID: tc.ID,
				})
			}

			// 第二次 API 调用
			resp2, err := openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
				Model: cfg.OpenAIModel, Messages: messages, Tools: tools,
			})
			if err != nil {
				log.Printf("Error on second API call: %v\n", err)
				continue
			}

			finalMsg := resp2.Choices[0].Message
			messages = append(messages, finalMsg)
			finalAnswer = finalMsg.Content

			// 更新 generation
			genEndTime := time.Now()
			usageLangfuse := langfuse.Usage{
				Input:  ptr(usage.PromptTokens + resp2.Usage.PromptTokens),
				Output: ptr(usage.CompletionTokens + resp2.Usage.CompletionTokens),
				Total:  ptr(usage.TotalTokens + resp2.Usage.TotalTokens),
			}
			updateParams := langfuse.GenerationParams{
				SpanParams: langfuse.SpanParams{
					ObservationParams: langfuse.ObservationParams{
						Output: map[string]any{
							"tool_calls":          toolResults,
							"tool_calls_detailed": toolCallsDetailed,
							"final_response":      finalAnswer,
						},
					},
					EndTime: &genEndTime,
				},
				Usage: &usageLangfuse,
			}
			langfuseClient.UpdateGeneration(genID, updateParams)

		} else {
			// 无工具调用
			finalAnswer = assistantMsg.Content
			genEndTime := time.Now()
			usageLangfuse := langfuse.Usage{
				Input:  ptr(usage.PromptTokens),
				Output: ptr(usage.CompletionTokens),
				Total:  ptr(usage.TotalTokens),
			}
			updateParams := langfuse.GenerationParams{
				SpanParams: langfuse.SpanParams{
					ObservationParams: langfuse.ObservationParams{
						Output: map[string]any{
							"content": finalAnswer,
						},
					},
					EndTime: &genEndTime,
				},
				Usage: &usageLangfuse,
			}
			langfuseClient.UpdateGeneration(genID, updateParams)
		}

		// 构建当前轮次的上下文
		tokenUsage := TokenUsage{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.TotalTokens,
		}

		contextBuilder.AddTurn(
			i+1,
			qa.question,
			finalAnswer,
			toolExecutions,
			tokenUsage,
			trace.ID(),
		)

		// 构建 ReplayContext
		replayCtx := contextBuilder.Build(trace.ID())

		// 将 replay context 存储到 trace 的 output 中
		traceOutput := map[string]any{
			"answer":       finalAnswer,
			"round":        i + 1,
			"tool_calls":   toolCallsUsed,
			"total_tokens": usage.TotalTokens,
		}

		// 核心：存储 replay_context
		traceOutput["replay_context"] = replayCtx

		trace.Update(langfuse.TraceParams{
			Output: traceOutput,
			Metadata: map[string]any{
				"success":          true,
				"tool_calls_used":  toolCallsUsed,
				"response_time_ms": time.Since(traceStartTime).Milliseconds(),
				"tokens_used":      usage.TotalTokens,
				"tool_count":       len(assistantMsg.ToolCalls),
				"replay_enabled":   true, // 标记此 trace 支持 replay
				"has_rag":          true, // 标记此 trace 包含 RAG
			},
		})

		fmt.Printf("[Round %d] Answer: %s\n", i+1, finalAnswer)
		if toolCallsUsed {
			fmt.Printf("[Round %d] Tools used: yes (%d)\n", i+1, len(assistantMsg.ToolCalls))
		}
		fmt.Printf("[Round %d] Tokens: %d\n", i+1, usage.TotalTokens)
		fmt.Printf("[Round %d] Trace ID: %s\n", i+1, trace.ID())
	}

	// 创建 session summary trace
	summaryTrace, _ := langfuseClient.CreateTrace(langfuse.TraceParams{
		Name:      ptr("session-summary"),
		UserID:    &userID,
		SessionID: &sessionID,
		Metadata: map[string]any{
			"total_rounds":     len(questions),
			"session_start":    conversationStartTime.Format(time.RFC3339),
			"session_end":      time.Now().Format(time.RFC3339),
			"session_duration": time.Since(conversationStartTime).Milliseconds(),
			"replay_enabled":   true,
		},
		Tags: []string{"summary", "session", "replay-enabled"},
	})

	// 在 summary 中也存储完整的 replay context
	finalReplayCtx := contextBuilder.Build(summaryTrace.ID())
	finalContextJSON, _ := finalReplayCtx.ToJSON()

	summaryTrace.Update(langfuse.TraceParams{
		Output: map[string]any{
			"completed_rounds": len(questions),
			"session_id":       sessionID,
			"replay_context":   finalReplayCtx,
			"replay_context_json": finalContextJSON, // 同时存储 JSON 字符串便于导出
		},
	})

	// Flush
	fmt.Println("\n========================================")
	fmt.Println("  Flushing events to Langfuse...")
	fmt.Println("========================================")

	flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	langfuseClient.Flush(flushCtx)

	// 显示 replay 上下文示例
	fmt.Println("\n--- Replay Context Example (Last Round) ---")
	if len(finalReplayCtx.ConversationHistory) > 0 {
		lastTurn := finalReplayCtx.ConversationHistory[len(finalReplayCtx.ConversationHistory)-1]
		fmt.Printf("Round: %d\n", lastTurn.Round)
		fmt.Printf("User Input: %s\n", lastTurn.UserInput.Content)
		fmt.Printf("LLM Response: %s\n", lastTurn.LLMResponse.Content)
		if len(lastTurn.ToolCalls) > 0 {
			fmt.Printf("Tool Calls: %d\n", len(lastTurn.ToolCalls))
			for _, tc := range lastTurn.ToolCalls {
				fmt.Printf("  - %s: %v -> %s\n", tc.ToolName, tc.Arguments, tc.Result)
			}
		}
	}

	// 显示如何使用 replay context
	fmt.Println("\n--- How to Use Replay Context ---")
	fmt.Println("1. From Langfuse UI, copy the 'replay_context' field from any trace")
	fmt.Println("2. Unmarshal the JSON into ReplayContext struct")
	fmt.Println("3. Call replayContext.ToOpenAIMessages() to rebuild conversation")
	fmt.Println("4. Use the messages with OpenAI API to replay the conversation")
	fmt.Println("\nExample code:")
	fmt.Println("```go")
	fmt.Println("var replayCtx ReplayContext")
	fmt.Println("json.Unmarshal(replayContextJSON, &replayCtx)")
	fmt.Println("messages := replayCtx.ToOpenAIMessages()")
	fmt.Println("resp := openaiClient.CreateChatCompletion(ctx, messages)")
	fmt.Println("```")

	// 显示指标
	fmt.Println("\n--- Metrics Snapshot ---")
	snapshot := langfuseClient.GetMetrics()
	fmt.Printf("%s\n", snapshot.String())
	fmt.Printf("Success Rate: %.2f%%\n", snapshot.SuccessRate())

	fmt.Println("\n========================================")
	fmt.Println("Test completed!")
	fmt.Println("========================================")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("Total traces: %d\n", len(questions)+1)
	fmt.Printf("\nView at: %s\n", cfg.LangfuseBaseURL)
	fmt.Printf("Filter by session ID: %s\n", sessionID)
}
