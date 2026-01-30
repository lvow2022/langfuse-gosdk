package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/lvow2022/langfuse-gosdk/langfuse"
)

func main() {
	godotenv.Load()
	fmt.Println("========================================")
	fmt.Println("  Langfuse Fetch Data Example")
	fmt.Println("========================================")
	// 1. 获取 trace
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

	traceID := "7ee0b92d-4b98-4022-b10a-5929f7b62ec8"
	fmt.Printf("Trace ID: %s\n\n", traceID)
	trace, err := client.GetTrace(ctx, langfuse.GetTraceParams{
		TraceID: traceID,
	})
	if err != nil {
		log.Printf("Failed to fetch trace: %v\n", err)
	}
	fmt.Println()

	// 2. 组装历史上下文
	// - 找到第一条 generation 记录
	// - 拼接 input + output，得到上下文
	var firstGeneration *langfuse.ObservationDetails
	for i := 0; i < len(trace.Observations); i++ {
		if trace.Observations[i].Type == "GENERATION" {
			firstGeneration = &trace.Observations[i]
			break
		}
	}

	if firstGeneration == nil {
		log.Fatalf("No generation found in trace")
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("First Generation Details:\n")
	fmt.Printf("========================================\n")
	fmt.Printf("ID: %s\n", firstGeneration.ID)
	if firstGeneration.Name != nil {
		fmt.Printf("Name: %s\n", *firstGeneration.Name)
	}
	if firstGeneration.Model != nil {
		fmt.Printf("Model: %s\n", *firstGeneration.Model)
	}
	fmt.Printf("Start Time: %s\n", firstGeneration.StartTime)
	if firstGeneration.EndTime != nil {
		fmt.Printf("End Time: %s\n", *firstGeneration.EndTime)
	}

	// Debug: 打印原始 Input 和 Output 类型
	fmt.Printf("\n[DEBUG] Input type: %T\n", firstGeneration.Input)
	fmt.Printf("[DEBUG] Output type: %T\n", firstGeneration.Output)

	// 组装上下文：input + output
	contextMessages := []map[string]any{}

	// 添加 input - 处理可能是数组的情况
	if firstGeneration.Input != nil {
		// 尝试将 Input 断言为数组
		if inputArray, ok := firstGeneration.Input.([]any); ok {
			// Input 是数组，遍历并添加每个消息
			fmt.Printf("[DEBUG] Input array length: %d\n", len(inputArray))
			for idx, item := range inputArray {
				if msgMap, ok := item.(map[string]any); ok {
					fmt.Printf("[DEBUG] Input[%d]: role=%v\n", idx, msgMap["role"])
					contextMessages = append(contextMessages, msgMap)
				}
			}
		} else {
			// Input 不是数组，作为单个消息处理
			inputMap := map[string]any{
				"role":    "user",
				"content": firstGeneration.Input,
			}
			contextMessages = append(contextMessages, inputMap)
		}
	}

	// 添加 output - 处理可能是数组的情况
	if firstGeneration.Output != nil {
		fmt.Printf("[DEBUG] Processing output...\n")
		// 尝试将 Output 断言为数组
		if outputArray, ok := firstGeneration.Output.([]any); ok {
			// Output 是数组，遍历并添加每个消息
			fmt.Printf("[DEBUG] Output is array, length: %d\n", len(outputArray))
			for _, item := range outputArray {
				if msgMap, ok := item.(map[string]any); ok {
					contextMessages = append(contextMessages, msgMap)
				}
			}
		} else if outputMap, ok := firstGeneration.Output.(map[string]any); ok {
			// Output 是单个对象
			fmt.Printf("[DEBUG] Output is map: %+v\n", outputMap)
			contextMessages = append(contextMessages, outputMap)
		} else {
			// Output 是其他类型，作为 assistant 消息的 content
			fmt.Printf("[DEBUG] Output is other type: %T\n", firstGeneration.Output)
			outputMsg := map[string]any{
				"role":    "assistant",
				"content": firstGeneration.Output,
			}
			contextMessages = append(contextMessages, outputMsg)
		}
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("Context Messages:\n")
	fmt.Printf("========================================\n")
	for i, msg := range contextMessages {
		fmt.Printf("Message %d:\n", i+1)

		// 打印 role
		if role, ok := msg["role"]; ok && role != nil {
			fmt.Printf("  Role: %v\n", role)
		} else {
			fmt.Printf("  Role: (missing)\n")
		}

		// 打印 content
		if content, ok := msg["content"]; ok {
			// 如果 content 是字符串，截断显示以便阅读
			if contentStr, ok := content.(string); ok {
				if len(contentStr) > 200 {
					fmt.Printf("  Content: %s... (truncated, total length: %d)\n", contentStr[:200], len(contentStr))
				} else {
					fmt.Printf("  Content: %s\n", contentStr)
				}
			} else {
				fmt.Printf("  Content: %v\n", content)
			}
		}

		// 打印 tool_calls（如果有）
		if toolCalls, ok := msg["tool_calls"]; ok && toolCalls != nil {
			fmt.Printf("  Tool Calls: %v\n", toolCalls)
		}

		// 打印 tool_call_id（如果有）
		if toolCallID, ok := msg["tool_call_id"]; ok && toolCallID != nil {
			fmt.Printf("  Tool Call ID: %v\n", toolCallID)
		}

		fmt.Println()
	}

	// 3. 发送请求
	fmt.Printf("\n========================================\n")
	fmt.Printf("Sending Request to Replay API\n")
	fmt.Printf("========================================\n")

	// 读取请求体模板文件
	templateData, err := os.ReadFile("examples/replay/request_template.json")
	if err != nil {
		log.Fatalf("Failed to read request template: %v", err)
	}

	// 解析原始请求体模板
	var requestBody map[string]any
	if err := json.Unmarshal(templateData, &requestBody); err != nil {
		log.Fatalf("Failed to parse request body template: %v", err)
	}

	// 替换 history 为 contextMessages
	requestBody["history"] = contextMessages

	// 序列化请求体
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		log.Fatalf("Failed to marshal request body: %v", err)
	}

	// 发送 POST 请求
	replayURL := "http://localhost:9001/api/v1/replay"
	fmt.Printf("Sending POST request to: %s\n", replayURL)
	fmt.Printf("Request body size: %d bytes\n", len(requestJSON))

	resp, err := http.Post(replayURL, "application/json", bytes.NewBuffer(requestJSON))
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("========================================\n")

	// 格式化输出响应
	if resp.StatusCode == http.StatusOK {
		var responseData map[string]any
		if err := json.Unmarshal(respBody, &responseData); err != nil {
			fmt.Printf("Response (raw): %s\n", string(respBody))
		} else {
			prettyJSON, err := json.MarshalIndent(responseData, "", "  ")
			if err != nil {
				fmt.Printf("Response: %+v\n", responseData)
			} else {
				fmt.Printf("Response:\n%s\n", string(prettyJSON))
			}
		}
	} else {
		fmt.Printf("Error response: %s\n", string(respBody))
	}

}
