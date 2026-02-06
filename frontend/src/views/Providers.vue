<template>
  <div>
    <div class="toolbar">
      <el-input 
        v-model="searchText" 
        placeholder="搜索供应商名称" 
        style="width: 300px; margin-right: 10px;"
        clearable
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-button type="primary" @click="handleAdd">
        <el-icon style="margin-right: 5px;"><Plus /></el-icon>
        添加供应商
      </el-button>
    </div>

    <el-table :data="filteredData" border style="width: 100%" v-loading="loading">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="Name" label="供应商名称" />
      <el-table-column prop="ModelName" label="模型名称" width="180" />
      <el-table-column prop="APIAddress" label="API 地址" show-overflow-tooltip />
      <el-table-column label="API Key" width="200">
        <template #default="scope">
          <div class="key-cell">
            <span>{{ maskKey(scope.row.APIKey) }}</span>
            <el-button 
              link 
              type="primary" 
              @click="copyToClipboard(scope.row.APIKey)"
              size="small"
            >
              <el-icon><DocumentCopy /></el-icon>
            </el-button>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180">
        <template #default="scope">
          <el-button size="small" @click="handleEdit(scope.row)">编辑</el-button>
          <el-button size="small" type="danger" @click="handleDelete(scope.row.ID)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="filteredData.length === 0 && !loading" description="暂无供应商数据" />

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="600px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="form.Name" />
        </el-form-item>
        <el-form-item label="模型名称">
          <el-input v-model="form.ModelName" placeholder="例如: gpt-4, deepseek-chat, glm-4" />
        </el-form-item>
        <el-form-item label="API 地址">
          <el-input v-model="form.APIAddress" placeholder="例如: https://api.example.com/v1/chat" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input v-model="form.APIKey" type="password" show-password />
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
const searchText = ref('')
const loading = ref(false)
const saveLoading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const form = reactive({
  ID: null,
  Name: '',
  ModelName: '',
  APIAddress: '',
  APIKey: ''
})

const filteredData = computed(() => {
  if (!searchText.value) return tableData.value
  return tableData.value.filter(item => 
    item.Name.toLowerCase().includes(searchText.value.toLowerCase())
  )
})

const fetchData = async () => {
  loading.value = true
  try {
    const data = await api.get('/providers')
    tableData.value = data
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
  dialogTitle.value = '添加供应商'
  form.ID = null
  form.Name = ''
  form.ModelName = ''
  form.APIAddress = ''
  form.APIKey = ''
  dialogVisible.value = true
}

const handleEdit = (row) => {
  dialogTitle.value = '编辑供应商'
  Object.assign(form, row)
  dialogVisible.value = true
}

const handleSave = async () => {
  if (!form.Name || !form.ModelName || !form.APIAddress || !form.APIKey) {
    ElMessage.warning('请填写完整信息')
    return
  }
  saveLoading.value = true
  try {
    if (form.ID) {
      await api.put(`/providers/${form.ID}`, form)
    } else {
      await api.post('/providers', form)
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
    await api.delete(`/providers/${id}`)
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
