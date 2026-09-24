import axios from 'axios'
import { useAuthStore } from './stores/auth'
import router from './router'

const http = axios.create({
  baseURL: '/api',
  timeout: 15000,
})

http.interceptors.request.use((config) => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => res,
  (err) => {
    const msg = err.response?.data?.error || err.message || '请求失败'
    if (err.response?.status === 401 || err.response?.status === 403) {
      const auth = useAuthStore()
      if (err.response?.status === 401 || msg.includes('禁用')) {
        auth.logout()
        router.push('/login')
      }
    }
    return Promise.reject(new Error(msg))
  }
)

export default http

export function fenToYuan(fen) {
  return (Number(fen || 0) / 100).toFixed(2)
}

export const statusMap = {
  pending: { label: '待接单', type: 'warning' },
  accepted: { label: '已接单', type: 'primary' },
  submitted: { label: '待验收', type: 'info' },
  approved: { label: '已通过', type: 'success' },
  rejected: { label: '已退回', type: 'danger' },
  cancelled: { label: '已取消', type: '' },
}

export const withdrawStatusMap = {
  pending: { label: '待审核', type: 'warning' },
  approved: { label: '已通过', type: 'success' },
  rejected: { label: '已拒绝', type: 'danger' },
}

