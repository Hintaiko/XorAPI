<template>
  <div class="space-y-6">
    <div class="grid gap-4 sm:grid-cols-3">
      <div class="card p-5">
        <div class="text-sm text-slate-400">总用户数</div>
        <div class="mt-1 text-3xl font-bold">{{ data.total_users ?? '-' }}</div>
      </div>
      <div class="card p-5">
        <div class="text-sm text-slate-400">总调用量</div>
        <div class="mt-1 text-3xl font-bold text-indigo-600">{{ data.total_calls ?? '-' }}</div>
      </div>
      <div class="card p-5">
        <div class="text-sm text-slate-400">总消耗点数</div>
        <div class="mt-1 text-3xl font-bold text-amber-600">{{ data.total_points ?? '-' }}</div>
      </div>
    </div>

    <div class="grid gap-4 lg:grid-cols-2">
      <div class="card p-5">
        <h3 class="mb-4 font-semibold">近 7 日调用量</h3>
        <div class="flex h-40 items-end gap-2">
          <div v-for="d in days" :key="d.date" class="flex flex-1 flex-col items-center gap-1">
            <div class="w-full rounded-t bg-indigo-500/80 transition-all" :style="{ height: barHeight(d.calls) + '%', minHeight: d.calls > 0 ? '4px' : '0' }"></div>
            <span class="text-[10px] text-slate-400">{{ d.date?.slice(5) }}</span>
            <span class="text-[10px] font-medium text-slate-600">{{ d.calls }}</span>
          </div>
          <div v-if="days.length === 0" class="w-full py-14 text-center text-sm text-slate-400">暂无数据</div>
        </div>
      </div>

      <div class="card p-5">
        <h3 class="mb-4 font-semibold">模型调用排行</h3>
        <div class="space-y-2.5">
          <div v-for="(r, i) in ranks" :key="r.model" class="flex items-center gap-3 text-sm">
            <span class="w-5 text-center font-bold" :class="i < 3 ? 'text-amber-500' : 'text-slate-400'">{{ i + 1 }}</span>
            <span class="w-48 truncate font-mono text-xs">{{ r.model }}</span>
            <div class="h-2 flex-1 overflow-hidden rounded-full bg-slate-100">
              <div class="h-full rounded-full bg-indigo-500" :style="{ width: rankWidth(r.calls) + '%' }"></div>
            </div>
            <span class="w-14 text-right text-slate-500">{{ r.calls }} 次</span>
          </div>
          <div v-if="ranks.length === 0" class="py-10 text-center text-sm text-slate-400">暂无数据</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { get } from '../../api'

const data = ref({})
const days = computed(() => fillDays(data.value.recent_days || []))
const ranks = computed(() => data.value.model_ranks || [])

const maxCalls = computed(() => Math.max(1, ...days.value.map((d) => d.calls)))
const maxRank = computed(() => Math.max(1, ...ranks.value.map((r) => r.calls)))
const barHeight = (n) => (n / maxCalls.value) * 100
const rankWidth = (n) => (n / maxRank.value) * 100

function fillDays(rows) {
  const map = Object.fromEntries(rows.map((r) => [String(r.date).slice(0, 10), r.calls]))
  const out = []
  for (let i = 6; i >= 0; i--) {
    const d = new Date(Date.now() - i * 86400000).toISOString().slice(0, 10)
    out.push({ date: d, calls: map[d] || 0 })
  }
  return out
}

onMounted(async () => {
  data.value = await get('/api/admin/dashboard')
})
</script>
