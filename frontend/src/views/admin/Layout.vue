<template>
  <div class="shell">
    <header class="top">
      <div class="brand">TDY 管理端</div>
      <nav>
        <router-link v-if="auth.hasPerm('dashboard.view')" to="/admin/dashboard">仪表盘</router-link>
        <router-link v-if="auth.hasPerm('order.view')" to="/admin/orders">工单</router-link>
        <router-link v-if="auth.hasPerm('user.view','admin.manage')" to="/admin/users">用户</router-link>
        <router-link v-if="auth.hasPerm('wallet.view')" to="/admin/wallets">钱包</router-link>
        <router-link v-if="auth.hasPerm('withdraw.view')" to="/admin/withdrawals">提现</router-link>
        <router-link v-if="auth.hasPerm('reconcile.view')" to="/admin/reconcile">对账</router-link>
        <router-link v-if="auth.hasPerm('audit.view')" to="/admin/audit">审计</router-link>
        <router-link v-if="auth.hasPerm('role.manage')" to="/admin/roles">角色</router-link>
      </nav>
      <div class="spacer" />
      <NotifyBell />
      <el-tag v-if="auth.user?.admin_role?.name" size="small" type="info">{{ auth.user.admin_role.name }}</el-tag>
      <el-button link @click="pwdVisible=true">改密</el-button>
      <span class="muted">{{ auth.user?.display_name }}</span>
      <el-button link type="danger" @click="logout">退出</el-button>
    </header>
    <main class="page">
      <router-view />
    </main>

    <el-dialog v-model="pwdVisible" title="修改密码" width="400px">
      <el-form label-position="top">
        <el-form-item label="原密码"><el-input v-model="pwd.old_password" type="password" show-password /></el-form-item>
        <el-form-item label="新密码"><el-input v-model="pwd.new_password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdVisible=false">取消</el-button>
        <el-button type="primary" @click="changePwd">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../../stores/auth'
import http from '../../api'
import NotifyBell from '../../components/NotifyBell.vue'

const auth = useAuthStore()
const router = useRouter()
const pwdVisible = ref(false)
const pwd = reactive({ old_password: '', new_password: '' })

onMounted(() => {
  auth.fetchMe().catch(() => {})
})

function logout() {
  auth.logout()
  router.push('/login')
}

async function changePwd() {
  try {
    await http.post('/auth/change-password', pwd)
    ElMessage.success('密码已修改，请重新登录')
    pwdVisible.value = false
    logout()
  } catch (e) {
    ElMessage.error(e.message)
  }
}
</script>

<style scoped>
.shell { min-height: 100%; }
.top {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 20px;
  background: rgba(255,255,255,0.85);
  border-bottom: 1px solid var(--line);
  backdrop-filter: blur(8px);
  position: sticky;
  top: 0;
  z-index: 10;
}
.brand { font-weight: 700; color: var(--accent); }
nav { display: flex; gap: 12px; flex-wrap: wrap; }
nav a {
  color: var(--muted);
  text-decoration: none;
  padding: 4px 8px;
  border-radius: 6px;
}
nav a.router-link-active {
  color: var(--accent);
  background: var(--accent-soft);
  font-weight: 600;
}
</style>
