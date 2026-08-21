import axios from 'axios'
import { toast } from '../stores/toast'

export const http = axios.create({ baseURL: '', timeout: 30000 })

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (resp) => {
    if (resp.data && resp.data.code !== undefined && resp.data.code !== 0) {
      toast(resp.data.msg || '请求失败', 'error')
      return Promise.reject(new Error(resp.data.msg))
    }
    return resp
  },
  (err) => {
    const msg = err.response?.data?.msg || err.response?.data?.error?.message || err.message || '网络错误'
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      if (!location.pathname.startsWith('/login') && !location.pathname.startsWith('/install')) {
        location.href = '/login'
      }
    } else {
      toast(msg, 'error')
    }
    return Promise.reject(err)
  }
)

export async function api(method, url, data, params) {
  const resp = await http({ method, url, data, params })
  return resp.data?.data
}
export const get = (url, params) => api('get', url, null, params)
export const post = (url, data) => api('post', url, data)
export const put = (url, data) => api('put', url, data)
export const del = (url) => api('delete', url)
