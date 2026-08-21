<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-bold">API Key 管理</h2>
      <button class="btn-primary" @click="showCreate = true">+ 创建 Key</button>
    </div>

    <div class="card overflow-x-auto">
      <table class="w-full min-w-[720px]">
        <thead class="bg-slate-50"><tr>
          <th class="th">名称</th><th class="th">Key</th><th class="th">状态</th>
          <th class="th">IP 白名单</th><th class="th">日限额</th><th class="th">调用次数</th><th class="th">操作</th>
        </tr></thead>
        <tbody>
          <tr v-for="k in keys" :key="k.id" class="border-t border-slate-50">
            <td class="td font-medium">{{ k.name }}</td>
            <td class="td font-mono text-xs">{{ k.key_preview }}</td>
            <td class="td">
              <span class="badge" :class="k.status === 'active' ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'">
                {{ k.status === 'active' ? '启用' : '禁用' }}
              </span>
            </td>
            <td class="td text-xs text-slate-400">{{ formatIP(k.ip_whitelist) }}</td>
            <td class="td">{{ k.daily_limit > 0 ? k.daily_limit + ' 次' : '不限' }}</td>
            <td class="td">{{ k.times }}</td>
            <td class="td">
              <div class="flex gap-2 text-sm">
                <button class="text-indigo-600 hover:underline" @click="toggle(k)">{{ k.status === 'active' ? '禁用' : '启用' }}</button>
                <button class="text-red-600 hover:underline" @click="remove(k)">删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="keys.length === 0"><td colspan="7" class="td py-10 text-center text-slate-400">还没有 API Key，点击右上角创建</td></tr>
        </tbody>
      </table>
    </div>

    <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="showCreate = false">
      <div class="card w-full max-w-md p-6">
        <h3 class="font-bold">创建 API Key</h3>
        <div class="mt-4 space-y-4">
          <div><label class="label">名称</label><input v-model="form.name" class="input" placeholder="如：我的 Cherry Studio" /></div>
          <div>
            <label class="label">IP 白名单（JSON 数组，留空不限）</label>
            <input v-model="form.ip_whitelist" class="input font-mono text-xs" placeholder='["1.2.3.4", "5.6.7.8"]' />
          </div>
          <div>
            <label class="label">每日调用上限（0 为不限）</label>
            <input v-model.number="form.daily_limit" type="number" min="0" class="input" />
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="btn-ghost" @click="showCreate = false">取消</button>
          <button class="btn-primary" @click="create">创建</button>
        </div>
      </div>
    </div>

    <div v-if="newKey" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div class="card w-full max-w-lg p-6">
        <h3 class="font-bold text-emerald-600">创建成功</h3>
        <p class="mt-2 text-sm text-slate-500">请立即复制保存，此 Key 只显示一次：</p>
        <div class="mt-3 flex items-center gap-2">
          <code class="flex-1 overflow-auto rounded-lg bg-slate-900 px-3 py-2.5 font-mono text-xs text-emerald-300">{{ newKey }}</code>
          <button class="btn-ghost" @click="copy">复制</button>
        </div>
        <div class="mt-5 text-right"><button class="btn-primary" @click="newKey = ''">我已保存</button></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { get, post, put, del } from '../api'
import { ok } from '../stores/toast'

const keys = ref([])
const showCreate = ref(false)
const newKey = ref('')
const form = reactive({ name: '', ip_whitelist: '', daily_limit: 0 })

function formatIP(s) {
  try {
    const arr = JSON.parse(s || '[]')
    return arr.length ? arr.join(', ') : '不限'
  } catch {
    return '不限'
  }
}

async function load() {
  keys.value = await get('/api/keys')
}

async function create() {
  if (!form.name.trim()) return
  let ipw = form.ip_whitelist.trim()
  if (ipw) {
    try {
      JSON.parse(ipw)
    } catch {
      ok('IP 白名单需为合法 JSON 数组')
      return
    }
  } else {
    ipw = ''
  }
  const data = await post('/api/keys', { ...form, ip_whitelist: ipw })
  newKey.value = data.key
  showCreate.value = false
  form.name = ''
  form.ip_whitelist = ''
  form.daily_limit = 0
  load()
}

async function toggle(k) {
  await put(`/api/keys/${k.id}`, { status: k.status === 'active' ? 'disabled' : 'active' })
  load()
}

async function remove(k) {
  if (!confirm(`确定删除 Key「${k.name}」？删除后立即失效。`)) return
  await del(`/api/keys/${k.id}`)
  ok('已删除')
  load()
}

function copy() {
  navigator.clipboard.writeText(newKey.value)
  ok('已复制到剪贴板')
}

onMounted(load)
</script>
