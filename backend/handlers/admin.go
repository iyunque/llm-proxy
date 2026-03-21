package handlers

import (
	"ai-api-platform/backend/models"
	"ai-api-platform/backend/services"
	"ai-api-platform/backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// --- Auth ---

func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := models.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(utils.GlobalConfig.Server.JwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// --- Providers ---

func GetProviders(c *gin.Context) {
	var providers []models.AIProvider
	models.DB.Find(&providers)
	c.JSON(http.StatusOK, providers)
}

func CreateProvider(c *gin.Context) {
	var provider models.AIProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	models.DB.Create(&provider)
	c.JSON(http.StatusOK, provider)
}

func UpdateProvider(c *gin.Context) {
	id := c.Param("id")
	var provider models.AIProvider
	if err := models.DB.First(&provider, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := models.DB.Save(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save provider"})
		return
	}

	// Provider 更新后需要刷新已缓存的 endpoints 以保证立即生效。
	if err := services.RefreshEndpointCache(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh endpoint cache"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

func DeleteProvider(c *gin.Context) {
	id := c.Param("id")
	models.DB.Delete(&models.AIProvider{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// --- Endpoints ---

func GetEndpoints(c *gin.Context) {
	var endpoints []models.APIEndpoint
	models.DB.Preload("Provider").Find(&endpoints)
	c.JSON(http.StatusOK, endpoints)
}

func CreateEndpoint(c *gin.Context) {
	var endpoint models.APIEndpoint
	if err := c.ShouldBindJSON(&endpoint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := models.DB.Create(&endpoint).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create endpoint"})
		return
	}
	// 更新缓存
	services.UpdateEndpointCache(&endpoint)
	c.JSON(http.StatusOK, endpoint)
}

func UpdateEndpoint(c *gin.Context) {
	id := c.Param("id")

	// 先获取原始记录
	var oldEndpoint models.APIEndpoint
	if err := models.DB.Preload("Provider").First(&oldEndpoint, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	oldPath := oldEndpoint.Path
	//oldProviderID := oldEndpoint.ProviderID

	// 接收更新数据
	var input struct {
		Path           string `json:"Path"`
		ApiKey         string `json:"ApiKey"`
		ProviderID     uint   `json:"ProviderID"`
		SystemPrompt   string `json:"SystemPrompt"`
		StreamOutput   bool   `json:"StreamOutput"`
		EnableThinking bool   `json:"EnableThinking"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 直接更新特定字段，确保 ProviderID 被正确更新
	updates := models.APIEndpoint{
		Path:           input.Path,
		ApiKey:         input.ApiKey,
		ProviderID:     input.ProviderID,
		SystemPrompt:   input.SystemPrompt,
		StreamOutput:   input.StreamOutput,
		EnableThinking: input.EnableThinking,
	}

	if err := models.DB.Model(&models.APIEndpoint{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update endpoint: " + err.Error()})
		return
	}

	// 重新加载完整数据
	var updatedEndpoint models.APIEndpoint
	models.DB.Preload("Provider").First(&updatedEndpoint, id)

	// 缓存处理
	if oldPath != input.Path {
		services.DeleteEndpointCache(oldPath)
	}
	services.UpdateEndpointCache(&updatedEndpoint)

	c.JSON(http.StatusOK, updatedEndpoint)
}

func DeleteEndpoint(c *gin.Context) {
	id := c.Param("id")

	// 先获取路径信息
	var endpoint models.APIEndpoint
	if err := models.DB.First(&endpoint, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Endpoint not found"})
		return
	}

	// 删除数据库记录
	if err := models.DB.Delete(&models.APIEndpoint{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete endpoint"})
		return
	}

	// 删除缓存
	services.DeleteEndpointCache(endpoint.Path)

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// --- Stats ---

func GetStats(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	var stats []models.APIStats
	var count int64

	models.DB.Where("date = ?", date).Find(&stats)
	models.DB.Model(&models.APIStats{}).Where("date = ?", date).Count(&count)

	// 返回统计信息
	result := gin.H{
		"total_count": count,
		"data":        stats,
		"date_filter": date,
	}

	c.JSON(http.StatusOK, result)
}

// --- User Management ---

// GetUserInfo 获取当前用户信息
func GetUserInfo(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userInfo := user.(models.User)
	c.JSON(http.StatusOK, gin.H{
		"id":       userInfo.ID,
		"username": userInfo.Username,
		"created":  userInfo.CreatedAt,
	})
}

// UpdatePassword 修改用户密码
func UpdatePassword(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userInfo := user.(models.User)

	var input struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(userInfo.Password), []byte(input.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "旧密码不正确"})
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 更新数据库
	if err := models.DB.Model(&userInfo).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

// UpdateUserInfo 更新用户信息
func UpdateUserInfo(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userInfo := user.(models.User)

	var input struct {
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户名是否已存在
	var existingUser models.User
	if err := models.DB.Where("username = ? AND id != ?", input.Username, userInfo.ID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
		return
	}

	// 更新数据库
	if err := models.DB.Model(&userInfo).Update("username", input.Username).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户信息更新成功"})
}
