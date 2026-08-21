<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-bold">前端模板</h2>
        <p class="text-sm text-slate-400">上传 ZIP 模板包一键切换站点外观；模板仅负责展示，API 由后端统一处理</p>
      </div>
      <label class="btn-primary cursor-pointer">
        上传模板 ZIP
        <input type="file" accept=".zip" class="hidden" @change="upload" />
      </label>
    </div>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <div class="card p-5" :class="{ 'ring-2 ring-indigo-500': active === 'default' }">
        <div class="flex h-28 items-center justify-center rounded-lg bg-gradient-to-br from-indigo-500 to-violet-600 text-white">
          <span class="text-lg font-bold">默认模板</span>
        </div>
        <div class="mt-3 flex items-center justify-between">
          <div>
            <div class="font-semibold">default</div>
            <div class="text-xs text-slate-400">内置 Vue 3 模板，随程序更新</div>
          </div>
          <span v-if="active === 'default'" class="badge bg-emerald-50 text-emerald-700">使用中</span>
          <button v-else class="btn-ghost !py-1 text-xs" @click="activate('default')">启用</button>
        </div>
      </div>

      <div v-for="t in list" :key="t.slug" class="card p-5" :class="{ 'ring-2 ring-indigo-500': active === t.slug }">
        <div class="flex h-28 items-center justify-center rounded-lg bg-slate-100">
          <img v-if="t.preview" :src="t.preview" class="h-full w-full rounded-lg object-cover" />
          <span v-else class="text-4xl">🎨</span>
        </div>
        <div class="mt-3 flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="font-semibold">{{ t.name }}</div>
            <div class="truncate text-xs text-slate-400">v{{ t.version || '1.0.0' }} · {{ t.author || '未知作者' }}</div>
            <div class="mt-1 line-clamp-2 text-xs text-slate-500">{{ t.description }}</div>
          </div>
          <span v-if="active === t.slug" class="badge shrink-0 bg-emerald-50 text-emerald-700">使用中</span>
          <button v-else class="btn-ghost shrink-0 !py-1 text-xs" @click="activate(t.slug)">启用</button>
        </div>
      </div>
    </div>

    <div class="card p-6">
      <h3 class="font-semibold">模板开发说明</h3>
      <p class="mt-2 text-sm text-slate-500">ZIP 包根目录需包含 manifest.json 与 index.html：</p>
      <pre class="mt-3 overflow-auto rounded-lg bg-slate-900 p-4 text-xs leading-relaxed text-slate-100">my-template.zip
├── manifest.json
├── index.html
└── assets/...

// manifest.json
{
  "name": "我的模板",
  "slug": "my-template",      // 可选，安装目录名
  "version": "1.0.0",
  "author": "yourname",
  "description": "一个简洁的模板"
}</pre>
      <p class="mt-3 text-sm text-slate-500">
        模板页面可通过同源接口获取数据：<code class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs">GET /api/square/models</code>（模型广场）、
        <code class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-xs">GET /api/status</code>（站点状态）。
        路由模式请使用 hash（#）以兼容任意路径刷新。激活后站点首页将渲染该模板，管理后台不受影响。
      </p>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { get, post, put } from '../../api'
import { ok } from '../../stores/toast'

const list = ref([])
const active = ref('default')

async function load() {
  const data = await get('/api/admin/templates')
  list.value = data.list || []
  active.value = data.active || 'default'
}

async function upload(e) {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  const data = await post('/api/admin/templates/upload', fd)
  ok(`模板「${data.name}」上传成功`)
  load()
  e.target.value = ''
}

async function activate(slug) {
  await put('/api/admin/templates/activate', { slug })
  ok('模板已切换')
  load()
}

onMounted(load)
</script>
