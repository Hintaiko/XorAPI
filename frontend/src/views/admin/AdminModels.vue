<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-bold">模型管理</h2>
        <p class="text-sm text-slate-400">每个模型归属一个分组；同名模型存在于多个分组时自动跨组 fallback</p>
      </div>
      <button class="btn-primary" @click="openEdit(null)">+ 新建模型</button>
    </div>

    <div class="card overflow-x-auto">
      <table class="w-full min-w-[900px]">
        <thead class="bg-slate-50"><tr>
          <th class="th">模型名</th><th class="th">显示名</th><th class="th">分组</th><th class="th">计费</th>
          <th class="th">单价</th><th class="th">广场展示</th><th class="th">允许调用</th><th class="th">操作</th>
        </tr></thead>
        <tbody>
          <tr v-for="m in models" :key="m.id" class="border-t border-slate-50">
            <td class="td font-mono text-xs">{{ m.name }}</td>
            <td class="td">{{ m.display_name || '-' }}</td>
            <td class="td"><span class="badge bg-indigo-50 text-indigo-700">{{ m.group_name }}</span></td>
            <td class="td">{{ m.billing_type === 'token' ? '按 Token' : '按次' }}</td>
            <td class="td text-xs">
              <template v-if="m.billing_type === 'token'">入 {{ m.input_price }} / 出 {{ m.output_price }} 点/百万tok</template>
              <template v-else>{{ m.per_call_price }} 点/次</template>
            </td>
            <td class="td">
              <button class="badge" :class="m.visible ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'" @click="quickToggle(m, 'visible')">
                {{ m.visible ? '展示中' : '隐藏' }}
              </button>
            </td>
            <td class="td">
              <button class="badge" :class="m.callable ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-600'" @click="quickToggle(m, 'callable')">
                {{ m.callable ? '允许' : '禁止' }}
              </button>
            </td>
            <td class="td">
              <div class="flex gap-2 text-sm">
                <button class="text-indigo-600 hover:underline" @click="openEdit(m)">编辑</button>
                <button class="text-red-600 hover:underline" @click="remove(m)">删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="models.length === 0"><td colspan="8" class="td py-10 text-center text-slate-400">暂无模型，请先创建分组再添加模型</td></tr>
        </tbody>
      </table>
    </div>

    <div v-if="editing" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="editing = null">
      <div class="card max-h-[85vh] w-full max-w-xl overflow-auto p-6">
        <h3 class="font-bold">{{ editing.id ? '编辑模型' : '新建模型' }}</h3>
        <div class="mt-4 grid gap-4 sm:grid-cols-2">
          <div><label class="label">模型名（调用时的 model 参数）</label><input v-model="editing.name" class="input font-mono text-xs" placeholder="gpt-4o" /></div>
          <div><label class="label">显示名</label><input v-model="editing.display_name" class="input" placeholder="GPT-4o" /></div>
          <div>
            <label class="label">所属分组</label>
            <select v-model.number="editing.group_id" class="input">
              <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
            </select>
          </div>
          <div>
            <label class="label">计费方式</label>
            <select v-model="editing.billing_type" class="input">
              <option value="token">按 Token</option><option value="per_call">按次</option>
            </select>
          </div>
          <template v-if="editing.billing_type === 'token'">
            <div><label class="label">输入单价（点/百万 Token）</label><input v-model.number="editing.input_price" type="number" step="0.01" class="input" /></div>
            <div><label class="label">输出单价（点/百万 Token）</label><input v-model.number="editing.output_price" type="number" step="0.01" class="input" /></div>
          </template>
          <div v-else><label class="label">每次调用价格（点）</label><input v-model.number="editing.per_call_price" type="number" step="0.01" class="input" /></div>
          <div><label class="label">标签（逗号分隔）</label><input v-model="editing.tags" class="input" placeholder="对话,视觉" /></div>
          <div class="sm:col-span-2"><label class="label">描述（模型广场展示）</label><textarea v-model="editing.description" rows="3" class="input"></textarea></div>
          <div class="flex items-center gap-6 sm:col-span-2">
            <label class="flex items-center gap-2 text-sm"><input v-model="editing.visible" type="checkbox" class="h-4 w-4" />广场展示</label>
            <label class="flex items-center gap-2 text-sm"><input v-model="editing.callable" type="checkbox" class="h-4 w-4" />允许调用</label>
          </div>
        </div>
        <div class="mt-5 flex justify-end gap-3">
          <button class="btn-ghost" @click="editing = null">取消</button>
          <button class="btn-primary" @click="save">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { get, post, del } from '../../api'
import { ok } from '../../stores/toast'

const models = ref([])
const groups = ref([])
const editing = ref(null)

async function load() {
  const data = await get('/api/admin/models')
  models.value = data.list || []
  groups.value = data.groups || []
}

function openEdit(m) {
  editing.value = m
    ? { ...m }
    : { id: 0, name: '', display_name: '', group_id: groups.value[0]?.id || 0, description: '', tags: '',
        billing_type: 'token', input_price: 0, output_price: 0, per_call_price: 0, visible: true, callable: true }
}

async function save() {
  if (!editing.value.name || !editing.value.group_id) return ok('模型名与分组必填')
  await post('/api/admin/models', editing.value)
  ok('已保存')
  editing.value = null
  load()
}

async function quickToggle(m, field) {
  await post('/api/admin/models', { ...m, [field]: !m[field] })
  load()
}

async function remove(m) {
  if (!confirm(`确定删除模型「${m.name}」？`)) return
  await del(`/api/admin/models/${m.id}`)
  ok('已删除')
  load()
}

onMounted(load)
</script>
