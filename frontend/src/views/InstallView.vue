<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-900 via-indigo-950 to-slate-900 flex items-center justify-center p-4">
    <div class="w-full max-w-2xl">
      <div class="text-center mb-8">
        <div class="inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-600 text-2xl font-bold text-white mb-3">X</div>
        <h1 class="text-2xl font-bold text-white">XorAPI 安装向导</h1>
        <p class="text-sm text-slate-400 mt-1">AI API 中转站 · 只需三步完成部署</p>
      </div>

      <div class="mb-6 flex items-center justify-center gap-2 text-xs">
        <template v-for="(label, i) in steps" :key="i">
          <div class="flex items-center gap-1.5 rounded-full px-3 py-1.5"
            :class="step === i ? 'bg-indigo-600 text-white' : step > i ? 'bg-emerald-600/20 text-emerald-400' : 'bg-white/10 text-slate-400'">
            <span class="flex h-5 w-5 items-center justify-center rounded-full text-[11px]"
              :class="step === i ? 'bg-white/20' : step > i ? 'bg-emerald-500 text-white' : 'bg-white/10'">
              {{ step > i ? '✓' : i + 1 }}
            </span>{{ label }}
          </div>
          <div v-if="i < steps.length - 1" class="h-px w-6 bg-white/15"></div>
        </template>
      </div>

      <div class="card p-6">
        <div v-if="installed" class="text-center py-8">
          <div class="text-4xl mb-3">🔒</div>
          <h2 class="text-lg font-semibold">系统已安装</h2>
          <p class="text-sm text-slate-500 mt-1">如需重新安装，请删除 config.yaml 与 data/install.lock</p>
          <div class="mt-6 flex justify-center gap-3">
            <router-link to="/login" class="btn-primary">前往登录</router-link>
          </div>
        </div>

        <div v-else-if="step === 0">
          <h2 class="font-semibold mb-1">第一步 · 环境检测</h2>
          <p class="text-sm text-slate-500 mb-5">填写连接信息后点击检测，全部通过才能进入下一步。</p>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="md:col-span-2">
              <div class="mb-1 text-sm font-semibold text-slate-700">MySQL 数据库</div>
              <div class="grid gap-3 md:grid-cols-2">
                <div><label class="label">地址</label><input v-model="form.mysql.host" class="input" placeholder="127.0.0.1" /></div>
                <div><label class="label">端口</label><input v-model.number="form.mysql.port" class="input" placeholder="3306" /></div>
                <div><label class="label">用户名</label><input v-model="form.mysql.user" class="input" placeholder="root" /></div>
                <div><label class="label">密码</label><input v-model="form.mysql.password" type="password" class="input" placeholder="数据库密码" /></div>
                <div class="md:col-span-2"><label class="label">数据库名（不存在将自动创建）</label><input v-model="form.mysql.name" class="input" placeholder="xorapi" /></div>
              </div>
            </div>
            <div>
              <div class="mb-1 text-sm font-semibold text-slate-700">Redis（可选）</div>
              <div class="grid gap-3">
                <div><label class="label">地址</label><input v-model="form.redis.host" class="input" placeholder="127.0.0.1" /></div>
                <div class="grid grid-cols-2 gap-3">
                  <div><label class="label">端口</label><input v-model.number="form.redis.port" class="input" placeholder="6379" /></div>
                  <div><label class="label">密码</label><input v-model="form.redis.password" type="password" class="input" placeholder="可空" /></div>
                </div>
              </div>
            </div>
            <div class="flex items-end">
              <div class="w-full space-y-2 rounded-lg border border-slate-200 p-3 text-sm">
                <div class="flex items-center justify-between">
                  <span>数据目录可写</span>
                  <span :class="checkClass(checks.dir)">{{ checkText(checks.dir) }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>MySQL 连接</span>
                  <span :class="checkClass(checks.mysql)">{{ checkText(checks.mysql) }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>Redis 连接</span>
                  <span :class="checkClass(checks.redis)">{{ checkText(checks.redis) }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="mt-6 flex justify-end gap-3">
            <button class="btn-ghost" :disabled="checking" @click="runChecks">
              {{ checking ? '检测中...' : '开始检测' }}
            </button>
            <button class="btn-primary" :disabled="!canNext" @click="step = 1">下一步</button>
          </div>
        </div>

        <div v-else-if="step === 1">
          <h2 class="font-semibold mb-1">第二步 · 站点与管理员</h2>
          <p class="text-sm text-slate-500 mb-5">设置站点名称与超级管理员账号（安装后用于登录管理后台）。</p>
          <div class="grid gap-4 max-w-md">
            <div><label class="label">站点名称</label><input v-model="form.site_name" class="input" placeholder="XorAPI" /></div>
            <div><label class="label">管理员邮箱</label><input v-model="form.admin.email" class="input" placeholder="admin@example.com" /></div>
            <div><label class="label">管理员用户名</label><input v-model="form.admin.username" class="input" placeholder="admin" /></div>
            <div>
              <label class="label">管理员密码（至少 6 位）</label>
              <input v-model="form.admin.password" type="password" class="input" placeholder="••••••" />
            </div>
          </div>
          <div class="mt-6 flex justify-between">
            <button class="btn-ghost" @click="step = 0">上一步</button>
            <button class="btn-primary" :disabled="!step2Valid" @click="step = 2">下一步</button>
          </div>
        </div>

        <div v-else-if="step === 2">
          <h2 class="font-semibold mb-1">第三步 · 确认并安装</h2>
          <p class="text-sm text-slate-500 mb-5">确认以下信息无误后开始安装，安装过程将自动创建数据表与管理员账号。</p>
          <div class="rounded-lg bg-slate-50 p-4 text-sm space-y-1.5 mb-6">
            <div class="flex justify-between"><span class="text-slate-500">站点名称</span><span>{{ form.site_name || 'XorAPI' }}</span></div>
            <div class="flex justify-between"><span class="text-slate-500">MySQL</span><span>{{ form.mysql.user }}@{{ form.mysql.host }}:{{ form.mysql.port }}/{{ form.mysql.name }}</span></div>
            <div class="flex justify-between"><span class="text-slate-500">Redis</span><span>{{ form.redis.host ? form.redis.host + ':' + form.redis.port : '未配置' }}</span></div>
            <div class="flex justify-between"><span class="text-slate-500">管理员</span><span>{{ form.admin.email }}</span></div>
          </div>
          <div class="flex justify-between">
            <button class="btn-ghost" :disabled="installing" @click="step = 1">上一步</button>
            <button class="btn-primary" :disabled="installing" @click="doInstall">
              {{ installing ? '正在安装，请稍候...' : '开始安装' }}
            </button>
          </div>
          <div v-if="error" class="mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-600">{{ error }}</div>
        </div>

        <div v-else class="text-center py-8">
          <div class="text-4xl mb-3">🎉</div>
          <h2 class="text-lg font-semibold">安装完成！</h2>
          <p class="text-sm text-slate-500 mt-1">配置文件 config.yaml 已生成，系统即刻可用。</p>
          <div class="mt-6 flex justify-center gap-3">
            <router-link to="/login" class="btn-primary">前往登录</router-link>
          </div>
          <p class="mt-4 text-xs text-slate-400">建议立即登录并修改默认设置，安装文件请妥善保管 config.yaml</p>
        </div>
      </div>

      <p class="mt-6 text-center text-xs text-slate-500">XorAPI · 基于 Go + Vue 的高性能 AI API 中转站</p>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { get, post } from '../api'
import { ok, toast } from '../stores/toast'

const steps = ['环境检测', '配置填写', '安装执行']
const step = ref(0)
const installed = ref(false)
const checking = ref(false)
const installing = ref(false)
const error = ref('')

const form = reactive({
  mysql: { host: '127.0.0.1', port: 3306, user: 'root', password: '', name: 'xorapi' },
  redis: { host: '127.0.0.1', port: 6379, password: '', db: 0 },
  site_name: 'XorAPI',
  admin: { email: '', username: 'admin', password: '' },
})

const checks = reactive({
  dir: { state: 'pending', msg: '未检测' },
  mysql: { state: 'pending', msg: '未检测' },
  redis: { state: 'pending', msg: '未检测' },
})

const canNext = computed(() => checks.mysql.state === 'ok' && checks.dir.state === 'ok')
const step2Valid = computed(
  () => form.admin.email.includes('@') && form.admin.password.length >= 6
)

function checkClass(c) {
  return {
    ok: 'text-emerald-600',
    fail: 'text-red-600',
    pending: 'text-slate-400',
  }[c.state]
}
function checkText(c) {
  return c.state === 'pending' ? c.msg : (c.state === 'ok' ? '✓ ' + c.msg : '✗ ' + c.msg)
}

async function runChecks() {
  checking.value = true
  try {
    const data = await post('/api/install/preflight', form)
    checks.mysql = data.mysql
    checks.redis = data.redis
    checks.dir = data.dir
    if (data.mysql.ok && data.dir.ok) ok('环境检测通过')
    else toast('MySQL 或目录检测未通过，请修正后重试', 'error')
  } finally {
    checking.value = false
  }
}

async function doInstall() {
  installing.value = true
  error.value = ''
  try {
    await post('/api/install/execute', form)
    step.value = 3
    ok('安装成功')
  } catch (e) {
    error.value = e.response?.data?.msg || '安装失败，请检查填写信息'
  } finally {
    installing.value = false
  }
}

onMounted(async () => {
  try {
    const data = await get('/api/install/status')
    installed.value = data.installed
  } catch {
    /* 忽略 */
  }
})
</script>
