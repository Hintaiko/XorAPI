<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-900 via-indigo-950 to-slate-900 flex items-center justify-center p-4">
    <div class="w-full max-w-md">
      <div class="text-center mb-8">
        <div class="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-indigo-600 text-xl font-bold text-white mb-3">X</div>
        <h1 class="text-2xl font-bold text-white">注册 XorAPI</h1>
        <p class="text-sm text-slate-400 mt-1">{{ status.site_name || 'XorAPI' }} · AI API 中转站</p>
      </div>
      <form class="card p-6 space-y-4" @submit.prevent="submit" v-if="status.registration_enabled !== false">
        <div>
          <label class="label">邮箱</label>
          <input v-model="form.email" type="email" required class="input" placeholder="you@example.com" />
        </div>
        <div>
          <label class="label">用户名</label>
          <input v-model="form.username" class="input" placeholder="留空则使用邮箱前缀" />
        </div>
        <div>
          <label class="label">密码（至少 6 位）</label>
          <input v-model="form.password" type="password" required minlength="6" class="input" placeholder="••••••" />
        </div>
        <div v-if="status.invite_required">
          <label class="label">邀请码 <span class="text-red-500">*</span></label>
          <input v-model="form.invite_code" required class="input" placeholder="请输入邀请码" />
        </div>
        <div v-if="status.email_verify">
          <label class="label">邮箱验证码</label>
          <div class="flex gap-2">
            <input v-model="form.email_code" class="input" placeholder="6 位验证码" />
            <button type="button" class="btn-ghost whitespace-nowrap" :disabled="sending || !form.email" @click="sendCode">
              {{ countdown > 0 ? countdown + 's' : '获取验证码' }}
            </button>
          </div>
        </div>
        <button class="btn-primary w-full" :disabled="loading" type="submit">
          {{ loading ? '注册中...' : '注 册' }}
        </button>
        <div class="flex justify-between text-sm">
          <router-link to="/login" class="text-indigo-600 hover:underline">已有账号？登录</router-link>
          <router-link to="/" class="text-slate-500 hover:underline">返回首页</router-link>
        </div>
      </form>
      <div v-else class="card p-8 text-center text-slate-500">注册已关闭，请联系管理员。</div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { get, post } from '../api'
import { ok } from '../stores/toast'
import { useUser } from '../stores/user'

const status = reactive({})
const form = reactive({ email: '', username: '', password: '', invite_code: '', email_code: '' })
const loading = ref(false)
const sending = ref(false)
const countdown = ref(0)
const router = useRouter()
const user = useUser()

async function sendCode() {
  sending.value = true
  try {
    await post('/api/auth/email-code', { email: form.email })
    ok('验证码已发送，请查收邮箱')
    countdown.value = 60
    const timer = setInterval(() => {
      if (--countdown.value <= 0) clearInterval(timer)
    }, 1000)
  } finally {
    sending.value = false
  }
}

async function submit() {
  loading.value = true
  try {
    const data = await post('/api/auth/register', form)
    user.setAuth(data.token, data.user)
    ok('注册成功')
    router.push('/dashboard')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    Object.assign(status, await get('/api/status'))
  } catch { /* ignore */ }
})
</script>
