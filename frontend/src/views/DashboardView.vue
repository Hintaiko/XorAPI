<template>
  <div class="space-y-6">
    <div class="card overflow-hidden">
      <div class="flex flex-col items-start justify-between gap-4 bg-gradient-to-r from-indigo-600 to-violet-600 p-6 text-white sm:flex-row sm:items-center">
        <div>
          <h2 class="text-lg font-bold">每日签到</h2>
          <p class="mt-1 text-sm text-indigo-100">
            <template v-if="signin.signed_today">今日已签到 · 连续 {{ signin.streak || 1 }} 天</template>
            <template v-else>连续签到奖励递增，签到可获得免费点数（有有效期，优先消耗）</template>
          </p>
        </div>
        <button class="btn bg-white !text-indigo-700 hover:bg-indigo-50" :disabled="signin.signed_today || !signin.enabled" @click="doSignin">
          {{ signin.signed_today ? '已签到 ✓' : '立即签到' }}
        </button>
      </div>
    </div>

    <div class="grid gap-4 sm:grid-cols-3">
      <div class="card p-5">
        <div class="text-sm text-slate-400">剩余点数</div>
        <div class="mt-1 text-3xl font-bold text-indigo-600">{{ (profile.user?.points ?? 0).toFixed(2) }}</div>
      </div>
      <div class="card p-5">
        <div class="text-sm text-slate-400">今日调用</div>
        <div class="mt-1 text-3xl font-bold">{{ profile.today_calls ?? 0 }}</div>
      </div>
      <div class="card p-5">
        <div class="text-sm text-slate-400">API Key 数量</div>
        <div class="mt-1 text-3xl font-bold">{{ keys.length }}</div>
      </div>
    </div>

    <div class="card p-6">
      <h3 class="font-semibold">快速开始</h3>
      <p class="mt-1 text-sm text-slate-500">创建 API Key 后，即可使用 OpenAI / Anthropic 兼容接口调用所有已开放模型。</p>
      <div class="mt-4 grid gap-3 text-sm sm:grid-cols-2">
        <div class="rounded-lg bg-slate-900 p-4 font-mono text-xs leading-relaxed text-slate-100 overflow-auto">
          <div class="mb-2 text-slate-400"># OpenAI 兼容</div>
          curl {{ origin }}/v1/chat/completions \\<br>
          &nbsp;&nbsp;-H "Authorization: Bearer sk-xxx" \\<br>
          &nbsp;&nbsp;-d '{"model": "gpt-4o", "messages": [...]}'
        </div>
        <div class="rounded-lg bg-slate-900 p-4 font-mono text-xs leading-relaxed text-slate-100 overflow-auto">
          <div class="mb-2 text-slate-400"># Anthropic 兼容</div>
          curl {{ origin }}/anthropic/v1/messages \\<br>
          &nbsp;&nbsp;-H "x-api-key: sk-xxx" \\<br>
          &nbsp;&nbsp;-d '{"model": "claude-sonnet-4", ...}'
        </div>
      </div>
    </div>

    <div class="card">
      <div class="flex items-center justify-between border-b border-slate-100 px-5 py-3">
        <h3 class="font-semibold">最近调用</h3>
        <router-link to="/dashboard/logs" class="text-sm text-indigo-600 hover:underline">查看全部</router-link>
      </div>
      <table class="w-full">
        <thead class="bg-slate-50"><tr>
          <th class="th">模型</th><th class="th">Tokens</th><th class="th">消耗</th><th class="th">状态</th><th class="th">时间</th>
        </tr></thead>
        <tbody>
          <tr v-for="log in recentLogs" :key="log.id" class="border-t border-slate-50">
            <td class="td font-mono text-xs">{{ log.model }}</td>
            <td class="td">{{ log.prompt_tokens }} + {{ log.completion_tokens }}</td>
            <td class="td">{{ log.points }}</td>
            <td class="td">
              <span class="badge" :class="log.status === 'success' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-600'">
                {{ log.status === 'success' ? '成功' : '失败' }}
              </span>
            </td>
            <td class="td text-slate-400">{{ fmtTime(log.created_at) }}</td>
          </tr>
          <tr v-if="recentLogs.length === 0"><td colspan="5" class="td py-8 text-center text-slate-400">暂无调用记录</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { get, post } from '../api'
import { ok } from '../stores/toast'
import { useUser } from '../stores/user'

const user = useUser()
const profile = reactive({})
const signin = reactive({ signed_today: false, streak: 0, enabled: true })
const keys = ref([])
const recentLogs = ref([])
const origin = computed(() => location.origin)

async function doSignin() {
  const data = await post('/api/user/signin')
  ok(`签到成功，获得 ${data.record.reward} 点（连续 ${data.record.streak} 天）`)
  signin.signed_today = true
  signin.streak = data.record.streak
  user.refresh()
  load()
}

function fmtTime(t) {
  return t ? new Date(t).toLocaleString('zh-CN', { hour12: false }) : '-'
}

async function load() {
  Object.assign(profile, await get('/api/user/profile'))
  Object.assign(signin, await get('/api/user/signin/status'))
  keys.value = await get('/api/keys')
  const logs = await get('/api/user/logs', { size: 8 })
  recentLogs.value = logs.list || []
}

onMounted(load)
</script>
