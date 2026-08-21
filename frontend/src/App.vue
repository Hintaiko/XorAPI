<template>
  <router-view />
  <div class="fixed top-4 right-4 z-[100] flex flex-col gap-2">
    <div
      v-for="t in toasts"
      :key="t.id"
      class="rounded-lg px-4 py-2 text-sm text-white shadow-lg animate-[fadeIn_.2s]"
      :class="{
        'bg-emerald-600': t.type === 'success',
        'bg-red-600': t.type === 'error',
        'bg-slate-800': t.type === 'info',
      }"
    >
      {{ t.msg }}
    </div>
  </div>
</template>

<script setup>
import { toasts } from './stores/toast'
</script>

<style>
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-6px); }
  to { opacity: 1; transform: none; }
}
</style>
