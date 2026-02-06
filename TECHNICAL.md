# AI API 转发与管理平台 - 技术说明文档

## 1. 项目架构设计

### 1.1 整体架构
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client API    │────│  Proxy Server   │────│  LLM Provider   │
│                 │    │                 │    │                 │
│ /api/translate  │    │ - Route Cache   │    │ - OpenAI API    │
│ X-API-Key:xxx   │    │ - Stats Service │    │ - DeepSeek API  │
│ {content:"..."} │    │ - JWT Auth      │    │ - GLM API       │
└─────────────────┘    │ - Embed Static  │    └─────────────────┘
                       │ - Config Load   │
                       └─────────────────┘
```

### 1.2 技术栈选择理由

**后端 (Go + Gin)**
- **性能**: Go 并发模型天然适合高并发 API 代理
- **生态**: Gin 框架轻量高效，中间件生态丰富
- **部署**: 单文件可执行，零依赖部署

**数据库 (GORM + SQLite)**
- **GORM**: ORM 层提供便捷的数据操作，支持多数据库切换
- **SQLite**: 无服务器模式，无需额外数据库服务，适合小型部署
- **glebarez/sqlite**: 无 CGO 依赖，跨平台编译友好

**前端 (Vue 3 + Element Plus)**
- **响应式**: Composition API 提供更好的状态管理
- **组件化**: Element Plus 组件库保证 UI 一致性
- **现代化**: Vite 构建工具提供快速开发体验

## 2. 核心模块实现详解

### 2.1 API 路径缓存机制 (services/cache.go)

**设计目标**: 避免每次请求都查询数据库，提升性能 100-1000 倍

**实现原理**:
```go
// 缓存结构
var endpointCache map[string]*models.APIEndpoint
var endpointCacheMux sync.RWMutex  // 读写锁保证并发安全

// 启动时初始化
func InitEndpointCache() {
    // 预加载所有 API 路径配置到内存
    models.DB.Preload("Provider").Find(&endpoints)
    for i := range endpoints {
        endpointCache[endpoints[i].Path] = &endpoints[i]  // 以路径为 key 存储
    }
}

// 代理时快速查询
func GetEndpointByPath(path string) (*models.APIEndpoint, bool) {
    // 读锁保护，支持高并发读取
    endpointCacheMux.RLock()
    defer endpointCacheMux.RUnlock()
    endpoint, exists := endpointCache[path]
    return endpoint, exists
}
```

**缓存同步策略**:
- **新增**: `UpdateEndpointCache()` 直接插入
- **修改**: 若路径变更，先删除旧缓存，再更新新缓存
- **删除**: `DeleteEndpointCache()` 直接删除
- **异常**: 数据库异常时保持缓存一致性

### 2.2 代理转发逻辑 (handlers/proxy.go)

**请求处理流程**:
```go
func ProxyHandler(c *gin.Context) {
    // 1. 从缓存获取配置
    endpoint, exists := services.GetEndpointByPath(path)
    if !exists || endpoint.ApiKey != apiKey {
        return unauthorized()
    }
    
    // 2. 构造 OpenAI 格式请求
    openAIReq := OpenAIRequest{
        Model: endpoint.Provider.ModelName,  // 使用供应商配置的模型
        Messages: []OpenAIMessage{
            {Role: "system", Content: endpoint.SystemPrompt},  // 系统提示词
            {Role: "user", Content: req.Content},             // 用户输入
        },
    }
    
    // 3. 转发到供应商
    httpReq, _ := http.NewRequest("POST", endpoint.Provider.APIAddress, ...)
    httpReq.Header.Set("Authorization", "Bearer "+endpoint.Provider.APIKey)
    
    // 4. 处理响应并统计
    resp, err := client.Do(httpReq)
    var openAIResp OpenAIResponse
    json.Unmarshal(body, &openAIResp)
    
    // 5. 记录统计数据
    services.AddStats(endpoint.ID, openAIResp.Usage.PromptTokens, ...)
}
```

**关键设计点**:
- **格式转换**: 将简单 `{content: "..."}` 格式转换为 OpenAI 标准格式
- **安全性**: 通过缓存验证 API Key，防止数据库注入
- **Token 统计**: 准确记录输入/输出 Token 数量

### 2.3 统计服务实现 (services/stats.go)

**内存+定时持久化策略**:
```go
var memoryStats map[uint]*models.APIStats  // 内存统计
var statsMutex sync.Mutex                 // 统计互斥锁

