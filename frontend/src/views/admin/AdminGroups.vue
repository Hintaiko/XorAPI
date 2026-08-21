<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-bold">分组与渠道</h2>
        <p class="text-sm text-slate-400">分组按优先级路由（数字越小越优先），同名模型跨分组自动 fallback</p>
      </div>
      <button class="btn-primary" @click="openEdit(null)">+ 新建分组</button>
    </div>

    <div v-for="g in groups" :key="g.id" class="card">
      <div class="flex flex-wrap items-center gap-3 border-b border-slate-100 px-5 py-3">
        <span class="font-semibold">{{ g.name }}</span>
        <span class="badge bg-indigo-50 text-indigo-700">优先级 {{ g.priority }}</span>
        <span class="badge" :class="g.status === 'active' ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'">
          {{ g.status === 'active' ? '启用' : '停用' }}
        </span>
        <span class="text-xs text-slate-400">{{ g.description }}</span>
        <div class="ml-auto flex gap-2 text-sm">
          <button class="text-indigo-600 hover:underline" @click="openEdit(g)">编辑</button>
          <button class="text-red-600 hover:underline" @click="removeGroup(g)">删除</button>
        </div>
      </div>
      <table class="w-full">
        <thead class="bg-slate-50"><tr>
          <th class="th">渠道名</th><th class="th">协议</th><th class="th">Base URL</th><th class="th">优先级</th>
          <th class="th">状态</th><th class="th">连通性</th><th class="th">操作</th>
        </tr></thead>
        <tbody>
          <tr v-for="ch in g.channels" :key="ch.id" class="border-t border-slate-50">
            <td class="td">{{ ch.name }}</td>
            <td class="td"><span class="badge" :class="ch.protocol === 'anthropic' ? 'bg-orange-50 text-orange-700' : 'bg-emerald-50 text-emerald-700'">{{ ch.protocol }}</span></td>
            <td class="td font-mono text-xs text-slate-500">{{ ch.base_url }}</td>
            <td class="td">{{ ch.priority }}</td>
            <td class="td">
              <span class="badge" :class="ch.status === 'active' ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'">
                {{ ch.status === 'active' ? '启用' : '停用' }}
              </span>
            </td>
            <td class="td">
              <span v-if="ch.test_status === 'ok'" class="badge bg-emerald-50 text-emerald-700">正常</span>
              <span v-else-if="ch.test_status === 'fail'" class="badge bg-red-50 text-red-600" :title="ch.last_test_at">异常</span>
              <span v-else class="badge bg-slate-100 text-slate-500">未测</span>
            </td>
            <td class="td">
              <button class="text-sm text-indigo-600 hover:underline" :disabled="testing === ch.id" @click="test(ch)">
                {{ testing === ch.id ? '测试中...' : '测试' }}
              </button>
            </td>
          </tr>
          <tr v-if="!g.channels?.length"><td colspan="7" class="td py-4 text-center text-xs text-slate-400">暂无渠道</td></tr>
        </tbody>
      </table>
    </div>

    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="editing = null">
      <div class="card max-h-[85vh] w-full max-w-2xl overflow-auto p-6">
        <h3 class="font-bold">{{ editing.id ? '编辑分组' : '新建分组' }}</h3>
        <div class="mt-4 grid gap-4 sm:grid-cols-2">
          <div><label class="label">分组名称</label><input v-model="editing.name" class="input" placeholder="如 OpenAI组" /></div>
          <div><label class="label">优先级（小者优先）</label><input v-model.number="editing.priority" type="number" class="input" /></div>
          <div class="sm:col-span-2"><label class="label">描述</label><input v-model="editing.description" class="input" /></div>
          <div>
            <label class="label">状态</label>
            <select v-model="editing.status" class="input">
              <option value="active">启用</option><option value="disabled">停用</option>
            </select>
          </div>
        </div>

        <div class="mt-5">
          <div class="mb-2 flex items-center justify-between">
            <span class="text-sm font-semibold">渠道列表</span>
            <button class="btn-ghost !py-1 text-xs" @click="editing.channels.push({ id: 0, name: '', base_url: '', api_key: '', protocol: 'openai', priority: 0, status: 'active' })">+ 添加渠道</button>
          </div>
          <div v-for="(ch, i) in editing.channels" :key="i" class="mb-3 grid gap-2 rounded-lg border border-slate-200 p-3 sm:grid-cols-2">
            <div><label class="label !text-xs">名称</label><input v-model="ch.name" class="input !py-1.5" /></div>
            <div>
              <label class="label !text-xs">协议</label>
              <select v-model="ch.protocol" class="input !py-1.5"><option value="openai">OpenAI</option><option value="anthropic">Anthropic</option></select>
            </div>
            <div><label class="label !text-xs">Base URL（含 /v1）</label><input v-model="ch.base_url" class="input !py-1.5 font-mono !text-xs" placeholder="https://api.openai.com/v1" /></div>
            <div><label class="label !text-xs">API Key（留空则不修改）</label><input v-model="ch.api_key" type="password" class="input !py-1.5 font-mono !text-xs" placeholder="sk-..." /></div>
            <div><label class="label !text-xs">优先级</label><input v-model.number="ch.priority" type="number" class="input !py-1.5" /></div>
            <div class="flex items-end justify-between">
              <select v-model="ch.status" class="input !py-1.5"><option value="active">启用</option><option value="disabled">停用</option></select>
              <button class="text-xs text-red-600 hover:underline" @click="editing.channels.splice(i, 1)">移除</button>
            </div>
          </div>
        </div>

        <div class="mt-5 flex justify-end gap-3">
          <button class="btn-ghost" @click="editing = null">取消</button>
          <button class="btn-primary" @click="save">保存分组</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { get, post, del } from '../../api'
import { ok } from '../../stores/toast'

const groups = ref([])
const editing = ref(null)
const testing = ref(0)

async function load() {
  groups.value = await get('/api/admin/groups')
}

function openEdit(g) {
  if (g) {
    editing.value = {
      id: g.id, name: g.name, description: g.description, priority: g.priority, status: g.status,
      channels: (g.channels || []).map((c) => ({ ...c, api_key: '' })),
    }
  } else {
    editing.value = { id: 0, name: '', description: '', priority: 0, status: 'active', channels: [] }
  }
}

async function save() {
  const e = editing.value
  if (!e.name.trim()) return ok('请填写分组名称')
  for (const ch of e.channels) {
    if (!ch.id && !ch.api_key) return ok('新渠道必须填写 API Key')
  }
  await post('/api/admin/groups', e)
  ok('已保存')
  editing.value = null
  load()
}

async function removeGroup(g) {
  if (!confirm(`确定删除分组「${g.name}」？`)) return
  try {
    await del(`/api/admin/groups/${g.id}`)
    ok('已删除')
    load()
  } catch { /* 错误提示已由拦截器处理 */ }
}

async function test(ch) {
  testing.value = ch.id
  try {
    const r = await post(`/api/admin/channels/${ch.id}/test`)
    if (r.test_status === 'ok') ok('渠道连通正常')
    else ok('渠道异常：' + r.msg)
    load()
  } finally {
    testing.value = 0
  }
}

onMounted(load)
</script>
