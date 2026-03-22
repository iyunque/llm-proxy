package handlers

import (
	"ai-api-platform/backend/models"
	"ai-api-platform/backend/services"
	"context"
	"net/http"
	"strings"

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

// ModelAttempt 表示一次模型调用尝试
type ModelAttempt struct {
	Provider   *models.AIProvider
	ModelName  string
	AttemptNum int
}

// callModelWithProvider 使用指定的供应商和模型进行API调用
func callModelWithProvider(ctx context.Context, attempt ModelAttempt, endpoint *models.APIEndpoint, req *ProxyRequest) (*openai.ChatCompletion, error) {
	// 创建 OpenAI 客户端
	client := openai.NewClient(
		option.WithAPIKey(attempt.Provider.APIKey),
		option.WithBaseURL(attempt.Provider.APIAddress),
	)

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(endpoint.SystemPrompt),
			openai.UserMessage(req.Content),
		},
		Model:       attempt.ModelName,
		Temperature: openai.Float(endpoint.Temperature),
	}

	// 思考模式开关（强制发送 true/false）
	extraFields := map[string]interface{}{
		"enable_thinking": endpoint.EnableThinking,
	}
	extraFields["reasoning_split"] = false

	params.SetExtraFields(extraFields)

	// 调用API
	completion, err := client.Chat.Completions.New(ctx, params)
	return completion, err
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

	// 准备模型尝试列表：主模型 + 备用模型1 + 备用模型2
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

	if len(attempts) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No model configured for provider"})
		return
	}

	// 按顺序尝试每个模型
	var lastError error
	var completion *openai.ChatCompletion

	for _, attempt := range attempts {
		completion, lastError = callModelWithProvider(c.Request.Context(), attempt, endpoint, &req)
		if lastError == nil && completion != nil && len(completion.Choices) > 0 {
			// 成功调用，跳出循环
			break
		}
		// 记录失败的尝试，继续下一个
	}

	if lastError != nil || completion == nil || len(completion.Choices) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "All model attempts failed",
			"details": lastError.Error(),
		})
		return
	}

	// 根据流式输出配置决定响应方式
	if endpoint.StreamOutput {
		// 流式输出 - 这里需要重新实现，因为我们现在有completion对象
		// 暂时保持原有逻辑，后续可以优化
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming with fallback not yet implemented"})
		return
	} else {
		// 非流式输出
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