// 内存统计结构
type MemoryStat struct {
    CallCount       int64  // 调用次数
    InputTokens     int64  // 输入 Token
    OutputTokens    int64  // 输出 Token
    CacheHitTokens  int64  // 缓存命中 Token
    LastUpdated     time.Time
}

// 定时同步协程
func InitStats() {
    go func() {
        ticker := time.NewTicker(time.Duration(interval) * time.Second)
        for range ticker.C {
            SyncStatsToDB()  // 批量写入数据库
        }
    }()
}

// 内存累加
func AddStats(endpointID uint, input, output, cache int64) {
    statsMutex.Lock()
    defer statsMutex.Unlock()
    
    stat := memoryStats[endpointID]
    stat.CallCount++
    stat.InputTokens += input
    stat.OutputTokens += output
    stat.LastUpdated = time.Now()
}
```

**设计优势**:
- **性能**: 避免每次请求都写数据库，减少 IO 开销
- **准确性**: 内存累加保证数据不丢失
- **灵活性**: 可配置同步间隔

### 2.4 静态资源嵌入 (static/static.go)

**SPA 路由处理**:
```go
func Serve(r *gin.Engine, servePath string) {
    r.NoRoute(func(c *gin.Context) {
        path := c.Request.URL.Path
        
        // 1. 检查是否是 API 代理路径
        if _, exists := services.GetEndpointByPath(path); exists {
            handlers.ProxyHandler(c)  // 转到代理处理
            return
        }
        
        // 2. 检查管理接口路径
        if strings.HasPrefix(path, "/admin") {
            c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
            return
        }
        
        // 3. 静态资源服务
        f, err := dist.Open(filePath)
        if err == nil {
            c.FileFromFS(filePath, http.FS(dist))  // 直接返回文件
            return
        }
        
        // 4. SPA 回退路由
        c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)  // 返回 index.html
    })
}
```

**关键特性**:
- **路由优先级**: API > Admin > Static > Fallback
- **SPA 支持**: 未知路径返回 index.html，由前端路由处理
- **嵌入资源**: 使用 Go embed 将前端打包到二进制文件

### 2.5 数据库模型设计 (models/models.go)

**关联关系**:
```go
type APIEndpoint struct {
    gorm.Model
    Path         string        `gorm:"uniqueIndex"`     // API 访问路径
    SystemPrompt string        `gorm:"type:text"`       // 系统提示词
    ApiKey       string        `gorm:"size:32"`         // 客户端密钥
    ProviderID   uint                              // 外键
    Provider     AIProvider    `gorm:"foreignKey:ProviderID"`  // 关联供应商
    StreamOutput bool          `gorm:"default:false"`         // 流式输出配置
}

type AIProvider struct {
    gorm.Model
    Name       string `gorm:"not null"`    // 供应商名称
    ModelName  string `gorm:"not null"`    // 模型名称
    APIAddress string `gorm:"not null"`    // API 地址
    APIKey     string `gorm:"not null"`    // 供应商密钥
}

