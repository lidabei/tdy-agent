<template>
  <div class="login-wrap">
    <div class="login-card card">
      <h1>TDY 派单管理</h1>
      <p class="muted">管理员发单指定用户 · 接单验收 · 钱包入账对账</p>
      <el-form @submit.prevent="onSubmit" label-position="top">
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="admin / zhangsan" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password placeholder="密码" @keyup.enter="onSubmit" />
        </el-form-item>
        <el-button type="primary" style="width:100%" :loading="loading" @click="onSubmit">登录</el-button>
      </el-form>
      <p class="hint muted">演示账号：admin/admin123 · zhangsan/pass1234 · lisi/pass1234</p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const loading = ref(false)
const form = reactive({ username: 'admin', password: 'admin123' })

async function onSubmit() {
  loading.value = true
  try {
    const user = await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    if (user.role !== 'admin') {
      router.push('/app/orders')
      return
    }
    const candidates = [
      ['dashboard.view', '/admin/dashboard'],
      ['order.view', '/admin/orders'],
      ['user.view', '/admin/users'],
      ['wallet.view', '/admin/wallets'],
      ['withdraw.view', '/admin/withdrawals'],
      ['reconcile.view', '/admin/reconcile'],
      ['audit.view', '/admin/audit'],
      ['role.manage', '/admin/roles'],
    ]
    const hit = candidates.find(([p]) => auth.hasPerm(p))
    router.push(hit ? hit[1] : '/admin/dashboard')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  min-height: 100%;
  display: grid;
  place-items: center;
  padding: 24px;
}
.login-card {
  width: min(420px, 100%);
}
.login-card h1 {
  margin: 0 0 6px;
  font-size: 28px;
  letter-spacing: 0.02em;
}
.hint { margin-top: 14px; font-size: 12px; }
</style>
