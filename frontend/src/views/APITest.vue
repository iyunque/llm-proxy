<template>
  <div class="api-test">
    <el-card shadow="hover" class="test-card">
      <template #header>
        <div class="card-header">
          <span>API 接口测试</span>
          <el-tag type="success">实时测试</el-tag>
        </div>
      </template>

      <el-form :model="testForm" label-width="120px">
        <el-form-item label="选择 API 路径">
          <el-select 
            v-model="testForm.endpointID" 
            placeholder="请选择要测试的 API 路径"
            style="width: 100%;"
            @change="handleEndpointChange"
          >
            <el-option 
              v-for="endpoint in endpoints" 
              :key="endpoint.ID" 
              :label="`${endpoint.Path} (${endpoint.Provider.Name})`" 
              :value="endpoint.ID"
            >
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <span>{{ endpoint.Path }}</span>
                <el-tag size="small" type="info">{{ endpoint.Provider.Name }}</el-tag>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item label="API Key">
          <el-input 
            v-model="testForm.apiKey" 
            placeholder="自动加载该 API 的访问密钥"
            readonly
            show-password
          />
          <div class="info-text">
            根据选择的 API 路径自动加载，不可编辑
          </div>
        </el-form-item>

        <el-form-item label="流式输出">
          <el-switch 
            v-model="testForm.streamOutput" 
            disabled
            :active-text="selectedEndpoint && selectedEndpoint.StreamOutput ? '启用' : '未配置'"
            :inactive-text="selectedEndpoint && selectedEndpoint.StreamOutput ? '禁用' : '不支持'"
          />
          <div class="info-text">
            &nbsp;&nbsp;根据 API 路径配置自动设置，不可编辑
          </div>
        </el-form-item>

        <el-form-item label="系统提示词">
          <el-input 
            v-model="currentPrompt" 
            type="textarea" 
            :rows="3" 
            readonly
            placeholder="系统提示词（自动加载）"
          />
        </el-form-item>

        <el-form-item label="测试内容">
          <el-input 
            v-model="testForm.content" 
            type="textarea" 
            :rows="6" 
            placeholder="输入您要发送给 AI 的内容..."
          />
        </el-form-item>

        <el-form-item>
          <el-button 
            type="primary" 
            @click="handleTest" 
            :loading="testing"
            size="large"
            :disabled="eventSource !== null"
          >
            <el-icon style="margin-right: 5px;"><Promotion /></el-icon>
            {{ eventSource ? '流式接收中...' : (testing ? '测试中...' : '开始测试') }}
          </el-button>
          <el-button @click="handleClear">清空结果</el-button>
        </el-form-item>
      </el-form>

      <el-alert 
        v-if="selectedEndpoint && selectedEndpoint.StreamOutput" 
        title="注意：该 API 路径配置为流式输出，测试结果将逐字显示" 
        type="info" 
        show-icon 
        :closable="false"
      />
    </el-card>

    <el-card shadow="hover" class="result-card" v-if="testResult">
      <template #header>
        <div class="card-header">
          <span>测试结果</span>
          <el-tag :type="testResult.success ? 'success' : 'danger'">
            {{ testResult.success ? '成功' : '失败' }}
          </el-tag>
        </div>
      </template>

      <div v-if="testResult.success">
        <div class="result-section">
          <h4>AI 响应内容</h4>
          <div 
            class="response-content" 
            :class="{ streaming: testForm.streamOutput && isStreaming }"
          >
            <div v-if="testForm.streamOutput && isStreaming">
              <div class="streaming-content">
                <span v-for="(char, index) in streamedContent.split('')" 
                      :key="index" 
                      :style="{ animationDelay: `${index * 10}ms` }"
                      class="stream-char"
                >{{ char }}</span>
              </div>
              <div class="stream-status">
                <el-tag type="warning" size="small">流式输出中...</el-tag>
              </div>
            </div>
            <div v-else-if="testForm.streamOutput && !isStreaming">
              <div class="final-content">
                <span>{{ testResult.response }}</span>
              </div>
              <div class="stream-status">
                <el-tag type="success" size="small">流式输出完成</el-tag>
              </div>
            </div>
            <div v-else>
              {{ testResult.response }}
            </div>
          </div>
        </div>

        <el-divider />

        <div class="result-section">
          <h4>Token 使用统计</h4>
          <el-row :gutter="20">
            <el-col :span="8">
              <div class="stat-item">
                <div class="stat-label">输入 Tokens</div>
                <div class="stat-value input">{{ testResult.usage.prompt_tokens }}</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="stat-item">
                <div class="stat-label">输出 Tokens</div>
                <div class="stat-value output">{{ testResult.usage.completion_tokens }}</div>
              </div>
            </el-col>
            <el-col :span="8">
              <div class="stat-item">
                <div class="stat-label">总计 Tokens</div>
                <div class="stat-value total">{{ testResult.usage.total_tokens }}</div>
              </div>
            </el-col>
          </el-row>
        </div>

        <el-divider />

        <div class="result-section">
          <h4>请求信息</h4>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="请求路径">{{ testResult.path }}</el-descriptions-item>
            <el-descriptions-item label="响应时间">{{ testResult.duration }} ms</el-descriptions-item>
            <el-descriptions-item label="模型">{{ testResult.model || 'default' }}</el-descriptions-item>
            <el-descriptions-item label="请求 ID">{{ testResult.id || 'N/A' }}</el-descriptions-item>
            <el-descriptions-item label="输出模式">
              <el-tag :type="testForm.streamOutput ? 'success' : 'info'">
                {{ testForm.streamOutput ? '流式' : '一次性' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </div>

      <div v-else class="error-section">
        <el-alert
          :title="testResult.error"
          type="error"
          :description="testResult.details"
          show-icon
          :closable="false"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
import api from '../api'
import { ElMessage } from 'element-plus'
import { Promotion } from '@element-plus/icons-vue'

const endpoints = ref([])
const testing = ref(false)
const testResult = ref(null)
const currentPrompt = ref('')
const eventSource = ref(null)  // 用于管理 EventSource 连接
const isStreaming = ref(false)
const streamedContent = ref('')

const testForm = reactive({
  endpointID: null,
  apiKey: '',
  content: '',
  streamOutput: false
})

const selectedEndpoint = computed(() => {
  return endpoints.value.find(e => e.ID === testForm.endpointID)
})

const fetchEndpoints = async () => {
  try {
    const data = await api.get('/endpoints')
    endpoints.value = data
  } catch (e) {
    ElMessage.error('加载 API 路径失败')
  }
}

const handleEndpointChange = () => {
  if (selectedEndpoint.value) {
    currentPrompt.value = selectedEndpoint.value.SystemPrompt || ''
    testForm.apiKey = selectedEndpoint.value.ApiKey || ''
    testForm.streamOutput = selectedEndpoint.value.StreamOutput
  }
}

const handleTest = async () => {
  if (!testForm.endpointID) {
    ElMessage.warning('请选择 API 路径')
    return
  }
  if (!testForm.apiKey) {
    ElMessage.warning('请输入 API Key')
    return
  }
  if (!testForm.content) {
    ElMessage.warning('请输入测试内容')
    return
  }

  // 清理之前的连接
  if (eventSource.value) {
    eventSource.value.close()
    eventSource.value = null
  }

  testing.value = true
  testResult.value = null
  streamedContent.value = ''
  const startTime = Date.now()

  try {
    const endpoint = selectedEndpoint.value
    
    if (testForm.streamOutput) {
      // 流式输出处理
      isStreaming.value = true
      // 使用相对路径而不是绝对 URL，以便后端路由能正确识别
      const url = endpoint.Path
      
      // 创建 POST 请求的数据
      const requestBody = JSON.stringify({ content: testForm.content })
      
      // 使用 fetch 发送 POST 请求并处理流式响应
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': testForm.apiKey
        },
        body: requestBody
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}, statusText: ${response.statusText}`)
      }

      testResult.value = {
        success: true,
        response: '',
        usage: { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 },
        path: endpoint.Path,
        duration: 0,
        model: '',
        id: ''
      }

      // 检查是否是流式响应
      const contentType = response.headers.get('Content-Type')
      if (!contentType || !contentType.includes('text/event-stream')) {
        // 如果不是流式响应，按普通响应处理
        const textData = await response.text()
        try {
          const jsonData = JSON.parse(textData)
          testResult.value.response = jsonData.choices?.[0]?.message?.content || '无响应内容'
          testResult.value.usage = jsonData.usage || { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 }
          testResult.value.model = jsonData.model
          testResult.value.id = jsonData.id
        } catch (e) {
          testResult.value.response = textData
        }
        
        const duration = Date.now() - startTime
        testResult.value.duration = duration
        isStreaming.value = false
        ElMessage.success('测试完成')
        return
      }

      // 读取流式响应
      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        try {
          const { done, value } = await reader.read()
          if (done) break

          const chunk = decoder.decode(value, { stream: true })
          buffer += chunk

          // 按行分割缓冲区内容
          const lines = buffer.split('\n')
          // 保留最后一行（可能不完整）
          buffer = lines.pop() || ''

          // 处理每一行
          for (const line of lines) {
            if (line.trim() === '') continue
            
            // 检查是否是 SSE 格式的数据行
            if (line.startsWith('data: ')) {
              const data = line.slice(6).trim() // 移除 'data: ' 前缀
              
              if (data === '[DONE]') {
                break // 流结束
              }
              
              try {
                const parsed = JSON.parse(data)
                
                // 检查是否是 OpenAI 格式的流数据
                if (parsed.choices && parsed.choices[0]) {
                  const choice = parsed.choices[0]
                  if (choice.delta && choice.delta.content) {
                    // 追加内容到显示区域
                    streamedContent.value += choice.delta.content
                    testResult.value.response = streamedContent.value
                  }
                  // 更新使用统计
                  if (parsed.usage) {
                    testResult.value.usage = parsed.usage
                  }
                  // 更新模型和ID信息
                  if (parsed.model) testResult.value.model = parsed.model
                  if (parsed.id) testResult.value.id = parsed.id
                } else if (parsed.error) {
                  // 处理错误情况
                  testResult.value.success = false
                  testResult.value.error = parsed.error
                  break
                }
              } catch (e) {
                // 如果不是 JSON，当作普通文本处理
                if (data.trim() && data !== '[DONE]') {
                  streamedContent.value += data
                  testResult.value.response = streamedContent.value
                }
              }
            } else {
              // 不是标准 SSE 格式，直接追加
              if (line.trim() && !line.startsWith(':')) { // 忽略注释行
                streamedContent.value += line
                testResult.value.response = streamedContent.value
              }
            }
          }
        } catch (readError) {
          console.error('读取流数据错误:', readError)
          break
        }
      }

      // 处理缓冲区中剩余的内容
      if (buffer.trim() && !buffer.startsWith('data: [DONE]')) {
        if (buffer.startsWith('data: ')) {
          const data = buffer.slice(6).trim()
          if (data !== '[DONE]' && data.trim()) {
            try {
              const parsed = JSON.parse(data)
              if (parsed.choices && parsed.choices[0] && parsed.choices[0].delta) {
                const content = parsed.choices[0].delta.content
                if (content) {
                  streamedContent.value += content
                  testResult.value.response = streamedContent.value
                }
              }
            } catch (e) {
              streamedContent.value += data
              testResult.value.response = streamedContent.value
            }
          }
        } else if (buffer.trim() && !buffer.startsWith(':')) {
          streamedContent.value += buffer
          testResult.value.response = streamedContent.value
        }
      }

      // 完成流式输出
      const duration = Date.now() - startTime
      testResult.value.duration = duration
      isStreaming.value = false
      
      ElMessage.success('流式测试完成')
    } else {
      // 非流式输出处理
      const axios = (await import('axios')).default
      const response = await axios.post(
        endpoint.Path,
        { content: testForm.content },
        {
          headers: {
            'X-API-Key': testForm.apiKey,
            'Content-Type': 'application/json'
          },
          baseURL: window.location.origin
        }
      )

      const duration = Date.now() - startTime
      const data = response.data

      testResult.value = {
        success: true,
        response: data.choices?.[0]?.message?.content || '无响应内容',
        usage: data.usage || { prompt_tokens: 0, completion_tokens: 0, total_tokens: 0 },
        path: endpoint.Path,
        duration: duration,
        model: data.model,
        id: data.id
      }

      ElMessage.success('测试成功')
    }
  } catch (error) {
    console.error('API 测试错误:', error)
    testResult.value = {
      success: false,
      error: error.message || '请求失败',
      details: error.toString() || '未知错误'
    }
    ElMessage.error(`测试失败: ${error.message || '未知错误'}`)
    isStreaming.value = false
  } finally {
    testing.value = false
  }
}

const handleClear = () => {
  testResult.value = null
  testForm.content = ''
  streamedContent.value = ''
  
  if (eventSource.value) {
    eventSource.value.close()
    eventSource.value = null
  }
}

onMounted(fetchEndpoints)
</script>

<style scoped>
.api-test {
  padding: 10px;
}

.test-card, .result-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  font-size: 16px;
}

.info-text {
  font-size: 12px;
  color: #999;
  margin-top: 5px;
}

.result-section {
  margin-bottom: 20px;
}

.result-section h4 {
  margin: 0 0 15px 0;
  font-size: 15px;
  color: #333;
  font-weight: 600;
}

.response-content {
  padding: 15px;
  background: #f5f7fa;
  border-radius: 8px;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
  min-height: 100px;
  max-height: 400px;
  overflow-y: auto;
}

.response-content.streaming {
  min-height: 150px;
}

.streaming-content {
  white-space: pre-wrap;
  word-break: break-word;
}

.stream-char {
  display: inline-block;
  animation: fadeInChar 0.05s ease-out;
}

@keyframes fadeInChar {
  from { opacity: 0; transform: translateY(5px); }
  to { opacity: 1; transform: translateY(0); }
}

.final-content {
  white-space: pre-wrap;
  word-break: break-word;
}

.stream-status {
  margin-top: 10px;
}

.stat-item {
  text-align: center;
  padding: 15px;
  background: #f5f7fa;
  border-radius: 8px;
  transition: all 0.3s;
}

.stat-item:hover {
  background: #e8f4ff;
  transform: translateY(-2px);
}

.stat-label {
  font-size: 13px;
  color: #999;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
}

.stat-value.input {
  color: #409eff;
}

.stat-value.output {
  color: #67c23a;
}

.stat-value.total {
  color: #e6a23c;
}

.error-section {
  padding: 10px 0;
}
</style>