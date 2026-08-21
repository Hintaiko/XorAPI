<template>
  <div class="max-w-2xl space-y-6">
    <div class="card p-6">
      <h3 class="font-semibold">基本信息</h3>
      <div class="mt-4 space-y-4">
        <div>
          <label class="label">用户 ID（不可更改）</label>
          <input :value="user.user?.id" disabled class="input bg-slate-50" />
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div><label class="label">用户名</label><input v-model="form.username" class="input" /></div>
          <div><label class="label">邮箱</label><input v-model="form.email" type="email" class="input" /></div>
        </div>
        <div class="text-right"><button class="btn-primary" @click="save">保存修改</button></div>
      </div>
    </div>

    <div class="card p-6">
      <h3 class="font-semibold">修改密码</h3>
      <div class="mt-4 space-y-4">
        <div><label class="label">原密码</label><input v-model="pwd.old" type="password" class="input" /></div>
        <div><label class="label">新密码（至少 6 位）</label><input v-model="pwd.next" type="password" class="input" /></div>
        <div class="text-right"><button class="btn-ghost" @click="changePwd">修改密码</button></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive } from 'vue'
import { put, post } from '../api'
import { ok } from '../stores/toast'
import { useUser } from '../stores/user'

const user = useUser()
const form = reactive({
  username: user.user?.username || '',
  email: user.user?.email || '',
})
const pwd = reactive({ old: '', next: '' })

async function save() {
  await put('/api/user/profile', form)
  ok('已保存')
  user.refresh()
}

async function changePwd() {
  if (pwd.next.length < 6) return ok('新密码至少 6 位')
  await post('/api/user/password', { old_password: pwd.old, new_password: pwd.next })
  ok('密码已修改')
  pwd.old = ''
  pwd.next = ''
}
</script>
