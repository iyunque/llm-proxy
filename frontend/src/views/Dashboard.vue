<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stats-cards">
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card calls-card">
          <div class="card-icon">
            <el-icon :size="40"><DataLine /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-label">今日调用总数</div>
            <div class="card-value">{{ totalCalls.toLocaleString() }}</div>
            <div class="card-trend">较昨日 <span class="trend-up">+12%</span></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card failed-card">
          <div class="card-icon">
            <el-icon :size="40"><Warning /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-label">今日失败次数</div>
            <div class="card-value">{{ totalFailed.toLocaleString() }}</div>
            <div class="card-trend">失败率 <span class="trend-down">{{ failedRate }}%</span></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card input-card">
          <div class="card-icon">
            <el-icon :size="40"><Upload /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-label">今日输入 Tokens</div>
            <div class="card-value">{{ totalInput.toLocaleString() }}</div>
            <div class="card-trend">较昨日 <span class="trend-up">+8%</span></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card output-card">
          <div class="card-icon">
            <el-icon :size="40"><Download /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-label">今日输出 Tokens</div>
            <div class="card-value">{{ totalOutput.toLocaleString() }}</div>
            <div class="card-trend">较昨日 <span class="trend-up">+15%</span></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card cache-card">
          <div class="card-icon">
            <el-icon :size="40"><Lightning /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-label">今日缓存命中 Tokens</div>
            <div class="card-value">{{ totalCache.toLocaleString() }}</div>
            <div class="card-trend">较昨日 <span class="trend-up">+5%</span></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="4">
        <el-card shadow="hover" class="stat-card top-failed-card">
          <div class="card-icon">
            <el-icon :size="40"><CloseBold /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-label">最常失败模型</div>
            <div class="card-value">{{ topFailedModel }}</div>
            <div class="card-trend">失败 {{ topFailedCount }} 次</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="chart-header">
              <span>API 调用分布</span>
            </div>
          </template>
          <div ref="pieChart" style="width: 100%; height: 400px;"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="chart-header">
              <span>Token 使用对比</span>
            </div>
          </template>
          <div ref="barChart" style="width: 100%; height: 400px;"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="chart-header">
              <span>失败模型统计（全部）</span>
            </div>
          </template>
          <div ref="totalFailedChart" style="width: 100%; height: 400px;"></div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="chart-header">
              <span>失败模型统计（今日）</span>
            </div>
          </template>
          <div ref="todayFailedChart" style="width: 100%; height: 400px;"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, computed } from 'vue'
import api from '../api'
import * as echarts from 'echarts'
import { DataLine, Upload, Download, Lightning, Warning, CloseBold } from '@element-plus/icons-vue'

const totalCalls = ref(0)
const totalFailed = ref(0)
const totalInput = ref(0)
const totalOutput = ref(0)
const totalCache = ref(0)
const pieChart = ref(null)
const barChart = ref(null)
const totalFailedChart = ref(null)
const todayFailedChart = ref(null)
const statsData = ref([])

// 解析 FailedModels JSON 字段
const parseFailedModels = (failedModelsJson) => {
  if (!failedModelsJson) return {}
  try {
    return JSON.parse(failedModelsJson)
  } catch {
    return {}
  }
}

// 全部失败模型统计
const allFailedModels = computed(() => {
  const modelCounts = {}
  statsData.value.forEach(item => {
    const failedModels = parseFailedModels(item.FailedModels)
    Object.entries(failedModels).forEach(([model, count]) => {
      modelCounts[model] = (modelCounts[model] || 0) + count
    })
  })
  return modelCounts
})

// 今日失败模型统计
const todayFailedModels = computed(() => {
  const modelCounts = {}
  statsData.value.forEach(item => {
    const failedModels = parseFailedModels(item.FailedModels)
    Object.entries(failedModels).forEach(([model, count]) => {
      modelCounts[model] = (modelCounts[model] || 0) + count
    })
  })
  return modelCounts
})

const failedRate = computed(() => {
  if (totalCalls.value === 0) return '0.00'
  return ((totalFailed.value / totalCalls.value) * 100).toFixed(2)
})

const topFailedModel = computed(() => {
  if (statsData.value.length === 0) return '暂无'
  
  // 按失败次数排序，取失败次数最多的模型
  const sortedByFailed = [...statsData.value]
    .filter(item => item.FailedCallCount > 0)
    .sort((a, b) => b.FailedCallCount - a.FailedCallCount)
  
  if (sortedByFailed.length === 0) return '暂无'
  
  return sortedByFailed[0].LastFailedModel || '未知模型'
})

const topFailedCount = computed(() => {
  if (statsData.value.length === 0) return 0
  
  const sortedByFailed = [...statsData.value]
    .filter(item => item.FailedCallCount > 0)
    .sort((a, b) => b.FailedCallCount - a.FailedCallCount)
  
  return sortedByFailed.length > 0 ? sortedByFailed[0].FailedCallCount : 0
})

const fetchData = async () => {
  const response = await api.get('/stats')
  const data = response.data
  statsData.value = data
  totalCalls.value = data.reduce((acc, cur) => acc + cur.CallCount, 0)
  totalFailed.value = data.reduce((acc, cur) => acc + cur.FailedCallCount, 0)
  totalInput.value = data.reduce((acc, cur) => acc + cur.InputTokens, 0)
  totalOutput.value = data.reduce((acc, cur) => acc + cur.OutputTokens, 0)
  totalCache.value = data.reduce((acc, cur) => acc + cur.CacheHitTokens, 0)
  
  await nextTick()
  initCharts()
}

