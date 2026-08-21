<template>
  <div class="min-h-screen">
    <header class="border-b border-slate-200 bg-white/80 backdrop-blur sticky top-0 z-10">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <router-link to="/" class="flex items-center gap-2">
          <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 text-sm font-bold text-white">X</span>
          <span class="font-bold">{{ status.site_name || 'XorAPI' }}</span>
        </router-link>
        <nav class="flex items-center gap-4 text-sm">
          <template v-if="user.loggedIn">
            <router-link v-if="user.isAdmin" to="/admin" class="text-indigo-600 hover:underline">管理后台</router-link>
            <router-link to="/dashboard" class="btn-primary !py-1.5">控制台</router-link>
          </template>
          <template v-else>
            <router-link to="/login" class="text-slate-600 hover:text-indigo-600">登录</router-link>
            <router-link to="/register" class="btn-primary !py-1.5">注册</router-link>
          </template>
        </nav>
      </div>
    </header>

    <section class="bg-gradient-to-b from-indigo-50 to-transparent py-12 text-center">
      <h1 class="text-3xl font-bold">模型广场</h1>
      <p class="mt-2 text-slate-500">{{ list.exchange_note || '一个 Key，调用所有大模型' }}</p>
      <div class="mx-auto mt-6 flex max-w-xl gap-2">
        <input v-model="search" class="input" placeholder="搜索模型名称 / 标签，如 gpt、claude..." @keyup.enter="load" />
        <button class="btn-primary" @click="load">搜索</button>
      </div>
    </section>

    <main class="mx-auto max-w-6xl px-4 pb-16">
      <div v-if="list.announcement" class="mb-6 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
        📢 {{ list.announcement }}
      </div>

      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="m in models" :key="m.id" class="card p-5 hover:shadow-md transition cursor-pointer" @click="openDetail(m)">
          <div class="flex items-start justify-between gap-2">
            <div>
              <div class="font-semibold">{{ m.display_name || m.name }}</div>
              <div class="mt-0.5 font-mono text-xs text-slate-400">{{ m.name }}</div>
            </div>
            <span class="badge" :class="m.callable ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'">
              {{ m.callable ? '可调用' : '维护中' }}
            </span>
          </div>
          <p class="mt-3 line-clamp-2 text-sm text-slate-500">{{ m.description || '暂无描述' }}</p>
          <div class="mt-4 flex flex-wrap gap-1.5">
            <span class="badge bg-indigo-50 text-indigo-700">{{ m.group_name }}</span>
            <span v-for="t in tags(m.tags)" :key="t" class="badge bg-slate-100 text-slate-600">{{ t }}</span>
          </div>
          <div class="mt-4 flex items-center justify-between text-xs text-slate-400">
            <span v-if="m.billing_type === 'token'">输入 ¥{{ m.input_price }}/百万tok · 输出 ¥{{ m.output_price }}/百万tok</span>
            <span v-else>{{ m.per_call_price }} 点/次</span>
            <span class="text-indigo-600">详情 →</span>
          </div>
        </div>
      </div>
      <div v-if="!loading && models.length === 0" class="py-16 text-center text-slate-400">暂无公开模型</div>
    </main>

    <div v-if="detail" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="detail = null">
      <div class="card max-h-[85vh] w-full max-w-2xl overflow-auto p-6">
        <div class="flex items-start justify-between">
          <div>
            <h2 class="text-lg font-bold">{{ detail.display_name || detail.name }}</h2>
            <p class="font-mono text-xs text-slate-400">{{ detail.name }}</p>
          </div>
          <button class="text-slate-400 hover:text-slate-600" @click="detail = null">✕</button>
        </div>
        <p class="mt-3 text-sm text-slate-600">{{ detail.description || '暂无描述' }}</p>
        <div class="mt-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          <div class="rounded-lg bg-slate-50 p-3"><div class="text-xs text-slate-400">分组</div>{{ detail.group_name }}</div>
          <div class="rounded-lg bg-slate-50 p-3"><div class="text-xs text-slate-400">计费</div>{{ detail.billing_type === 'token' ? '按 Token' : '按次' }}</div>
          <div class="rounded-lg bg-slate-50 p-3"><div class="text-xs text-slate-400">输入价</div>{{ detail.input_price }}/百万tok</div>
          <div class="rounded-lg bg-slate-50 p-3"><div class="text-xs text-slate-400">输出价</div>{{ detail.output_price }}/百万tok</div>
        </div>
        <h3 class="mt-6 mb-2 text-sm font-semibold">示例代码</h3>
        <div class="flex gap-2 mb-2 text-xs">
          <button v-for="(v, k) in snippets" :key="k" class="rounded-md px-2.5 py-1"
            :class="activeSnippet === k ? 'bg-indigo-600 text-white' : 'bg-slate-100 text-slate-600'" @click="activeSnippet = k">{{ k }}</button>
        </div>
        <pre class="overflow-auto rounded-lg bg-slate-900 p-4 text-xs leading-relaxed text-slate-100">{{ snippets[activeSnippet] }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { get } from '../api'
import { useUser } from '../stores/user'

const user = useUser()
const models = ref([])
const loading = ref(true)
const search = ref('')
const detail = ref(null)
const activeSnippet = ref('curl')
const status = ref({})
const list = ref({})

const snippets = computed(() => {
  const m = detail.value?.name || 'model'
  const base = location.origin
  return {
    curl: `curl ${base}/v1/chat/completions \\
  -H "Authorization: Bearer sk-你的APIKey" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${m}",
    "messages": [{"role": "user", "content": "你好"}]
  }'`,
    python: `from openai import OpenAI

client = OpenAI(
    base_url="${base}/v1",
    api_key="sk-你的APIKey",
)

resp = client.chat.completions.create(
    model="${m}",
    messages=[{"role": "user", "content": "你好"}],
)
print(resp.choices[0].message.content)`,
    'node.js': `import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "${base}/v1",
  apiKey: "sk-你的APIKey",
});

const resp = await client.chat.completions.create({
  model: "${m}",
  messages: [{ role: "user", content: "你好" }],
});
console.log(resp.choices[0].message.content);`,
    anthropic: `curl ${base}/anthropic/v1/messages \\
  -H "x-api-key: sk-你的APIKey" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${m}",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "你好"}]
  }'`,
  }
})

function tags(s) {
  return (s || '').split(',').map((t) => t.trim()).filter(Boolean)
}

function openDetail(m) {
  detail.value = m
  activeSnippet.value = 'curl'
}

async function load() {
  loading.value = true
  try {
    const data = await get('/api/square/models', { search: search.value })
    models.value = data.list || []
    list.value = data
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  load()
  try {
    status.value = await get('/api/status')
  } catch { /* ignore */ }
})
</script>
