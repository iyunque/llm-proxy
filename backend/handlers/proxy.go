package handlers

import (
	"ai-api-platform/backend/services"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type ProxyRequest struct {
	Content string `json:"content" binding:"required"`
}

type OpenAIRequest struct {
	Model     string                 `json:"model"`
	Messages  []OpenAIMessage        `json:"messages"`
	Stream    bool                   `json:"stream"`
	ExtraBody map[string]interface{} `json:"extra_body,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
}

func ProxyHandler(c *gin.Context) {
	path := c.Request.URL.Path
	apiKey := c.GetHeader("X-API-Key")

	// 从缓存获取 API 路径配置
	endpoint, exists := services.GetEndpointByPath(path)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
		return
	}
	if endpoint.ApiKey != apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Path or API Key"})
		return
	}

	var req ProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body, 'content' is required"})
		return
	}

	// 创建 OpenAI 客户端
	client := openai.NewClient(
		option.WithAPIKey(endpoint.Provider.APIKey),
		option.WithBaseURL(endpoint.Provider.APIAddress),
	)

	// 构造聊天完成请求参数
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(endpoint.SystemPrompt),
			openai.UserMessage(req.Content),
		},
		Model: endpoint.Provider.ModelName,
	}

	// 思考模式开关（强制发送 true/false）
	extraFields := map[string]interface{}{
		"enable_thinking": endpoint.EnableThinking,
	}
	if endpoint.EnableThinking {
		extraFields["reasoning_split"] = true
	} else {
		extraFields["reasoning_split"] = false
	}
	params.SetExtraFields(extraFields)

	// 根据流式输出配置决定响应方式
	if endpoint.StreamOutput {
		// 流式输出
		stream := client.Chat.Completions.NewStreaming(c.Request.Context(), params)

		// 设置响应头
		c.Status(http.StatusOK)
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "*")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
			return
		}

		// 监听客户端连接关闭
		clientClosed := c.Request.Context().Done()

		var usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		}

		// 处理流式响应
		for stream.Next() {
			select {
			case <-clientClosed:
				stream.Close()
				return
			default:
				evt := stream.Current()
				if len(evt.Choices) > 0 {
					choice := evt.Choices[0]
					if choice.Delta.Content != "" {
						// 发送内容数据
						data := map[string]interface{}{
							"id":      evt.ID,
							"object":  "chat.completion.chunk",
							"created": evt.Created,
							"model":   evt.Model,
							"choices": []map[string]interface{}{
								{
									"index": 0,
									"delta": map[string]interface{}{
										"content": choice.Delta.Content,
									},
									"finish_reason": choice.FinishReason,
								},
							},
						}
						jsonData, _ := json.Marshal(data)
						c.Writer.Write([]byte("data: " + string(jsonData) + "\n\n"))
						flusher.Flush()
					}
				}

				// 记录使用情况
				if evt.Usage.PromptTokens > 0 || evt.Usage.CompletionTokens > 0 {
					usage.PromptTokens = evt.Usage.PromptTokens
					usage.CompletionTokens = evt.Usage.CompletionTokens
					usage.TotalTokens = evt.Usage.PromptTokens + evt.Usage.CompletionTokens
				}
			}
		}

		if err := stream.Err(); err != nil {
			c.Writer.Write([]byte("data: {\"error\": \"" + err.Error() + "\"}\n\n"))
		}

		// 发送结束标记
		c.Writer.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()

		// 记录统计数据
		if usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
			services.AddStats(endpoint.ID, usage.PromptTokens, usage.CompletionTokens, 0)
		}
	} else {
		// 非流式输出
		completion, err := client.Chat.Completions.New(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call AI provider: " + err.Error()})
			return
		}

		if len(completion.Choices) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No response from AI provider"})
			return
		}

		// 构造 OpenAI 格式响应
		response := OpenAIResponse{
			ID: completion.ID,
			Choices: []struct {
				Message OpenAIMessage `json:"message"`
			}{
				{
					Message: OpenAIMessage{
						Role:    string(completion.Choices[0].Message.Role),
						Content: completion.Choices[0].Message.Content,
					},
				},
			},
		}

		if completion.Usage.PromptTokens > 0 || completion.Usage.CompletionTokens > 0 {
			response.Usage.PromptTokens = completion.Usage.PromptTokens
			response.Usage.CompletionTokens = completion.Usage.CompletionTokens
			response.Usage.TotalTokens = completion.Usage.PromptTokens + completion.Usage.CompletionTokens
		}

		// 记录统计数据
		if response.Usage.PromptTokens > 0 || response.Usage.CompletionTokens > 0 {
			services.AddStats(endpoint.ID, response.Usage.PromptTokens, response.Usage.CompletionTokens, 0)
		}

		c.JSON(http.StatusOK, response)
	}
}
