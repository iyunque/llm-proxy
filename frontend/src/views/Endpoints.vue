<template>
  <div>
    <div class="toolbar">
      <el-input 
        v-model="searchText" 
        placeholder="搜索端点" 
        style="width: 300px; margin-right: 10px;"
        clearable
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button type="primary" @click="handleAdd">
        <el-icon style="margin-right: 5px;"><Plus /></el-icon>
        添加 API 端点
      </el-button>
    </div>

    <el-table :data="filteredData" border style="width: 100%" v-loading="loading">
      <el-table-column prop="ID" label="ID" width="60" />
      <el-table-column prop="Path" label="访问路径" width="180">
        <template #default="scope">
          <el-tag type="primary">{{ scope.row.Path }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="API Key" width="200">
        <template #default="scope">
          <div class="key-cell">
            <span>{{ maskKey(scope.row.ApiKey) }}</span>
            <el-button 
              link 
              type="primary" 
              @click="copyToClipboard(scope.row.ApiKey)"
              size="small"
            >
              <el-icon><DocumentCopy /></el-icon>
            </el-button>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="Provider.Name" label="供应商" width="120" />
      <el-table-column prop="SelectedModel" label="大模型名称" width="180" />
      <el-table-column prop="SystemPrompt" label="系统提示词" show-overflow-tooltip />
      <el-table-column prop="StreamOutput" label="流式输出" width="100">
        <template #default="scope">
          <el-tag :type="scope.row.StreamOutput ? 'success' : 'info'">
            {{ scope.row.StreamOutput ? '开启' : '关闭' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="EnableThinking" label="思考模式" width="100">
        <template #default="scope">
          <el-tag :type="scope.row.EnableThinking ? 'success' : 'info'">
            {{ scope.row.EnableThinking ? '开启' : '关闭' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180">
        <template #default="scope">
          <el-button size="small" @click="handleEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="danger" @click="handleDelete(scope.row.ID)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="filteredData.length === 0 && !loading" description="暂无 API 端点数据" />

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="700px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="访问路径">
          <el-input v-model="form.Path" placeholder="例如 /api/translate" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="form.ApiKey" placeholder="32位以内的字符" show-password />
        </el-form-item>
        <el-form-item label="供应商">
          <el-select v-model="form.ProviderID" placeholder="选择供应商" style="width: 100%;" @change="handleProviderChange">
            <el-option v-for="p in providers" :key="p.ID" :label="p.Name" :value="p.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="大模型名称">
          <el-select v-model="form.SelectedModel" placeholder="请选择模型名称" style="width: 100%;">
            <el-option v-for="m in modelOptions" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="系统提示词">
          <el-input v-model="form.SystemPrompt" type="textarea" :rows="8" placeholder="输入系统提示词..." />
        </el-form-item>
        <el-form-item label="流式输出">
          <el-switch v-model="form.StreamOutput" />
        </el-form-item>
        <el-form-item label="思考模式">
          <el-switch v-model="form.EnableThinking" />
        </el-form-item>
        <el-form-item label="温度参数">
          <el-input-number 
            v-model="form.Temperature" 
            :min="0" 
            :max="2" 
            :step="0.1" 
            :precision="1"
            placeholder="0.0-2.0"
          />
          <div class="info-text">
            控制生成文本的随机性，0.0最确定，2.0最随机
          </div>
        </el-form-item>
        <el-divider>备用模型配置</el-divider>
        <el-form-item label="备用供应商1">
          <el-select v-model="form.FallbackProviderID1" placeholder="选择备用供应商1" style="width: 100%;" clearable @change="handleFallbackProviderChange1">
            <el-option v-for="p in providers" :key="p.ID" :label="p.Name" :value="p.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="备用模型1">
          <el-select v-model="form.FallbackModel1" placeholder="请选择备用模型1" style="width: 100%;" clearable>
            <el-option v-for="m in fallbackModelOptions1" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="备用供应商2">
          <el-select v-model="form.FallbackProviderID2" placeholder="选择备用供应商2" style="width: 100%;" clearable @change="handleFallbackProviderChange2">
            <el-option v-for="p in providers" :key="p.ID" :label="p.Name" :value="p.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="备用模型2">
          <el-select v-model="form.FallbackModel2" placeholder="请选择备用模型2" style="width: 100%;" clearable>
            <el-option v-for="m in fallbackModelOptions2" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave" :loading="saveLoading">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
import api from '../api'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus, DocumentCopy } from '@element-plus/icons-vue'

const tableData = ref([])
const providers = ref([])
const searchText = ref('')
const loading = ref(false)
const saveLoading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const form = reactive({
  ID: null,
  Path: '',
  ApiKey: '',
  ProviderID: null,
  SelectedModel: '',
  SystemPrompt: '',
  StreamOutput: false,
  EnableThinking: false,
  Temperature: 0.7,
  FallbackProviderID1: null,
  FallbackModel1: '',
  FallbackProviderID2: null,
  FallbackModel2: '',
})

const modelOptions = computed(() => {
  const provider = providers.value.find(p => p.ID === form.ProviderID)
  if (!provider || !provider.ModelName) return []
  return provider.ModelName.split(',').map(m => m.trim()).filter(m => m)
})

const fallbackModelOptions1 = computed(() => {
  const provider = providers.value.find(p => p.ID === form.FallbackProviderID1)
  if (!provider || !provider.ModelName) return []
  return provider.ModelName.split(',').map(m => m.trim()).filter(m => m)
})

const fallbackModelOptions2 = computed(() => {
  const provider = providers.value.find(p => p.ID === form.FallbackProviderID2)
  if (!provider || !provider.ModelName) return []
  return provider.ModelName.split(',').map(m => m.trim()).filter(m => m)
})

const filteredData = computed(() => {
  if (!searchText.value) return tableData.value
  return tableData.value.filter(item => 
    item.Path.toLowerCase().includes(searchText.value.toLowerCase())
  )
})

const fetchData = async () => {
  loading.value = true
  try {
    const [endpoints, providersData] = await Promise.all([
      api.get('/endpoints'),
      api.get('/providers')
    ])
    tableData.value = endpoints
    providers.value = providersData
  } finally {
    loading.value = false
  }
}

const maskKey = (key) => {
  if (!key || key.length < 8) return '***'
  return key.substring(0, 4) + '****' + key.substring(key.length - 4)
}

const copyToClipboard = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('API Key 已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

const handleAdd = () => {
  dialogTitle.value = '添加 API 端点'
  Object.assign(form, { 
    ID: null, 
    Path: '', 
    ApiKey: '', 
    ProviderID: null, 
    SelectedModel: '', 
    SystemPrompt: '', 
    StreamOutput: false, 
    EnableThinking: false, 
    Temperature: 0.7,
    FallbackProviderID1: null,
    FallbackModel1: '',
    FallbackProviderID2: null,
    FallbackModel2: '',
  })
  dialogVisible.value = true
}

const handleEdit = (row) => {
  dialogTitle.value = '编辑 API 端点'
  Object.assign(form, row)
  if (!form.SelectedModel && modelOptions.value.length > 0) {
    form.SelectedModel = modelOptions.value[0]
  }
  if (!form.FallbackModel1 && fallbackModelOptions1.value.length > 0) {
    form.FallbackModel1 = fallbackModelOptions1.value[0]
  }
  if (!form.FallbackModel2 && fallbackModelOptions2.value.length > 0) {
    form.FallbackModel2 = fallbackModelOptions2.value[0]
  }
  if (form.Temperature === undefined || form.Temperature === null) {
    form.Temperature = 0.7
  }
  dialogVisible.value = true
}

const handleProviderChange = (providerID) => {
  const options = modelOptions.value
  if (options.length > 0) {
    form.SelectedModel = options[0]
  } else {
    form.SelectedModel = ''
  }
}

const handleFallbackProviderChange1 = (providerID) => {
  const options = fallbackModelOptions1.value
  if (options.length > 0) {
    form.FallbackModel1 = options[0]
  } else {
    form.FallbackModel1 = ''
  }
}

const handleFallbackProviderChange2 = (providerID) => {
  const options = fallbackModelOptions2.value
  if (options.length > 0) {
    form.FallbackModel2 = options[0]
  } else {
    form.FallbackModel2 = ''
  }
}

const handleSave = async () => {
  if (!form.Path || !form.ApiKey || !form.ProviderID || !form.SelectedModel) {
    ElMessage.warning('请填写完整信息')
    return
  }
  saveLoading.value = true
  try {
    const data = { ...form }
    if (form.ID) {
      await api.put(`/endpoints/${form.ID}`, data)
    } else {
      await api.post('/endpoints', data)
    }
    ElMessage.success('保存成功')
    dialogVisible.value = false
    fetchData()
  } catch (e) {
  } finally {
    saveLoading.value = false
  }
}

const handleDelete = (id) => {
  ElMessageBox.confirm('确定删除吗?', '提示', { type: 'warning' }).then(async () => {
    await api.delete(`/endpoints/${id}`)
    ElMessage.success('删除成功')
    fetchData()
  })
}

onMounted(fetchData)
</script>

<style scoped>
.toolbar {
  margin-bottom: 20px;
  display: flex;
  align-items: center;
}

.key-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
