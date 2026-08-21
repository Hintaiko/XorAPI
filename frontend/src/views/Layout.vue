<template>
  <div class="min-h-screen flex">
    <aside class="hidden w-56 shrink-0 border-r border-slate-200 bg-white md:block">
      <div class="flex h-14 items-center gap-2 border-b border-slate-100 px-4">
        <span class="flex h-7 w-7 items-center justify-center rounded-md bg-indigo-600 text-xs font-bold text-white">X</span>
        <span class="font-bold">XorAPI</span>
      </div>
      <nav class="space-y-1 p-3 text-sm">
        <router-link v-for="item in nav" :key="item.to" :to="item.to"
          class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-slate-600 hover:bg-slate-50 hover:text-indigo-600"
          :class="{ '!bg-indigo-50 !text-indigo-600 font-medium': $route.path === item.to }">
          <span>{{ item.icon }}</span>{{ item.label }}
        </router-link>
        <router-link v-if="user.isAdmin" to="/admin"
          class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-slate-600 hover:bg-slate-50 hover:text-indigo-600">
          <span>🛠</span>管理后台
        </router-link>
        <router-link to="/" class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-slate-600 hover:bg-slate-50">
          <span>🌐</span>模型广场
        </router-link>
      </nav>
      <div class="absolute bottom-0 w-56 border-t border-slate-100 p-3"></div>
    </aside>

    <div class="flex-1 flex flex-col min-w-0">
      <header class="flex h-14 items-center justify-between border-b border-slate-200 bg-white px-4 md:px-6">
        <div class="flex items-center gap-3 md:hidden">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-indigo-600 text-xs font-bold text-white">X</span>
          <span class="font-bold">XorAPI</span>
        </div>
        <div class="ml-auto flex items-center gap-3 text-sm">
          <span class="badge bg-indigo-50 text-indigo-700">{{ (user.user?.points ?? 0).toFixed(2) }} 点</span>
          <span class="text-slate-600">{{ user.user?.username || user.user?.email }}</span>
          <button class="text-slate-400 hover:text-red-600" @click="logout" title="退出登录">退出</button>
        </div>
      </header>
      <main class="flex-1 p-4 md:p-6">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUser } from '../stores/user'

const user = useUser()
const router = useRouter()

const nav = [
  { to: '/dashboard', label: '总览', icon: '📊' },
  { to: '/dashboard/keys', label: 'API Key', icon: '🔑' },
  { to: '/dashboard/logs', label: '调用记录', icon: '📜' },
  { to: '/dashboard/profile', label: '个人信息', icon: '👤' },
]

function logout() {
  user.logout()
  router.push('/login')
}

onMounted(() => user.refresh())
</script>
