<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">用户管理</h2>
      <div class="spacer" />
      <el-radio-group v-model="filterRole" size="small" @change="() => { page=1; load() }">
        <el-radio-button label="">全部</el-radio-button>
        <el-radio-button label="user">普通用户</el-radio-button>
        <el-radio-button label="admin">管理员</el-radio-button>
      </el-radio-group>
      <el-button v-if="auth.hasPerm('user.create')" type="primary" @click="openCreate('user')">新建用户</el-button>
      <el-button v-if="auth.hasPerm('admin.manage')" type="warning" @click="openCreate('admin')">新建管理员</el-button>
    </div>
    <div class="card">
      <el-table :data="users" stripe v-loading="loading">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="username" label="用户名" />
        <el-table-column prop="display_name" label="昵称" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'warning' : ''" size="small">
              {{ row.role === 'admin' ? '管理员' : '用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="管理角色" min-width="120">
          <template #default="{ row }">
            <span v-if="row.role === 'admin'">{{ row.admin_role?.name || '—' }}</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="160" />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <template v-if="row.role === 'user' && auth.hasPerm('user.manage')">
              <el-button link type="primary" @click="resetPwd(row)">重置密码</el-button>
              <el-button link :type="row.enabled ? 'warning' : 'success'" @click="toggle(row)">
                {{ row.enabled ? '禁用' : '启用' }}
              </el-button>
            </template>
            <template v-else-if="row.role === 'admin' && auth.hasPerm('admin.manage')">
              <el-button link type="primary" @click="changeRole(row)">改角色</el-button>
              <el-button link type="primary" @click="resetPwd(row)">重置密码</el-button>
              <el-button
                link
                :type="row.enabled ? 'warning' : 'success'"
                :disabled="row.id === auth.user?.id"
                @click="toggle(row)"
              >
                {{ row.enabled ? '禁用' : '启用' }}
              </el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
      <div class="row" style="margin-top:12px;justify-content:flex-end">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="total"
          :page-size="size"
          :current-page="page"
          @current-change="(p) => { page=p; load() }"
        />
      </div>
    </div>

    <el-dialog v-model="visible" :title="form.role === 'admin' ? '新建管理员' : '新建普通用户'" width="420px">
      <el-form label-position="top">
        <el-form-item label="用户名"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="昵称"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" show-password /></el-form-item>
        <el-form-item v-if="form.role === 'admin'" label="管理角色">
          <el-select v-model="form.admin_role_id" style="width:100%" placeholder="选择角色">
            <el-option v-for="r in roles" :key="r.id" :label="`${r.name} (${r.code})`" :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="create">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="roleVisible" title="分配管理角色" width="400px">
      <el-form label-position="top">
        <el-form-item :label="`用户：${roleTarget?.display_name || ''}`">
          <el-select v-model="roleForm.admin_role_id" style="width:100%">
            <el-option v-for="r in roles" :key="r.id" :label="`${r.name} (${r.code})`" :value="r.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveRole">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../../api'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const users = ref([])
const roles = ref([])
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const page = ref(1)
const size = ref(20)
const total = ref(0)
const filterRole = ref('')
const form = reactive({
  username: '',
  display_name: '',
  password: 'pass1234',
  role: 'user',
  admin_role_id: null,
})
const roleVisible = ref(false)
const roleTarget = ref(null)
const roleForm = reactive({ admin_role_id: null })

async function loadRoles() {
  if (!auth.hasPerm('admin.manage', 'role.manage')) return
  try {
    const { data } = await http.get('/admin/roles')
    roles.value = data || []
  } catch {
    roles.value = []
  }
}

async function load() {
  loading.value = true
  try {
    const params = { page: page.value, size: size.value }
    if (filterRole.value) params.role = filterRole.value
    const { data } = await http.get('/admin/users', { params })
    users.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate(role) {
  form.role = role
  form.username = ''
  form.display_name = ''
  form.password = 'pass1234'
  form.admin_role_id = role === 'admin' ? (roles.value[0]?.id || null) : null
  visible.value = true
}

async function create() {
  saving.value = true
  try {
    const body = {
      username: form.username,
      display_name: form.display_name,
      password: form.password,
      role: form.role,
    }
    if (form.role === 'admin') body.admin_role_id = form.admin_role_id
    await http.post('/admin/users', body)
    ElMessage.success('已创建')
    visible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function toggle(row) {
  const next = !row.enabled
  await ElMessageBox.confirm(`确认${next ? '启用' : '禁用'}「${row.display_name}」？`, '提示')
  try {
    await http.post(`/admin/users/${row.id}/enabled`, { enabled: next })
    ElMessage.success('已更新')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function resetPwd(row) {
  const { value } = await ElMessageBox.prompt(`重置「${row.display_name}」的密码`, '重置密码', {
    inputValue: 'pass1234',
    inputPattern: /^(?=.*[A-Za-z])(?=.*\d).{6,}$/,
    inputErrorMessage: '至少6位且含字母和数字',
  })
  try {
    await http.post(`/admin/users/${row.id}/reset-password`, { password: value })
    ElMessage.success('密码已重置')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function changeRole(row) {
  if (!roles.value.length) await loadRoles()
  roleTarget.value = row
  roleForm.admin_role_id = row.admin_role_id || roles.value[0]?.id || null
  roleVisible.value = true
}

async function saveRole() {
  if (!roleTarget.value) return
  saving.value = true
  try {
    await http.post(`/admin/users/${roleTarget.value.id}/admin-role`, {
      admin_role_id: roleForm.admin_role_id,
    })
    ElMessage.success('已更新角色')
    roleVisible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadRoles()
  load()
})
</script>
