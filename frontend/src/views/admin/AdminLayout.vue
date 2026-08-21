<template>
  <div class="min-h-screen flex bg-slate-100">
    <aside class="hidden w-52 shrink-0 flex-col bg-slate-900 text-slate-300 md:flex">
      <div class="flex h-14 items-center gap-2 border-b border-slate-800 px-4">
        <span class="flex h-7 w-7 items-center justify-center rounded-md bg-indigo-600 text-xs font-bold text-white">X</span>
        <span class="font-bold text-white">管理后台</span>
      </div>
      <nav class="flex-1 space-y-1 p-3 text-sm">
        <router-link v-for="item in nav" :key="item.to" :to="item.to"
          class="flex items-center gap-2.5 rounded-lg px-3 py-2 hover:bg-slate-800 hover:text-white"
          :class="{ '!bg-indigo-600 !text-white': $route.path === item.to }">
          <span>{{ item.icon }}</span>{{ item.label }}
        </router-link>
      </nav>
      <div class="border-t border-slate-800 p-3">
        <router-link to="/dashboard" class="block rounded-lg px-3 py-2 text-sm hover:bg-slate-800 hover:text-white">← 用户控制台</router-link>
      </div>
    </aside>
    <div class="flex-1 min-w-0">
      <header class="flex h-14 items-center justify-between border-b border-slate-200 bg-white px-4 md:px-6">
        <div class="flex items-center gap-2 overflow-x-auto md:hidden">
          <router-link v-for="item in nav" :key="item.to" :to="item.to" class="whitespace-nowrap text-xs text-slate-600">{{ item.label }}</router-link>
        </div>
        <div class="ml-auto text-sm text-slate-500">{{ user.user?.email }}</div>
      </header>
      <main class="p-4 md:p-6"><router-view /></main>
    </div>
  </div>
</template>

<script setup>
import { useUser } from '../../stores/user'

const user = useUser()
const nav = [
  { to: '/admin', label: '数据看板', icon: '📊' },
  { to: '/admin/config', label: '系统设置', icon: '⚙️' },
  { to: '/admin/groups', label: '分组与渠道', icon: '🗂' },
  { to: '/admin/models', label: '模型管理', icon: '🧠' },
  { to: '/admin/users', label: '用户管理', icon: '👥' },
  { to: '/admin/templates', label: '模板管理', icon: '🎨' },
]
</script>
