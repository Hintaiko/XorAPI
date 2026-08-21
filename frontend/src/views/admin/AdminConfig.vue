<template>
  <div class="max-w-3xl space-y-6">
    <div class="card p-6">
      <h3 class="font-semibold">功能开关</h3>
      <div class="mt-4 divide-y divide-slate-100">
        <div v-for="s in switches" :key="s.key" class="flex items-center justify-between py-3">
          <div>
            <div class="text-sm font-medium">{{ s.label }}</div>
            <div class="text-xs text-slate-400">{{ s.desc }}</div>
          </div>
          <button class="relative h-6 w-11 rounded-full transition" :class="cfg[s.key] === 'true' ? 'bg-indigo-600' : 'bg-slate-300'"
            @click="cfg[s.key] = cfg[s.key] === 'true' ? 'false' : 'true'">
            <span class="absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-all"
              :class="cfg[s.key] === 'true' ? 'left-[22px]' : 'left-0.5'"></span>
          </button>
        </div>
      </div>
    </div>

    <div class="card p-6">
      <h3 class="font-semibold">签到奖励</h3>
      <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <div><label class="label">基础奖励（点）</label><input v-model="cfg.checkin_base" class="input" /></div>
        <div><label class="label">连续签到递增（点/天）</label><input v-model="cfg.checkin_streak_bonus" class="input" /></div>
        <div><label class="label">单日奖励上限（点）</label><input v-model="cfg.checkin_max_reward" class="input" /></div>
        <div><label class="label">签到点数有效天数</label><input v-model="cfg.checkin_expire_days" class="input" /></div>
      </div>
    </div>

    <div class="card p-6">
      <h3 class="font-semibold">站点与邮件</h3>
      <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <div class="sm:col-span-2"><label class="label">站点公告（模型广场顶部展示）</label><input v-model="cfg.site_announcement" class="input" /></div>
        <div class="sm:col-span-2"><label class="label">点数兑换说明（仅展示）</label><input v-model="cfg.exchange_note" class="input" /></div>
        <div><label class="label">SMTP 服务器</label><input v-model="cfg.smtp_host" class="input" placeholder="smtp.example.com" /></div>
        <div><label class="label">SMTP 端口（465 SSL / 587 STARTTLS）</label><input v-model="cfg.smtp_port" class="input" /></div>
        <div><label class="label">SMTP 用户名</label><input v-model="cfg.smtp_user" class="input" /></div>
        <div><label class="label">SMTP 密码/授权码</label><input v-model="cfg.smtp_pass" type="password" class="input" /></div>
        <div><label class="label">发件人地址</label><input v-model="cfg.smtp_from" class="input" placeholder="noreply@example.com" /></div>
        <div><label class="label">中继每分钟限速（次/Key）</label><input v-model="cfg.relay_rpm" class="input" /></div>
      </div>
    </div>

    <div class="card p-6">
      <div class="flex items-center justify-between">
        <h3 class="font-semibold">邀请码</h3>
        <div class="flex gap-2">
          <input v-model.number="inviteCount" type="number" min="1" max="100" class="input !w-24" />
          <button class="btn-ghost" @click="genInvites">批量生成</button>
        </div>
      </div>
      <div class="mt-3 flex max-h-40 flex-wrap gap-1.5 overflow-auto">
        <code v-for="ic in invites" :key="ic.id"
          class="rounded bg-slate-100 px-2 py-1 font-mono text-xs"
          :class="ic.status === 'used' ? 'text-slate-400 line-through' : 'text-indigo-700'">{{ ic.code }}</code>
      </div>
    </div>

    <div class="sticky bottom-4 flex justify-end">
      <button class="btn-primary shadow-lg" @click="save">保存全部设置</button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { get, post, put } from '../../api'
import { ok } from '../../stores/toast'

const cfg = reactive({})
const invites = ref([])
const inviteCount = ref(10)

const switches = [
  { key: 'registration_enabled', label: '开放注册', desc: '关闭后新用户无法注册' },
  { key: 'invite_required', label: '强制邀请码', desc: '注册时必须填写邀请码' },
  { key: 'email_verify', label: '邮箱验证', desc: '注册需邮箱验证码（需配置 SMTP）' },
  { key: 'checkin_enabled', label: '签到功能', desc: '用户每日签到领取免费点数' },
]

async function load() {
  Object.assign(cfg, await get('/api/admin/configs'))
  invites.value = await get('/api/admin/invites')
}

async function save() {
  const keys = ['registration_enabled', 'invite_required', 'email_verify', 'checkin_enabled',
    'checkin_base', 'checkin_streak_bonus', 'checkin_max_reward', 'checkin_expire_days',
    'site_announcement', 'exchange_note', 'smtp_host', 'smtp_port', 'smtp_user', 'smtp_pass', 'smtp_from', 'relay_rpm']
  const payload = {}
  keys.forEach((k) => { if (cfg[k] !== undefined) payload[k] = String(cfg[k]) })
  await put('/api/admin/configs', payload)
  ok('设置已保存')
}

async function genInvites() {
  const data = await post('/api/admin/invites', { count: inviteCount.value })
  ok(`已生成 ${data.codes.length} 个邀请码`)
  invites.value = await get('/api/admin/invites')
}

onMounted(load)
</script>
