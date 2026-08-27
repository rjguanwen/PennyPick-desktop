<template>
  <div class="app-shell">
    <UnlockPanel v-if="locked" :status="dbStatus" @unlocked="onUnlocked" />
    <router-view v-else />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import UnlockPanel from './views/UnlockPanel.vue'

const router = useRouter()
const locked = ref(true)
const dbStatus = ref('encrypted')

onMounted(async () => {
  dbStatus.value = window.__PENNYPICK_DB_STATUS__ || 'encrypted'
  // 后端已用本机保存的密钥自动解锁：注入 API 地址后进入登录页
  if (dbStatus.value === 'ready') {
    try {
      const app = window.go && window.go.main && window.go.main.App
      window.__PENNYPICK_API_BASE__ = await app.GetAPIBaseURL()
      enterLogin()
    } catch (e) {
      console.error('[PennyPick] 获取 API 地址失败', e)
      dbStatus.value = 'encrypted'
    }
  }
})

function onUnlocked(apiBase) {
  window.__PENNYPICK_API_BASE__ = apiBase
  enterLogin()
}

// 数据库主密码解锁 ≠ 业务登录：无论是否曾登录过，解锁后都清空登录态并进入登录页，
// 由用户输入用户名/密码登录（登录用于区分不同账号）。
// 先完成路由跳转再渲染应用，避免短暂渲染受保护页面触发 401 错误提示。
async function enterLogin() {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  await router.replace('/login')
  locked.value = false
}
</script>

<style>
#app {
  height: 100%;
}
</style>
