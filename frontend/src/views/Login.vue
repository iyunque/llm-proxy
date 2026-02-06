<template>
  <div class="login-container">
    <div class="login-bg">
      <div class="particle"></div>
      <div class="particle"></div>
      <div class="particle"></div>
    </div>
    <el-card class="login-card">
      <div class="login-header">
        <h2>AI API 管理平台</h2>
        <p class="subtitle">智能 API 转发与管理系统</p>
      </div>
      <el-form :model="form" @submit.prevent="handleLogin" autocomplete="off">
        <el-form-item>
          <el-input 
            v-model="form.username" 
            placeholder="请输入用户名"
            prefix-icon="User"
            size="large"
            autocomplete="username"
          ></el-input>
        </el-form-item>
        <el-form-item>
          <el-input 
            v-model="form.password" 
            type="password" 
            placeholder="请输入密码"
            prefix-icon="Lock"
            size="large"
            autocomplete="current-password"
            @keyup.enter="handleLogin"
          ></el-input>
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="rememberMe">记住我</el-checkbox>
        </el-form-item>
        <el-form-item>
          <el-button 
            type="primary" 
            @click="handleLogin" 
            :loading="loading" 
            size="large"
            class="login-btn"
          >
            {{ loading ? '登录中...' : '登 录' }}
          </el-button>
        </el-form-item>
      </el-form>
      <div class="login-footer">
        <p>默认账户：admin / admin123</p>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const rememberMe = ref(false)
const form = reactive({
  username: '',
  password: ''
})

const handleLogin = async () => {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const res = await api.post('/login', form)
    localStorage.setItem('token', res.token)
    if (rememberMe.value) {
      localStorage.setItem('username', form.username)
    }
    ElMessage.success('登录成功')
    router.push('/')
  } catch (e) {
    // 错误已由 api 拦截器处理
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100vh;
  overflow: hidden;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-bg {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.particle {
  position: absolute;
  width: 300px;
  height: 300px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  animation: float 20s infinite;
}

.particle:nth-child(1) {
  top: 10%;
  left: 20%;
  animation-delay: 0s;
}

.particle:nth-child(2) {
  top: 60%;
  left: 70%;
  animation-delay: 7s;
}

.particle:nth-child(3) {
  top: 40%;
  left: 40%;
  animation-delay: 14s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) rotate(0deg);
  }
  50% {
    transform: translateY(-50px) rotate(180deg);
  }
}

.login-card {
  position: relative;
  z-index: 10;
  width: 450px;
  padding: 40px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  border-radius: 16px;
  animation: slideIn 0.5s ease-out;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.login-header {
  text-align: center;
  margin-bottom: 40px;
}

.login-header h2 {
  margin: 0 0 10px 0;
  font-size: 28px;
  font-weight: 600;
  color: #333;
}

.subtitle {
  margin: 0;
  font-size: 14px;
  color: #999;
}

.login-btn {
  width: 100%;
  height: 45px;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 2px;
  transition: all 0.3s;
}

.login-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.4);
}

.login-footer {
  margin-top: 20px;
  text-align: center;
}

.login-footer p {
  margin: 0;
  font-size: 13px;
  color: #999;
}
</style>