const initCharts = () => {
  // 饼图 - API 调用分布
  if (pieChart.value) {
    const pieInstance = echarts.init(pieChart.value)
    const pieData = statsData.value.map(item => ({
      name: `API ${item.APIEndpointID}`,
      value: item.CallCount
    }))
    
    pieInstance.setOption({
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)'
      },
      legend: {
        orient: 'vertical',
        right: 10,
        top: 'center'
      },
      series: [
        {
          name: '调用次数',
          type: 'pie',
          radius: ['40%', '70%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 10,
            borderColor: '#fff',
            borderWidth: 2
          },
          label: {
            show: false
          },
          emphasis: {
            label: {
              show: true,
              fontSize: 16,
              fontWeight: 'bold'
            }
          },
          data: pieData.length > 0 ? pieData : [{ name: '暂无数据', value: 1 }]
        }
      ]
    })
  }

  // 柱状图 - Token 对比
  if (barChart.value) {
    const barInstance = echarts.init(barChart.value)
    const categories = statsData.value.map(item => `API ${item.APIEndpointID}`)
    const inputData = statsData.value.map(item => item.InputTokens)
    const outputData = statsData.value.map(item => item.OutputTokens)
    const cacheData = statsData.value.map(item => item.CacheHitTokens)

    barInstance.setOption({
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        }
      },
      legend: {
        data: ['输入 Token', '输出 Token', '缓存命中 Token']
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: {
        type: 'category',
        data: categories.length > 0 ? categories : ['暂无数据']
      },
      yAxis: {
        type: 'value'
      },
      series: [
        {
          name: '输入 Token',
          type: 'bar',
          data: inputData.length > 0 ? inputData : [0],
          itemStyle: {
            color: '#409eff'
          }
        },
        {
          name: '输出 Token',
          type: 'bar',
          data: outputData.length > 0 ? outputData : [0],
          itemStyle: {
            color: '#67c23a'
          }
        },
        {
          name: '缓存命中 Token',
          type: 'bar',
          data: cacheData.length > 0 ? cacheData : [0],
          itemStyle: {
            color: '#e6a23c'
          }
        }
      ]
    })
  }

  // 柱状图 - 全部失败模型统计
  if (totalFailedChart.value) {
    const totalFailedInstance = echarts.init(totalFailedChart.value)
    const allFailed = allFailedModels.value
    const sortedModels = Object.entries(allFailed)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 20) // 限制显示前20个

    totalFailedInstance.setOption({
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: {
        type: 'category',
        data: sortedModels.length > 0 ? sortedModels.map(([m]) => m) : ['暂无数据'],
        axisLabel: {
          rotate: 45,
          interval: 0
        }
      },
      yAxis: {
        type: 'value',
        name: '失败次数'
      },
      series: [
        {
          name: '失败次数',
          type: 'bar',
          data: sortedModels.length > 0 ? sortedModels.map(([, c]) => c) : [0],
          itemStyle: {
            color: '#f56c6c'
          },
          label: {
            show: true,
            position: 'top'
          }
        }
      ]
    })
  }

  // 柱状图 - 今日失败模型统计
  if (todayFailedChart.value) {
    const todayFailedInstance = echarts.init(todayFailedChart.value)
    const todayFailed = todayFailedModels.value
    const sortedModels = Object.entries(todayFailed)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 20) // 限制显示前20个

    todayFailedInstance.setOption({
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        }
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        containLabel: true
      },
      xAxis: {
        type: 'category',
        data: sortedModels.length > 0 ? sortedModels.map(([m]) => m) : ['暂无数据'],
        axisLabel: {
          rotate: 45,
          interval: 0
        }
      },
      yAxis: {
        type: 'value',
        name: '失败次数'
      },
      series: [
        {
          name: '失败次数',
          type: 'bar',
          data: sortedModels.length > 0 ? sortedModels.map(([, c]) => c) : [0],
          itemStyle: {
            color: '#e6a23c'
          },
          label: {
            show: true,
            position: 'top'
          }
        }
      ]
    })
  }
}

onMounted(fetchData)
</script>

<style scoped>
.dashboard {
  padding: 10px;
}

.stats-cards {
  margin-bottom: 20px;
}

.stat-card {
  display: flex;
  align-items: center;
  padding: 20px;
  min-height: 140px;
  cursor: pointer;
  transition: transform 0.3s, box-shadow 0.3s;
  position: relative;
  overflow: hidden;
}

.stat-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12) !important;
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 100px;
  height: 100px;
  border-radius: 50%;
  opacity: 0.1;
}

.calls-card::before {
  background: #409eff;
}

.input-card::before {
  background: #67c23a;
}

.output-card::before {
  background: #e6a23c;
}

.card-icon {
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  margin-right: 20px;
  flex-shrink: 0;
}

.calls-card .card-icon {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
}

.input-card .card-icon {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: #fff;
}

.output-card .card-icon {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: #fff;
}

.card-content {
  flex: 1;
  min-width: 0;
}

.card-label {
  font-size: 14px;
  color: #999;
  margin-bottom: 8px;
}

.card-value {
  font-size: 32px;
  font-weight: bold;
  color: #333;
  margin-bottom: 8px;
  line-height: 1.2;
}

.card-trend {
  font-size: 13px;
  color: #666;
}

.trend-up {
  color: #67c23a;
  font-weight: 600;
}

.chart-header {
  font-weight: 600;
  font-size: 16px;
}
</style>
