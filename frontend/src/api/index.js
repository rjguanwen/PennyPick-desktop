import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

// 桌面版 API 基础地址由解锁成功后注入 __PENNYPICK_API_BASE__（Wails 绑定获取）；
// 需在请求时动态读取，因为解锁发生在模块加载之后。Web 开发模式回退到 /api。
const api = axios.create({
  timeout: 20000,
})

api.interceptors.request.use((config) => {
  config.baseURL = window.__PENNYPICK_API_BASE__ || '/api'
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const status = error.response?.status
    if (status === 401) {
      const url = error.config?.url || ''
      // 登录接口失败（用户名/密码错误）必须提示；其余 401 属预期行为
      // （未登录/凭证过期，桌面版每次启动都会强制重新登录），静默处理不弹错误提示
      if (url.includes('/auth/login')) {
        const detail = error.response?.data?.detail
        ElMessage.error(typeof detail === 'string' ? detail : '用户名或密码错误')
      } else {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        if (router.currentRoute.value.path !== '/login') {
          router.push('/login')
        }
      }
      return Promise.reject(error)
    }
    const detail = error.response?.data?.detail
    const msg = typeof detail === 'string' ? detail : error.message || '请求失败'
    ElMessage.error(msg)
    return Promise.reject(error)
  },
)

export default api

// ===== 认证 =====
export const authApi = {
  login: (username, password) => {
    const form = new URLSearchParams()
    form.append('username', username)
    form.append('password', password)
    return api.post('/auth/login', form)
  },
  register: (data) => api.post('/auth/register', data),
  me: () => api.get('/auth/me'),
  changePassword: (data) => api.put('/auth/password', data),
}

// ===== 分类 =====
export const categoryApi = {
  list: (type) => api.get('/categories', { params: { type } }),
  create: (data) => api.post('/categories', data),
  update: (id, data) => api.patch(`/categories/${id}`, data),
  remove: (id) => api.delete(`/categories/${id}`),
}

// ===== 账户 =====
export const accountApi = {
  list: () => api.get('/accounts'),
  create: (data) => api.post('/accounts', data),
  update: (id, data) => api.patch(`/accounts/${id}`, data),
  remove: (id) => api.delete(`/accounts/${id}`),
}

// ===== 还款 =====
export const repaymentApi = {
  list: (month) => api.get('/repayments', { params: { month } }),
  bills: (month, accountId) => api.get('/repayments/bills', { params: { month, account_id: accountId } }),
  mark: (data) => api.post('/repayments', data),
  unmark: (month, accountId) => api.delete('/repayments', { params: { month, account_id: accountId } }),
}

// ===== 标签 =====
export const tagApi = {
  list: () => api.get('/tags'),
  create: (data) => api.post('/tags', data),
  update: (id, data) => api.patch(`/tags/${id}`, data),
  remove: (id) => api.delete(`/tags/${id}`),
}

// ===== 账单 =====
export const billApi = {
  list: (params) => api.get('/bills', { params }),
  create: (data) => api.post('/bills', data),
  createBatch: (data) => api.post('/bills/batch', data),
  update: (id, data) => api.patch(`/bills/${id}`, data),
  remove: (id) => api.delete(`/bills/${id}`),
}

// ===== 预算 =====
export const budgetApi = {
  // 总预算
  get: (month) => api.get('/budgets', { params: { month } }),
  all: () => api.get('/budgets/all'),
  upsert: (data) => api.put('/budgets', data),
  remove: (month) => api.delete('/budgets', { params: { month } }),
  // 分类预算
  categories: (month) => api.get('/budgets/categories', { params: { month } }),
  upsertCategory: (data) => api.put('/budgets/category', data),
  removeCategory: (month, categoryId) =>
    api.delete('/budgets/category', { params: { month, category_id: categoryId } }),
}

// ===== 固定账单 =====
export const recurringBillApi = {
  list: () => api.get('/recurring-bills'),
  create: (data) => api.post('/recurring-bills', data),
  update: (id, data) => api.patch(`/recurring-bills/${id}`, data),
  remove: (id) => api.delete(`/recurring-bills/${id}`),
  apply: (data) => api.post('/recurring-bills/apply', data),
}

// ===== 收支报告（月度 + 年度）=====
export const reportsApi = {
  list: () => api.get('/reports'),
  generate: (data) => api.post('/reports/generate', data),
  detail: (id) => api.get(`/reports/${id}`),
  yearlyGenerate: (data) => api.post('/reports/yearly', data),
  yearlyList: () => api.get('/reports/yearly/list'),
  yearlyDetail: (id) => api.get(`/reports/yearly/${id}`),
}

// ===== 统计 =====
export const statsApi = {
  overview: (month) => api.get('/stats/overview', { params: { month } }),
  byCategory: (params) => api.get('/stats/by-category', { params }),
  trend: (params) => api.get('/stats/trend', { params }),
  accounts: (month) => api.get('/stats/accounts', { params: { month } }),
  tags: (params) => api.get('/stats/tags', { params }),
}

// ===== 导出 =====
export const exportApi = {
  // 返回 Blob，前端触发下载
  download: (params) =>
    api.get('/export', { params, responseType: 'blob' }),
}

// ===== 账单导入 =====
export const billImportApi = {
  // 上传账单文件并解析（预览）
  parse: (file, platform) => {
    const form = new FormData()
    form.append('file', file)
    form.append('platform', platform)
    return api.post('/bill-import/parse', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
    })
  },
  // 确认导入
  confirm: (data) => api.post('/bill-import/confirm', data),
  // 导入历史
  history: (page = 1, pageSize = 10) => api.get('/bill-import/history', { params: { page, page_size: pageSize } }),
  // 导入明细
  detail: (id) => api.get(`/bill-import/${id}`),
}
