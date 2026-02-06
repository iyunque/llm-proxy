<template>
  <div class="user-center">
    <el-card shadow="hover" class="profile-card">
      <template #header>
        <div class="card-header">
          <span>个人资料</span>
          <el-tag type="success">当前用户</el-tag>
        </div>
      </template>

      <el-form 
        :model="userInfo" 
        label-width="120px" 
        :disabled="!isEditing"
      >
        <el-form-item label="用户ID">
          <el-input v-model="userInfo.id" readonly />
        </el-form-item>
        
        <el-form-item label="用户名">
          <el-input 
            v-model="userInfo.username" 
            :readonly="!isEditing"
          />
        </el-form-item>
        
        <el-form-item label="创建时间">
          <el-input 
            :value="formatDate(userInfo.created)" 
            readonly 
          />
        </el-form-item>
        
        <el-form-item>
          <el-button 
            v-if="!isEditing" 
            type="primary" 
            @click="startEdit"
          >
            编辑信息
          </el-button>
          <div v-else>
            <el-button type="success" @click="saveUserInfo" :loading="saving">
              保存
            </el-button>
            <el-button @click="cancelEdit">取消</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="hover" class="password-card">
      <template #header>
        <div class="card-header">
          <span>修改密码</span>
          <el-tag type="warning">安全设置</el-tag>
        </div>
      </template>

      <el-form 
        :model="passwordForm" 
        label-width="120px"
        @submit.prevent="changePassword"
      >
        <el-form-item label="旧密码" prop="oldPassword" :rules="[{ required: true, message: '请输入旧密码' }]">
          <el-input 
            v-model="passwordForm.oldPassword" 
            type="password" 
            show-password
            placeholder="请输入当前密码"
          />
        </el-form-item>
        
        <el-form-item label="新密码" prop="newPassword" :rules="[
          { required: true, message: '请输入新密码' },
          { min: 6, message: '密码长度至少6位' }
        ]">
          <el-input 
            v-model="passwordForm.newPassword" 
            type="password" 
            show-password
            placeholder="请输入新密码（至少6位）"
          />
        </el-form-item>
        
        <el-form-item label="确认密码" prop="confirmPassword" :rules="[
          { required: true, message: '请确认新密码' },
          { validator: validateConfirmPassword, trigger: 'blur' }
        ]">
          <el-input 
            v-model="passwordForm.confirmPassword" 
            type="password" 
            show-password
            placeholder="请再次输入新密码"
          />
        </el-form-item>
        
        <el-form-item>
          <el-button 
            type="warning" 
            @click="changePassword" 
            :loading="changingPassword"
          >
            修改密码
          </el-button>
          <el-button @click="resetPasswordForm">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { ElMessage } from 'element-plus'

const userInfo = ref({
  id: '',
  username: '',
  created: ''
})

const isEditing = ref(false)
const saving = ref(false)
const changingPassword = ref(false)

const originalUserInfo = ref({})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

// 格式化日期
const formatDate = (dateString) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN')
}

// 验证确认密码
const validateConfirmPassword = (rule, value, callback) => {
  if (value !== passwordForm.newPassword) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

// 开始编辑
const startEdit = () => {
  originalUserInfo.value = { ...userInfo.value }
  isEditing.value = true
}

// 取消编辑
const cancelEdit = () => {
  userInfo.value = { ...originalUserInfo.value }
  isEditing.value = false
}

// 保存用户信息
const saveUserInfo = async () => {
  if (!userInfo.value.username.trim()) {
    ElMessage.error('用户名不能为空')
    return
  }

  try {
    saving.value = true
    await api.put('/user/info', {
      username: userInfo.value.username
    })
    
    ElMessage.success('用户信息更新成功')
    isEditing.value = false
    originalUserInfo.value = { ...userInfo.value }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '更新失败')
  } finally {
    saving.value = false
  }
}

// 修改密码
const changePassword = async () => {
  if (!passwordForm.oldPassword) {
    ElMessage.error('请输入旧密码')
    return
  }
  if (!passwordForm.newPassword) {
    ElMessage.error('请输入新密码')
    return
  }
  if (passwordForm.newPassword.length < 6) {
    ElMessage.error('新密码长度至少6位')
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    ElMessage.error('两次输入的密码不一致')
    return
  }

  try {
    changingPassword.value = true
    await api.put('/user/password', {
      oldPassword: passwordForm.oldPassword,
      newPassword: passwordForm.newPassword
    })
    
    ElMessage.success('密码修改成功，请重新登录')
    resetPasswordForm()
    
    // 延迟跳转到登录页
    setTimeout(() => {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }, 1500)
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '密码修改失败')
  } finally {
    changingPassword.value = false
  }
}

// 重置密码表单
const resetPasswordForm = () => {
  passwordForm.oldPassword = ''
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
}

// 获取用户信息
const fetchUserInfo = async () => {
  try {
    const data = await api.get('/user/info')
    userInfo.value = data
    originalUserInfo.value = { ...data }
  } catch (error) {
    ElMessage.error('获取用户信息失败')
  }
}

onMounted(() => {
  fetchUserInfo()
})
</script>

<style scoped>
.user-center {
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
}

.profile-card, .password-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  font-size: 16px;
}

.el-form-item {
  margin-bottom: 20px;
}

.el-button {
  margin-right: 10px;
}

:deep(.el-card__header) {
  border-bottom: 1px solid #ebeef5;
  padding: 18px 20px;
}

:deep(.el-card__body) {
  padding: 20px;
}
</style>