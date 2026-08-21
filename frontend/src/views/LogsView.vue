<template>
  <div class="space-y-4">
    <div class="card">
      <div class="flex flex-wrap items-center gap-2 border-b border-slate-100 px-5 py-3">
        <button v-for="t in tabs" :key="t.key" class="rounded-lg px-3 py-1.5 text-sm"
          :class="tab === t.key ? 'bg-indigo-600 text-white' : 'text-slate-600 hover:bg-slate-50'" @click="switchTab(t.key)">
          {{ t.label }}
        </button>
        <div class="ml-auto text-sm text-slate-400">共 {{ total }} 条</div>
      </div>
      <div class="overflow-x-auto">
        <table v-if="tab === 'logs'" class="w-full min-w-[760px]">
          <thead class="bg-slate-50"><tr>
            <th class="th">模型</th><th class="th">Tokens(入/出)</th><th class="th">消耗点数</th>
            <th class="th">耗时</th><th class="th">状态</th><th class="th">IP</th><th class="th">时间</th>
          </tr></thead>
          <tbody>
            <tr v-for="row in rows" :key="row.id" class="border-t border-slate-50">
              <td class="td font-mono text-xs">{{ row.model }}</td>
              <td class="td">{{ row.prompt_tokens }} / {{ row.completion_tokens }}</td>
              <td class="td text-indigo-600">{{ row.points }}</td>
              <td class="td">{{ row.latency_ms }}ms</td>
              <td class="td">
                <span class="badge" :class="row.status === 'success' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-600'">
                  {{ row.status === 'success' ? '成功' : '失败' }}
                </span>
              </td>
              <td class="td text-xs text-slate-400">{{ row.ip }}</td>
              <td class="td text-xs text-slate-400">{{ fmtTime(row.created_at) }}</td>
            </tr>
          </tbody>
        </table>
        <table v-else class="w-full min-w-[640px]">
          <thead class="bg-slate-50"><tr>
            <th class="th">类型</th><th class="th">变动</th><th class="th">余额</th><th class="th">说明</th><th class="th">时间</th>
          </tr></thead>
          <tbody>
            <tr v-for="row in rows" :key="row.id" class="border-t border-slate-50">
              <td class="td">
                <span class="badge" :class="typeClass(row.type)">{{ typeLabel(row.type) }}</span>
              </td>
              <td class="td" :class="row.amount >= 0 ? 'text-emerald-600' : 'text-red-600'">
                {{ row.amount >= 0 ? '+' : '' }}{{ row.amount }}
              </td>
              <td class="td">{{ row.balance_after }}</td>
              <td class="td text-xs text-slate-500">{{ row.detail || row.model || '-' }}</td>
              <td class="td text-xs text-slate-400">{{ fmtTime(row.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="flex items-center justify-between border-t border-slate-100 px-5 py-3 text-sm">
        <button class="btn-ghost !py-1.5" :disabled="page <= 1" @click="page--; load()">上一页</button>
        <span class="text-slate-400">第 {{ page }} 页</span>
        <button class="btn-ghost !py-1.5" :disabled="page * size >= total" @click="page++; load()">下一页</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { get } from '../api'

const tabs = [
  { key: 'logs', label: '调用记录' },
  { key: 'txns', label: '点数流水' },
]
const tab = ref('logs')
const rows = ref([])
const total = ref(0)
const page = ref(1)
const size = 20

function switchTab(key) {
  tab.value = key
  page.value = 1
  load()
}

async function load() {
  const url = tab.value === 'logs' ? '/api/user/logs' : '/api/user/transactions'
  const data = await get(url, { page: page.value, size })
  rows.value = data.list || []
  total.value = data.total || 0
}

function fmtTime(t) {
  return t ? new Date(t).toLocaleString('zh-CN', { hour12: false }) : '-'
}

function typeLabel(t) {
  return { consume: '消费', recharge: '充值', signin: '签到', adjust: '调整', refund: '退款' }[t] || t
}
function typeClass(t) {
  return {
    consume: 'bg-red-50 text-red-600',
    recharge: 'bg-emerald-50 text-emerald-700',
    signin: 'bg-indigo-50 text-indigo-700',
    adjust: 'bg-amber-50 text-amber-700',
  }[t] || 'bg-slate-100 text-slate-600'
}

onMounted(load)
</script>
