<template>
  <div class="space-y-4">
    <div class="card">
      <div class="flex flex-wrap items-center gap-2 border-b border-slate-100 px-5 py-3">
        <input v-model="search" class="input max-w-xs" placeholder="搜索邮箱 / 用户名" @keyup.enter="reload" />
        <button class="btn-ghost" @click="reload">搜索</button>
        <span class="ml-auto text-sm text-slate-400">共 {{ total }} 人</span>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[820px]">
          <thead class="bg-slate-50"><tr>
            <th class="th">ID</th><th class="th">邮箱</th><th class="th">用户名</th><th class="th">角色</th>
            <th class="th">点数</th><th class="th">状态</th><th class="th">注册时间</th><th class="th">操作</th>
          </tr></thead>
          <tbody>
            <tr v-for="u in users" :key="u.id" class="border-t border-slate-50">
              <td class="td text-slate-400">{{ u.id }}</td>
              <td class="td">{{ u.email }}</td>
              <td class="td">{{ u.username }}</td>
              <td class="td">
                <span class="badge" :class="u.role === 'admin' ? 'bg-purple-50 text-purple-700' : 'bg-slate-100 text-slate-600'">{{ u.role }}</span>
              </td>
              <td class="td font-medium text-indigo-600">{{ u.points }}</td>
              <td class="td">
                <span class="badge" :class="u.status === 'active' ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-600'">
                  {{ u.status === 'active' ? '正常' : '禁用' }}
                </span>
              </td>
              <td class="td text-xs text-slate-400">{{ fmtTime(u.created_at) }}</td>
              <td class="td">
                <div class="flex gap-2 text-sm">
                  <button class="text-indigo-600 hover:underline" @click="openAdjust(u)">点数</button>
                  <button class="text-amber-600 hover:underline" @click="toggleStatus(u)">{{ u.status === 'active' ? '禁用' : '启用' }}</button>
                  <button class="text-slate-500 hover:underline" @click="resetPwd(u)">重置密码</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="flex items-center justify-between border-t border-slate-100 px-5 py-3 text-sm">
        <button class="btn-ghost !py-1.5" :disabled="page <= 1" @click="page--; load()">上一页</button>
        <span class="text-slate-400">第 {{ page }} 页</span>
        <button class="btn-ghost !py-1.5" :disabled="page * 20 >= total" @click="page++; load()">下一页</button>
      </div>
    </div>

    <div v-if="adjustTarget" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="adjustTarget = null">
      <div class="card w-full max-w-md p-6">
        <h3 class="font-bold">调整点数 · {{ adjustTarget.email }}</h3>
        <p class="mt-1 text-sm text-slate-500">当前余额 <b class="text-indigo-600">{{ adjustTarget.points }}</b> 点（正数为充值，负数为扣减）</p>
        <div class="mt-4 space-y-4">
          <div><label class="label">调整数额</label><input v-model.number="adjustForm.delta" type="number" step="0.01" class="input" placeholder="如 100 或 -50" /></div>
          <div><label class="label">备注</label><input v-model="adjustForm.note" class="input" placeholder="如：月度充值" /></div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="btn-ghost" @click="adjustTarget = null">取消</button>
          <button class="btn-primary" @click="submitAdjust">确认调整</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { get, post, put } from '../../api'
import { ok } from '../../stores/toast'

const users = ref([])
const total = ref(0)
const page = ref(1)
const search = ref('')
const adjustTarget = ref(null)
const adjustForm = ref({ delta: 0, note: '' })

async function load() {
  const data = await get('/api/admin/users', { page: page.value, size: 20, search: search.value })
  users.value = data.list || []
  total.value = data.total || 0
}
function reload() {
  page.value = 1
  load()
}

function openAdjust(u) {
  adjustTarget.value = u
  adjustForm.value = { delta: 0, note: '' }
}

async function submitAdjust() {
  await post(`/api/admin/users/${adjustTarget.value.id}/points`, adjustForm.value)
  ok('点数已调整')
  adjustTarget.value = null
  load()
}

async function toggleStatus(u) {
  await put(`/api/admin/users/${u.id}`, { status: u.status === 'active' ? 'disabled' : 'active' })
  load()
}

async function resetPwd(u) {
  const pwd = prompt(`为 ${u.email} 设置新密码（至少 6 位）：`)
  if (!pwd || pwd.length < 6) return
  await put(`/api/admin/users/${u.id}`, { password: pwd })
  ok('密码已重置')
}

function fmtTime(t) {
  return t ? new Date(t).toLocaleDateString('zh-CN') : '-'
}

onMounted(load)
</script>
