package handlers

import (
	"ai-api-platform/backend/models"
	"ai-api-platform/backend/services"
	"ai-api-platform/backend/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// 默认 HTTP 客户端，超时时间由配置决定（非流式）
var defaultHTTPClient = &http.Client{
	Timeout: time.Duration(utils.GlobalConfig.Proxy.Timeout) * time.Second,
}

// 流式输出使用的 HTTP 客户端，不设置超时（由 context 控制客户端断开）
var streamingHTTPClient = &http.Client{
	Timeout: 0,
}

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

// ModelAttempt 表示一次模型调用尝试
type ModelAttempt struct {
	Provider   *models.AIProvider
	ModelName  string
	AttemptNum int
}

// buildChatCompletionParams 构建聊天补全参数
func buildChatCompletionParams(endpoint *models.APIEndpoint, req ProxyRequest, modelName string) openai.ChatCompletionNewParams {
	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(endpoint.SystemPrompt),
			openai.UserMessage(req.Content),
		},
		Model:       modelName,
		Temperature: openai.Float(endpoint.Temperature),
	}

	extraFields := map[string]interface{}{
		"enable_thinking": endpoint.EnableThinking,
		"reasoning_split": false,
	}
	params.SetExtraFields(extraFields)

	return params
}

// buildAttemptsList 构建模型尝试列表
func buildAttemptsList(endpoint *models.APIEndpoint) ([]ModelAttempt, error) {
	var attempts []ModelAttempt

	// 主模型
	mainModelName := strings.TrimSpace(endpoint.SelectedModel)
	if mainModelName == "" {
		mainModelName = endpoint.Provider.ModelName
	}
	if strings.Contains(mainModelName, ",") {
		for _, m := range strings.Split(mainModelName, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				mainModelName = m
				break
			}
		}
	}
	if mainModelName != "" {
		attempts = append(attempts, ModelAttempt{
			Provider:   &endpoint.Provider,
			ModelName:  mainModelName,
			AttemptNum: 1,
		})
	}

	// 备用模型1
	if endpoint.FallbackProviderID1 > 0 && endpoint.FallbackModel1 != "" {
		var fallbackProvider1 models.AIProvider
		if err := models.DB.First(&fallbackProvider1, endpoint.FallbackProviderID1).Error; err == nil {
			attempts = append(attempts, ModelAttempt{
				Provider:   &fallbackProvider1,
				ModelName:  strings.TrimSpace(endpoint.FallbackModel1),
				AttemptNum: 2,
			})
		}
	}

	// 备用模型2
	if endpoint.FallbackProviderID2 > 0 && endpoint.FallbackModel2 != "" {
		var fallbackProvider2 models.AIProvider
		if err := models.DB.First(&fallbackProvider2, endpoint.FallbackProviderID2).Error; err == nil {
			attempts = append(attempts, ModelAttempt{
				Provider:   &fallbackProvider2,
				ModelName:  strings.TrimSpace(endpoint.FallbackModel2),
				AttemptNum: 3,
			})
		}
	}

	return attempts, nil
}

// handleStreamingOutput 处理流式输出
func handleStreamingOutput(c *gin.Context, attempts []ModelAttempt, endpoint *models.APIEndpoint, req ProxyRequest) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	var lastStreamErr error
	streamStarted := false
	c.Status(http.StatusForbidden)

	for _, attempt := range attempts {
		// 如果客户端已断开，直接返回
		if c.Request.Context().Err() != nil {
			return
		}

		client := openai.NewClient(
			option.WithAPIKey(attempt.Provider.APIKey),
			option.WithBaseURL(attempt.Provider.APIAddress),
			option.WithHTTPClient(streamingHTTPClient),
		)

		params := buildChatCompletionParams(endpoint, req, attempt.ModelName)
		stream := client.Chat.Completions.NewStreaming(c.Request.Context(), params)

		for stream.Next() {
			// 检查客户端是否已断开
			if c.Request.Context().Err() != nil {
				stream.Close()
				return
			}

			chunk := stream.Current()
			if len(chunk.Choices) > 0 {
				if !streamStarted {
					c.Status(http.StatusOK)
					streamStarted = true
				}
				content := chunk.Choices[0].Delta.Content
				if content != "" {
					contentJSON, _ := json.Marshal(content)
					sseData := fmt.Sprintf(`{"choices":[{"delta":{"content":%s},"index":0}]}`, string(contentJSON))
					c.Writer.Write([]byte("data: " + sseData + "\n\n"))
					c.Writer.Flush()
				}
			}
		}

		if err := stream.Err(); err != nil {
			// 如果是客户端断开导致的错误，不记录失败也不切换模型
			if c.Request.Context().Err() != nil {
				stream.Close()
				return
			}
			lastStreamErr = err
			services.AddFailedStats(endpoint.ID, attempt.Provider.Name, attempt.ModelName)
			stream.Close()
			continue
		}

		stream.Close()
		if !streamStarted {
			c.Status(http.StatusOK)
			streamStarted = true
		}
		c.Writer.Write([]byte("data: [DONE]\n\n"))
		c.Writer.Flush()
		return
	}

	// 所有模型都失败了
	c.Status(http.StatusInternalServerError)
	c.Writer.Write([]byte("data: " + fmt.Sprintf(`{"error":"All model attempts failed: %s"}`, strings.ReplaceAll(lastStreamErr.Error(), `"`, `\"`)) + "\n\n"))
	c.Writer.Write([]byte("data: [DONE]\n\n"))
	c.Writer.Flush()
}

// handleNonStreamingOutput 处理非流式输出
func handleNonStreamingOutput(c *gin.Context, attempts []ModelAttempt, endpoint *models.APIEndpoint, req *ProxyRequest) {
	var lastError error
	var completion *openai.ChatCompletion

	for _, attempt := range attempts {
		// 如果客户端已断开，直接返回
		if c.Request.Context().Err() != nil {
			return
		}

		client := openai.NewClient(
			option.WithAPIKey(attempt.Provider.APIKey),
			option.WithBaseURL(attempt.Provider.APIAddress),
			option.WithHTTPClient(defaultHTTPClient),
		)

		params := buildChatCompletionParams(endpoint, *req, attempt.ModelName)
		completion, lastError = client.Chat.Completions.New(c.Request.Context(), params)

		// 如果客户端已断开，不记录失败也不继续尝试
		if c.Request.Context().Err() != nil {
			return
		}

		if lastError == nil && completion != nil && len(completion.Choices) > 0 {
			break
		}
		// 只有明确失败才记录失败统计
		if lastError != nil {
			services.AddFailedStats(endpoint.ID, attempt.Provider.Name, attempt.ModelName)
		}
	}

	if lastError != nil || completion == nil || len(completion.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "All model attempts failed",
			"details": lastError.Error(),
		})
		return
	}

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

	if response.Usage.PromptTokens > 0 || response.Usage.CompletionTokens > 0 {
		services.AddStats(endpoint.ID, response.Usage.PromptTokens, response.Usage.CompletionTokens, 0)
	}

	c.JSON(http.StatusOK, response)
}

// ProxyHandler 处理代理请求
func ProxyHandler(c *gin.Context) {
	path := c.Request.URL.Path
	apiKey := c.GetHeader("X-API-Key")

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

	attempts, err := buildAttemptsList(endpoint)
	if err != nil || len(attempts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No model configured for provider"})
		return
	}

	if endpoint.StreamOutput {
		handleStreamingOutput(c, attempts, endpoint, req)
	} else {
		handleNonStreamingOutput(c, attempts, endpoint, &req)
	}
}