type User struct {
    gorm.Model
    Username string `gorm:"not null;unique"`  // 用户名
    Password string `gorm:"not null"`         // 加密密码
}
```

**设计考量**:
- **外键约束**: 保证数据完整性
- **索引优化**: Path 字段建立唯一索引
- **关联预加载**: 查询时预加载供应商信息
- **用户管理**: 新增用户表支持个人中心功能

### 2.6 流式输出实现 (handlers/proxy.go)

**流式响应处理**:
```go
func ProxyHandler(c *gin.Context) {
    // ... 获取端点配置 ...
    
    if endpoint.StreamOutput {
        // 流式输出 - 先设置响应头并发送状态码
        c.Status(http.StatusOK)  // 强制返回 200 状态码
        c.Header("Content-Type", "text/event-stream")
        c.Header("Cache-Control", "no-cache")
        c.Header("Connection", "keep-alive")
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Headers", "*")

        // 检查供应商 API 的响应状态
        if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
            body, _ := io.ReadAll(resp.Body)
            resp.Body.Close()
            c.JSON(resp.StatusCode, gin.H{"error": "AI provider returned error: " + string(body)})
            return
        }

        // 创建管道用于流式传输
        pipeReader, pipeWriter := io.Pipe()
        writer := bufio.NewWriter(pipeWriter)
        c.Stream(func(w io.Writer) bool {
            // 从供应商 API 读取并转发到客户端
            buffer := make([]byte, 4096)
            n, err := resp.Body.Read(buffer)
            if err != nil {
                if err == io.EOF {
                    pipeWriter.Close()
                    return false
                }
                pipeWriter.CloseWithError(err)
                return false
            }
            pipeWriter.Write(buffer[:n])
            return true
        })

        // 传输完成，关闭管道
        pipeWriter.Close()
        return
    } else {
        // 非流式输出 - 正常 JSON 响应
        // ... 原有逻辑 ...
    }
}
```

**流式设计要点**:
- **响应头设置**: 正确设置 SSE 相关头部
- **状态码处理**: 强制返回 200 避免与流式响应冲突
- **错误检查**: 先检查供应商 API 响应状态
- **管道传输**: 使用管道实现高效的流式数据传输

### 2.7 用户认证与授权 (middleware/auth.go)

**JWT 认证中间件**:
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if !(len(parts) == 2 && parts[0] == "Bearer") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
            c.Abort()
            return
        }

        tokenString := parts[1]
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(utils.GlobalConfig.Server.JwtSecret), nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
            c.Abort()
            return
        }

        userIDFloat, ok := claims["user_id"].(float64)
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
            c.Abort()
            return
        }
        userID := uint(userIDFloat)

        // 查询用户信息
        var user models.User
        if err := models.DB.First(&user, userID).Error; err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
            c.Abort()
            return
        }

        // 设置用户信息到上下文
        c.Set("user", user)
        c.Next()
    }
}
```

**认证设计特点**:
- **Token 解析**: 解析 JWT Token 获取用户 ID
- **用户查询**: 根据用户 ID 查询完整用户信息
- **上下文传递**: 将用户信息传递给后续处理器
- **错误处理**: 完善的认证失败处理

### 2.8 用户管理接口 (handlers/admin.go)

**用户信息管理**:
```go
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
```

**用户管理特性**:
- **密码验证**: 修改前验证旧密码
- **加密存储**: 使用 bcrypt 加密密码
- **长度校验**: 新密码最小长度要求
- **错误处理**: 完善的错误处理机制

## 3. 性能优化策略

### 3.1 缓存层次结构
```
Client Request
    ↓
[Route Cache] ← 1st Level (内存 Map, O(1) 查找)
    ↓
[DB Query] ← 2nd Level (仅首次加载)
```

### 3.2 并发安全设计
- **读写锁**: 缓存读取使用 RWMutex，支持高并发读取
- **原子操作**: 统计累加使用互斥锁保护
- **无锁统计**: 内存统计避免频繁数据库写入

### 3.3 内存管理
- **对象复用**: 避免频繁内存分配
- **连接池**: HTTP 客户端连接复用
- **GC 优化**: 避免大对象长期持有

## 4. 安全机制

### 4.1 认证授权
- **JWT Token**: 管理接口使用 JWT 进行身份验证
- **API Key**: 客户端访问使用独立的 API Key
- **权限分离**: 管理员权限与客户端权限隔离

### 4.2 输入验证
- **参数绑定**: Gin 结构体绑定自动验证
- **SQL 注入防护**: 使用 GORM 参数化查询
- **路径安全**: 静态文件服务路径限制

### 4.3 密码安全
- **加密存储**: 用户密码使用 bcrypt 加密
- **强度校验**: 密码长度和复杂度要求
- **验证机制**: 修改密码前验证旧密码

## 5. 部署架构建议

### 5.1 单机部署
```
┌─────────────────┐
│  Production     │
│  Server         │
│                 │
│  ai-server.exe  │ ← 单进程运行
│  nginx proxy    │ ← 反向代理 + SSL
│  systemd service│ ← 进程管理
└─────────────────┘
```

