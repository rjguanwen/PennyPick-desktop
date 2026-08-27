<template>
  <div class="unlock-container">
    <div class="unlock-card">
      <div class="app-title">拾财 PennyPick</div>
      <p class="subtitle">{{ subtitle }}</p>

      <el-form @submit.prevent="submit">
        <el-form-item>
          <el-input
            v-model="pass"
            type="password"
            show-password
            :placeholder="passPlaceholder"
            size="large"
            autofocus
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-form-item v-if="isFirstTime">
          <el-input
            v-model="confirmPass"
            type="password"
            show-password
            placeholder="请再次输入主密码确认"
            size="large"
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="submit">
            {{ submitText }}
          </el-button>
        </el-form-item>
      </el-form>

      <p v-if="err" class="err-msg">{{ err }}</p>
      <p class="tip">主密码用于加密本地数据库；设置后本机会加密记住，下次启动自动解锁，无需再次输入。请务必牢记。</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  status: { type: String, default: 'encrypted' },
})
const emit = defineEmits(['unlocked'])

const pass = ref('')
const confirmPass = ref('')
const loading = ref(false)
const err = ref('')

const isFirstTime = computed(() => props.status === 'new' || props.status === 'plain')
const subtitle = computed(() => {
  if (props.status === 'plain') return '检测到已有账本数据，设置主密码后将改为加密保存'
  if (props.status === 'new') return '欢迎使用，请设置数据库主密码以加密保护您的账本'
  return '请输入数据库主密码解锁'
})
const passPlaceholder = computed(() => (isFirstTime.value ? '请设置主密码（至少 8 位）' : '请输入主密码'))
const submitText = computed(() => (isFirstTime.value ? '设置并解锁' : '解锁'))

async function submit() {
  err.value = ''
  if (!pass.value) {
    err.value = '请输入主密码'
    return
  }
  if (isFirstTime.value) {
    if (pass.value.length < 8) {
      err.value = '主密码至少 8 位'
      return
    }
    if (pass.value !== confirmPass.value) {
      err.value = '两次输入的密码不一致'
      return
    }
  }
  loading.value = true
  try {
    const app = window.go && window.go.main && window.go.main.App
    if (!app || typeof app.UnlockDatabase !== 'function') {
      throw new Error('桌面服务不可用')
    }
    const apiBase = await app.UnlockDatabase(pass.value)
    emit('unlocked', apiBase)
  } catch (e) {
    err.value = (e && e.message) || String(e) || '解锁失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.unlock-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f6f7fa 0%, #e8ecf5 100%);
}
.unlock-card {
  width: 380px;
  padding: 40px 36px 28px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 30px rgba(31, 45, 61, 0.12);
  text-align: center;
}
.app-title {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 8px;
}
.subtitle {
  color: #909399;
  font-size: 13px;
  margin-bottom: 24px;
}
.err-msg {
  color: #f56c6c;
  font-size: 13px;
  margin-top: 4px;
}
.tip {
  color: #c0c4cc;
  font-size: 12px;
  margin-top: 16px;
}
</style>
