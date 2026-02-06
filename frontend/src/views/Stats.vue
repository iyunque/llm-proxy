<template>
  <div>
    <div style="margin-bottom: 20px;">
      <el-date-picker
        v-model="selectedDate"
        type="date"
        placeholder="选择日期"
        value-format="YYYY-MM-DD"
        @change="fetchData"
      />
    </div>

    <el-table :data="tableData" border style="width: 100%">
      <el-table-column prop="APIEndpointID" label="路径 ID" width="100" />
      <el-table-column prop="CallCount" label="调用次数" />
      <el-table-column prop="InputTokens" label="输入 Tokens" />
      <el-table-column prop="OutputTokens" label="输出 Tokens" />
      <el-table-column prop="CacheHitTokens" label="缓存命中 Tokens" />
      <el-table-column prop="LastUpdated" label="最后更新时间">
        <template #default="scope">
          {{ new Date(scope.row.LastUpdated).toLocaleString() }}
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const selectedDate = ref(new Date().toISOString().split('T')[0])
const tableData = ref([])

const fetchData = async () => {
  const data = await api.get('/stats', { params: { date: selectedDate.value } })
  tableData.value = data
}

onMounted(fetchData)
</script>
