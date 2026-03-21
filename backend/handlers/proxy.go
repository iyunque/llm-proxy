package handlers

import (
	"ai-api-platform/backend/services"
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

	// 构造 OpenAI 格式请求
	openAIReq := OpenAIRequest{
		Model: endpoint.Provider.ModelName, // 使用供应商配置的模型名称
		Messages: []OpenAIMessage{
			{Role: "system", Content: endpoint.SystemPrompt},
			{Role: "user", Content: req.Content},
		},
		Stream: endpoint.StreamOutput, // 根据配置决定是否启用流式输出
	}

	// 如果启用思考模式，设置 extra_body
	if endpoint.EnableThinking {
		openAIReq.ExtraBody = map[string]interface{}{
			"enable_thinking": true,
			"reasoning_split": true,
		}
	}

	jsonData, _ := json.Marshal(openAIReq)

	client := &http.Client{}
	httpReq, err := http.NewRequest("POST", endpoint.Provider.APIAddress, bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+endpoint.Provider.APIKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call AI provider: " + err.Error()})
		return
	}

	// 检查供应商 API 的响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// 读取供应商 API 的错误响应
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		c.JSON(resp.StatusCode, gin.H{"error": "AI provider returned error: " + string(body)})
		return
	}

	defer resp.Body.Close()

	// 根据流式输出配置决定响应方式
	if endpoint.StreamOutput {
		// 流式输出 - 先设置响应头并发送状态码
		c.Status(http.StatusOK) // 强制返回 200 状态码
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "*")

		// 创建一个流式响应
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

		// 读取并处理供应商的 SSE 响应
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-clientClosed:
				// 客户端已断开连接，停止转发并关闭响应体
				resp.Body.Close()
				return
			default:
				line := scanner.Text()
				//fmt.Println("data:", line) // Debugging
				// 检查是否是数据行
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if data != "[DONE]" {
						// 尝试解析JSON以提取usage
						var chunk map[string]interface{}
						if err := json.Unmarshal([]byte(data), &chunk); err == nil {
							if u, ok := chunk["usage"].(map[string]interface{}); ok {
								if pt, ok := u["prompt_tokens"].(float64); ok {
									usage.PromptTokens = int64(pt)
								}
								if ct, ok := u["completion_tokens"].(float64); ok {
									usage.CompletionTokens = int64(ct)
								}
							}
						}
					}
				}
				// 将供应商的 SSE 数据转发给客户端
				if strings.TrimSpace(line) != "" {
					c.Writer.Write([]byte(line + "\n"))
					flusher.Flush()
				}
			}
		}

		if err := scanner.Err(); err != nil {
			c.Writer.Write([]byte("data: {\"error\": \"" + err.Error() + "\"}\n"))
			c.Writer.Write([]byte("data: [DONE]\n"))
			flusher.Flush()
		}
		// 发送结束标记
		c.Writer.Write([]byte("data: [DONE]\n"))
		flusher.Flush()

		// 记录统计数据
		if usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
			services.AddStats(endpoint.ID, usage.PromptTokens, usage.CompletionTokens, 0)
		}
	} else {
		// 非流式输出
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			c.Data(resp.StatusCode, "application/json", body)
			return
		}

		var openAIResp OpenAIResponse
		if err := json.Unmarshal(body, &openAIResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI provider response"})
			return
		}

		// 记录统计数据
		services.AddStats(endpoint.ID, openAIResp.Usage.PromptTokens, openAIResp.Usage.CompletionTokens, 0)

		c.JSON(http.StatusOK, openAIResp)
	}
}
