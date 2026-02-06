# LLM代理网关

一款高性能的 LLM 代理转发与管理平台，支持多种 LLM 供应商，提供统一的 API 接口、流量统计、流式输出等功能。

## 🌟 功能特性

### 核心功能
- **多供应商支持**: 支持 OpenAI、DeepSeek、GLM 等多种 LLM 供应商
- **API 路由管理**: 动态配置 API 路径和供应商映射
- **统一接口**: 将不同供应商的 API 格式统一为标准格式
- **流量统计**: 实时统计 API 调用次数和 Token 消耗

### 高级功能
- **流式输出**: 支持 SSE 流式响应，实现逐字输出效果
- **用户管理**: 支持用户个人中心，包含密码修改等功能
- **系统提示词**: 为每个 API 路径配置独立的系统提示词
- **缓存机制**: 高性能内存缓存，提升 API 响应速度

### 安全特性
- **JWT 认证**: 管理后台使用 JWT 进行身份验证
- **API Key 验证**: 客户端访问使用独立的 API Key
- **权限控制**: 管理员与普通用户权限分离

## 🚀 快速开始

### 环境要求
- Go 1.19+
- Node.js 16+
- npm/yarn

### 本地开发

```bash
# 克隆项目
git clone <repository-url>
cd ai-api-platform

# 启动后端
cd backend
go run main.go

# 启动前端
cd frontend
npm install
npm run dev
```

### 生产部署

```bash
# 构建前端
cd frontend
npm run build

# 构建后端
cd ..
go build -o ai-server main.go

# 运行服务
./ai-server
```

## 🔧 配置说明

### 服务配置
```yaml
server:
  port: 8080
  jwt_secret: "your-secret-key-here"
  frontend_path: "./frontend/dist"
```

### 默认账户
- **用户名**: admin
- **密码**: admin123

## 📋 API 接口

### 管理接口
- `POST /admin/login` - 用户登录
- `GET /admin/providers` - 获取供应商列表
- `POST /admin/providers` - 创建供应商
- `PUT /admin/providers/:id` - 更新供应商
- `DELETE /admin/providers/:id` - 删除供应商
- `GET /admin/endpoints` - 获取 API 路径列表
- `POST /admin/endpoints` - 创建 API 路径
- `PUT /admin/endpoints/:id` - 更新 API 路径
- `DELETE /admin/endpoints/:id` - 删除 API 路径
- `GET /admin/stats` - 获取统计信息
- `GET /admin/user/info` - 获取用户信息
- `PUT /admin/user/password` - 修改用户密码
- `PUT /admin/user/info` - 更新用户信息

### 代理接口
- `POST /{custom_path}` - 自定义 API 路径（通过管理后台配置）

## 🎨 管理后台功能

### 仪表盘
- 实时 API 调用统计
- Token 使用情况图表
- 系统状态监控

### LMM供应商管理
- 添加/编辑/删除供应商
- 配置供应商 API 地址和密钥
- 支持多种 LLM 供应商

### API 路径管理
- 创建自定义 API 路径
- 绑定供应商和系统提示词
- 配置流式输出选项

### API 测试
- 在线测试 API 接口
- 支持流式和非流式输出模式
- 实时查看响应内容

### 个人中心
- 查看和编辑用户信息
- 修改登录密码
- 安全设置

## 📊 技术架构

### 后端技术栈
- **语言**: Go
- **框架**: Gin
- **数据库**: SQLite (GORM)
- **认证**: JWT
- **缓存**: 内存缓存

### 前端技术栈
- **框架**: Vue 3
- **组件库**: Element Plus
- **构建工具**: Vite
- **路由**: Vue Router

## 🔒 安全说明

1. **JWT Token**: 管理接口使用 JWT 进行身份验证，有效期 24 小时
2. **API Key**: 客户端访问使用独立的 API Key，与管理后台账号分离
3. **输入验证**: 所有接口参数都会进行验证，防止注入攻击
4. **权限控制**: 不同功能模块有严格的权限控制

## 🛠️ 扩展开发

### 添加新的 LLM 供应商
1. 在供应商管理页面添加新供应商配置
2. 配置供应商的 API 地址、密钥和模型名称
3. 在 API 路径管理中绑定新供应商

### 自定义 API 路径
1. 创建新的 API 路径配置
2. 绑定对应的供应商
3. 设置系统提示词和流式输出选项

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来帮助改进项目。

## 📄 许可证

MIT License