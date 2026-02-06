<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stats-cards">
      <el-col :span="8">
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
      <el-col :span="8">
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
      <el-col :span="8">
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
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import api from '../api'
import * as echarts from 'echarts'
import { DataLine, Upload, Download } from '@element-plus/icons-vue'

const totalCalls = ref(0)
const totalInput = ref(0)
const totalOutput = ref(0)
const pieChart = ref(null)
const barChart = ref(null)
const statsData = ref([])

const fetchData = async () => {
  const data = await api.get('/stats')
  statsData.value = data
  totalCalls.value = data.reduce((acc, cur) => acc + cur.CallCount, 0)
  totalInput.value = data.reduce((acc, cur) => acc + cur.InputTokens, 0)
  totalOutput.value = data.reduce((acc, cur) => acc + cur.OutputTokens, 0)
  
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
    
    barInstance.setOption({
      tooltip: {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow'
        }
      },
      legend: {
        data: ['输入 Token', '输出 Token']
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
