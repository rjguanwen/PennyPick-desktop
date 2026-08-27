import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

import App from './App.vue'
import router from './router'
import './styles/base.css'

const app = createApp(App)

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// 全局错误处理：避免单个组件渲染错误导致整页白屏，并输出诊断信息
app.config.errorHandler = (err, _instance, info) => {
  console.error('[PennyPick Vue 错误]', info, err)
}
window.addEventListener('unhandledrejection', (e) => {
  console.error('[PennyPick 未处理的 Promise 错误]', e.reason)
})

app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn })

// 桌面版：先查询数据库状态（决定是否显示解锁界面），再挂载应用。
// API 基础地址在解锁成功后由 App.vue 注入 __PENNYPICK_API_BASE__。
async function bootstrap() {
  if (window.go && window.go.main && window.go.main.App && typeof window.go.main.App.GetDatabaseStatus === 'function') {
    try {
      window.__PENNYPICK_DB_STATUS__ = await window.go.main.App.GetDatabaseStatus()
    } catch (e) {
      console.error('[PennyPick] 查询数据库状态失败', e)
    }
  }
  app.mount('#app')
}
bootstrap()
