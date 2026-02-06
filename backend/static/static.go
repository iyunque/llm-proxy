package static

import (
	"ai-api-platform/backend/handlers"
	"ai-api-platform/backend/services"
	"embed"
	"io/fs"
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distEmbed embed.FS

func Serve(r *gin.Engine, servePath string) {
	dist, _ := fs.Sub(distEmbed, "dist")

	// 确保 servePath 是以 / 开头的合法 URL 前缀
	if servePath == "" || servePath[0] != '/' {
		servePath = "/"
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 1. 检查是否是 API 代理路径 (使用缓存)
		if _, exists := services.GetEndpointByPath(path); exists {
			handlers.ProxyHandler(c)
			return
		}

		// 2. 如果是 admin 路径且没匹配到后端路由，返回 404
		if strings.HasPrefix(path, "/admin") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
			return
		}

		// 3. 静态资源服务
		filePath := path
		if servePath != "/" && strings.HasPrefix(path, servePath) {
			filePath = path[len(servePath):]
		}

		if filePath == "" || filePath == "/" {
			filePath = "index.html"
		} else if filePath[0] == '/' {
			filePath = filePath[1:]
		}

		f, err := dist.Open(filePath)
		if err == nil {
			f.Close()
			// 如果是 index.html，直接返回内容，避免重定向循环
			if filePath == "index.html" {
				data, _ := fs.ReadFile(dist, "index.html")
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
				return
			}
			c.FileFromFS(filePath, http.FS(dist))
			return
		}

		// 如果找不到文件，则返回 index.html (支持 SPA 路由)
		data, err := fs.ReadFile(dist, "index.html")
		if err == nil {
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		}
	})
}
