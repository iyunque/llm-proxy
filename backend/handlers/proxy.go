package handlers

import (
	"ai-api-platform/backend/services"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ProxyRequest struct {
	Content string `json:"content" binding:"required"`
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
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
		fmt.Printf("[PROXY DEBUG] Endpoint not found in cache: %s\n", path)
		c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
		return
	}
	if endpoint.ApiKey != apiKey {
		fmt.Printf("[PROXY DEBUG] API Key mismatch for path %s. Expected: %s, Got: %s\n", path, endpoint.ApiKey, apiKey)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Path or API Key"})
		return
	}
	fmt.Printf("[PROXY DEBUG] Successfully authenticated endpoint: %s (ID: %d)\n", path, endpoint.ID)

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

		// 读取并处理供应商的 SSE 响应
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println("data:", line) // Debugging
			// 将供应商的 SSE 数据转发给客户端
			if strings.TrimSpace(line) != "" {
				c.Writer.Write([]byte(line + "\n"))
				flusher.Flush()
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
