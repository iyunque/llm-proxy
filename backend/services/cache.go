package services

import (
	"ai-api-platform/backend/models"
	"fmt"
	"sync"
)

var (
	endpointCache    map[string]*models.APIEndpoint
	endpointCacheMux sync.RWMutex
)

// InitEndpointCache 初始化 API 路径缓存
func InitEndpointCache() error {
	endpointCacheMux.Lock()
	defer endpointCacheMux.Unlock()

	endpointCache = make(map[string]*models.APIEndpoint)

	var endpoints []models.APIEndpoint
	if err := models.DB.Preload("Provider").Find(&endpoints).Error; err != nil {
		return err
	}

	for i := range endpoints {
		endpointCache[endpoints[i].Path] = &endpoints[i]
		fmt.Printf("[CACHE DEBUG] Loaded endpoint: %s (ID: %d, StreamOutput: %t)\n", endpoints[i].Path, endpoints[i].ID, endpoints[i].StreamOutput)
	}

	fmt.Printf("[CACHE DEBUG] Total endpoints loaded: %d\n", len(endpoints))
	return nil
}

// GetEndpointByPath 从缓存获取 API 路径配置
func GetEndpointByPath(path string) (*models.APIEndpoint, bool) {
	endpointCacheMux.RLock()
	defer endpointCacheMux.RUnlock()

	endpoint, exists := endpointCache[path]
	if !exists {
		fmt.Printf("[CACHE DEBUG] Path not found in cache: %s\n", path)
	} else {
		fmt.Printf("[CACHE DEBUG] Found cached endpoint: %s (ID: %d, StreamOutput: %t)\n", path, endpoint.ID, endpoint.StreamOutput)
	}
	return endpoint, exists
}

// UpdateEndpointCache 更新缓存中的 API 路径
func UpdateEndpointCache(endpoint *models.APIEndpoint) {
	endpointCacheMux.Lock()
	defer endpointCacheMux.Unlock()

	// 重新加载 Provider 关联数据
	models.DB.Preload("Provider").First(endpoint, endpoint.ID)
	endpointCache[endpoint.Path] = endpoint
	fmt.Printf("[CACHE DEBUG] Updated cached endpoint: %s (ID: %d, StreamOutput: %t)\n", endpoint.Path, endpoint.ID, endpoint.StreamOutput)
}

// DeleteEndpointCache 从缓存中删除 API 路径
func DeleteEndpointCache(path string) {
	endpointCacheMux.Lock()
	defer endpointCacheMux.Unlock()

	delete(endpointCache, path)
	fmt.Printf("[CACHE DEBUG] Deleted cached endpoint: %s\n", path)
}

// RefreshEndpointCache 刷新整个缓存
func RefreshEndpointCache() error {
	return InitEndpointCache()
}

// GetAllCachedEndpoints 获取所有缓存的路径（用于调试）
func GetAllCachedEndpoints() map[string]*models.APIEndpoint {
	endpointCacheMux.RLock()
	defer endpointCacheMux.RUnlock()

	// 返回副本，避免外部修改
	result := make(map[string]*models.APIEndpoint, len(endpointCache))
	for k, v := range endpointCache {
		result[k] = v
	}
	return result
}