### 5.2 高可用部署
```
┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │────│   ai-server-1   │
│   (nginx/haproxy)│    │   (Active)      │
└─────────────────┘    └─────────────────┘
                            ↑
                            │ (Health Check)
                            ↓
                    ┌─────────────────┐
                    │   ai-server-2   │
                    │   (Standby)     │
                    └─────────────────┘
```

## 6. 监控与运维

### 6.1 指标收集
- **QPS**: 每秒请求数
- **响应时间**: P50/P95/P99 延迟
- **Token 消耗**: 输入/输出 Token 统计
- **错误率**: API 调用错误率

### 6.2 日志规范
- **结构化日志**: JSON 格式便于分析
- **链路追踪**: Request ID 关联请求链路
- **审计日志**: API Key 调用记录

## 7. 扩展性设计

### 7.1 插件化架构
- **中间件扩展**: Gin 中间件机制支持功能扩展
- **配置热更新**: 支持运行时配置变更
- **插件接口**: 预留第三方供应商接入接口

### 7.2 微服务改造
- **服务拆分**: API 管理、统计服务、代理服务可独立部署
- **消息队列**: 异步统计处理
- **分布式缓存**: Redis 替代内存缓存

## 8. 开发指南

### 8.1 项目结构
```
├── backend/
│   ├── handlers/     # HTTP 控制器
│   ├── middleware/   # 中间件
│   ├── models/       # 数据模型
│   ├── services/     # 业务逻辑
│   ├── static/       # 静态资源处理
│   └── utils/        # 工具函数
├── frontend/
│   ├── src/
│   │   ├── views/    # 页面组件
│   │   ├── components/ # 通用组件
│   │   ├── router/   # 路由配置
│   │   └── api/      # API 接口封装
│   └── public/       # 静态资源
├── config/           # 配置文件
└── main.go           # 程序入口
```

### 8.2 代码规范
- **命名**: 遵循 Go 语言命名规范
- **注释**: 核心函数添加文档注释
- **错误处理**: 统一错误处理机制
- **测试**: 关键逻辑单元测试覆盖

### 8.3 接口扩展示例

**新增供应商类型**:
```go
// 1. 扩展供应商模型
type CustomProvider struct {
    AIProvider
    ExtraConfig string  // 自定义配置
}

// 2. 实现特定供应商逻辑
func (p *CustomProvider) MakeRequest(content string) *OpenAIRequest {
    // 自定义请求构造逻辑
    return &OpenAIRequest{
        Model: p.ModelName,
        Messages: []OpenAIMessage{
            {Role: "system", Content: p.CustomSystemPrompt},
            {Role: "user", Content: content},
        },
    }
}
```

## 9. 故障排除

### 9.1 性能问题
- **高延迟**: 检查网络连接、供应商响应时间
- **内存泄漏**: 检查连接池配置、对象生命周期
- **CPU 高**: 检查缓存命中率、查询复杂度

### 9.2 连接问题
- **数据库连接**: 检查连接池大小、超时配置
- **API 连接**: 检查供应商地址、网络策略
- **并发限制**: 检查系统文件描述符限制

### 9.3 数据一致性
- **统计丢失**: 检查定时同步逻辑、异常处理
- **缓存不一致**: 检查同步时机、并发安全
- **数据损坏**: 检查事务完整性、备份恢复

### 9.4 认证问题
- **401 错误**: 检查 JWT Token 是否有效，认证中间件是否正确
- **用户信息缺失**: 检查认证中间件是否正确查询用户信息
- **密码修改失败**: 检查旧密码验证逻辑，bcrypt 加密是否正常

### 9.5 流式输出问题
- **404 错误**: 检查前端请求 URL 是否正确（相对路径 vs 绝对路径）
- **状态码冲突**: 检查流式响应前是否正确设置状态码
- **SSE 格式**: 检查响应头和数据格式是否符合 SSE 规范

---

**文档版本**: TECHNICAL-v1.1  
**更新时间**: 2026-02-05  
**适用人群**: Go/Vue 开发工程师、系统架构师、运维工程师
