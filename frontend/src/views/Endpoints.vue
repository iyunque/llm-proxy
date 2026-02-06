<template>
  <div>
    <div class="toolbar">
      <el-input 
        v-model="searchText" 
        placeholder="搜索路径" 
        style="width: 300px; margin-right: 10px;"
        clearable
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button type="primary" @click="handleAdd">
        <el-icon style="margin-right: 5px;"><Plus /></el-icon>
        添加 API 路径
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
      <el-table-column prop="SystemPrompt" label="系统提示词" show-overflow-tooltip />
      <el-table-column prop="StreamOutput" label="流式输出" width="100">
        <template #default="scope">
          <el-tag :type="scope.row.StreamOutput ? 'success' : 'info'">
            {{ scope.row.StreamOutput ? '开启' : '关闭' }}
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

    <el-empty v-if="filteredData.length === 0 && !loading" description="暂无 API 路径数据" />

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="700px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="访问路径">
          <el-input v-model="form.Path" placeholder="例如 /api/translate" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="form.ApiKey" placeholder="32位以内的字符" show-password />
        </el-form-item>
        <el-form-item label="供应商">
          <el-select v-model="form.ProviderID" placeholder="选择供应商" style="width: 100%;">
            <el-option v-for="p in providers" :key="p.ID" :label="p.Name" :value="p.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="系统提示词">
          <el-input v-model="form.SystemPrompt" type="textarea" :rows="8" placeholder="输入系统提示词..." />
        </el-form-item>
        <el-form-item label="流式输出">
          <el-switch v-model="form.StreamOutput" />
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
  SystemPrompt: '',
  StreamOutput: false,
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
  dialogTitle.value = '添加 API 路径'
  Object.assign(form, { ID: null, Path: '', ApiKey: '', ProviderID: null, SystemPrompt: '', StreamOutput: false })
  dialogVisible.value = true
}

const handleEdit = (row) => {
  dialogTitle.value = '编辑 API 路径'
  Object.assign(form, row)
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!form.Path || !form.ApiKey || !form.ProviderID) {
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
