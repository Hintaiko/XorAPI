<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-900 via-indigo-950 to-slate-900 flex items-center justify-center p-4">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <div class="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-indigo-600 text-xl font-bold text-white mb-3">X</div>
        <h1 class="text-2xl font-bold text-white">登录 XorAPI</h1>
        <p class="text-sm text-slate-400 mt-1">AI API 中转站</p>
      </div>
      <form class="card p-6 space-y-4" @submit.prevent="submit">
        <div>
          <label class="label">邮箱</label>
          <input v-model="email" type="email" required class="input" placeholder="you@example.com" />
        </div>
        <div>
          <label class="label">密码</label>
          <input v-model="password" type="password" required class="input" placeholder="••••••" />
        </div>
        <button class="btn-primary w-full" :disabled="loading" type="submit">
          {{ loading ? '登录中...' : '登 录' }}
        </button>
        <div class="flex justify-between text-sm">
          <router-link to="/register" class="text-indigo-600 hover:underline">注册账号</router-link>
          <router-link to="/" class="text-slate-500 hover:underline">返回首页</router-link>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUser } from '../stores/user'
import { ok } from '../stores/toast'

const email = ref('')
const password = ref('')
const loading = ref(false)
const user = useUser()
const router = useRouter()
const route = useRoute()

async function submit() {
  loading.value = true
  try {
    await user.login(email.value, password.value)
    ok('登录成功')
    router.push(route.query.redirect || (user.isAdmin ? '/admin' : '/dashboard'))
  } finally {
    loading.value = false
  }
}
</script>
