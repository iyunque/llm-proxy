<template>
  <el-container class="layout-container">
    <el-aside :width="isCollapse ? '64px' : '200px'" class="sidebar">
      <div class="logo-container">
        <transition name="fade">
          <h2 v-if="!isCollapse" class="logo-text">LLM Proxy</h2>
          <h2 v-else class="logo-text-mini">AI</h2>
        </transition>
      </div>
      <el-menu 
        :default-active="activePath" 
        router 
        class="el-menu-vertical" 
        :collapse="isCollapse"
        background-color="#001529"
        text-color="#fff"
        active-text-color="#409eff"
      >
        <el-menu-item index="/">
          <el-icon><Monitor /></el-icon>
          <template #title><span>仪表盘</span></template>
        </el-menu-item>
        <el-menu-item index="/providers">
          <el-icon><Connection /></el-icon>
          <template #title><span>LMM供应商</span></template>
        </el-menu-item>
        <el-menu-item index="/endpoints">
          <el-icon><Link /></el-icon>
          <template #title><span>API 路径</span></template>
        </el-menu-item>
        <el-menu-item index="/stats">
          <el-icon><DataLine /></el-icon>
          <template #title><span>调用统计</span></template>
        </el-menu-item>
        <el-menu-item index="/test">
          <el-icon><Operation /></el-icon>
          <template #title><span>API 测试</span></template>
        </el-menu-item>
        <el-menu-item index="/user-center">
          <el-icon><User /></el-icon>
          <template #title><span>个人中心</span></template>
        </el-menu-item>
      </el-menu>
    </el-aside>
    
    <el-container>
      <el-header class="header">
        <div class="header-left">
          <el-icon class="collapse-icon" @click="toggleCollapse">
            <Fold v-if="!isCollapse" />
            <Expand v-else />
          </el-icon>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="breadcrumb">{{ breadcrumb }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-icon><Avatar /></el-icon>
              <span>管理员</span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">个人中心</el-dropdown-item>
                <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="main-content">
        <router-view></router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Monitor, Connection, Link, DataLine, Operation, Fold, Expand, Avatar, User } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()
const activePath = computed(() => route.path)
const isCollapse = ref(false)

const breadcrumbMap = {
  '/': '仪表盘',
  '/providers': 'AI 供应商管理',
  '/endpoints': 'API 路径管理',
  '/stats': '调用统计查询',
  '/test': 'API 接口测试',
  '/user-center': '个人中心'
}

const breadcrumb = computed(() => breadcrumbMap[route.path] || '')

const toggleCollapse = () => {
  isCollapse.value = !isCollapse.value
}

const handleCommand = (command) => {
  if (command === 'logout') {
    localStorage.removeItem('token')
    router.push('/login')
    ElMessage.success('退出成功')
  } else if (command === 'profile') {
    router.push('/user-center')
  }
}
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

.sidebar {
  background-color: #001529;
  transition: width 0.3s;
  overflow: hidden;
}

.logo-container {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #002140;
  color: #fff;
}

.logo-text {
  margin: 0;
  font-size: 20px;
  font-weight: bold;
  white-space: nowrap;
}

.logo-text-mini {
  margin: 0;
  font-size: 24px;
  font-weight: bold;
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

.el-menu-vertical {
  height: calc(100% - 64px);
  border-right: none;
}

.header {
  background-color: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.collapse-icon {
  font-size: 20px;
  cursor: pointer;
  transition: color 0.3s;
}

.collapse-icon:hover {
  color: #409eff;
}

.header-right {
  display: flex;
  align-items: center;
}

.user-info {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #333;
  transition: color 0.3s;
}

.user-info:hover {
  color: #409eff;
}

.main-content {
  background-color: #f5f7fa;
  padding: 20px;
}
</style>
