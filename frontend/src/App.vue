<template>
  <router-view />
</template>

<script setup>
</script>

<style>
:root {
  --el-color-primary: #409eff;
  --bg-color: #f5f7fa;
  --card-shadow: 0 4px 12px 0 rgba(0, 0, 0, 0.05);
  --border-radius: 8px;
}

body {
  margin: 0;
  padding: 0;
  background-color: var(--bg-color);
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
}

.el-card {
  border-radius: var(--border-radius) !important;
  border: none !important;
  box-shadow: var(--card-shadow) !important;
}

.el-button {
  border-radius: 6px !important;
}
</style>
