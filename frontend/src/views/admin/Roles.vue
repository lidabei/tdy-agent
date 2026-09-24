<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">角色权限</h2>
      <div class="spacer" />
      <el-button type="primary" @click="openCreate">新建角色</el-button>
    </div>
    <div class="card">
      <el-table :data="roles" stripe v-loading="loading">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="code" label="编码" width="140" />
        <el-table-column prop="name" label="名称" width="140" />
        <el-table-column prop="description" label="说明" min-width="160" />
        <el-table-column label="系统" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.is_system" size="small" type="warning">是</el-tag>
            <span v-else class="muted">否</span>
          </template>
        </el-table-column>
        <el-table-column label="权限数" width="90">
          <template #default="{ row }">{{ row.permissions?.length || 0 }}</template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" :disabled="row.is_system" @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="visible" :title="editingId ? '编辑角色' : '新建角色'" width="560px">
      <el-form label-position="top">
        <el-form-item label="编码">
          <el-input v-model="form.code" :disabled="!!editingId" placeholder="如 custom_ops" />
        </el-form-item>
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="form.description" /></el-form-item>
        <el-form-item label="权限">
          <div v-for="(list, module) in permGroups" :key="module" class="perm-group">
            <div class="perm-mod">{{ moduleLabel(module) }}</div>
            <el-checkbox-group v-model="form.permission_ids">
              <el-checkbox v-for="p in list" :key="p.id" :label="p.id">{{ p.name }}</el-checkbox>
            </el-checkbox-group>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../../api'

const roles = ref([])
const perms = ref([])
const loading = ref(false)
const saving = ref(false)
const visible = ref(false)
const editingId = ref(null)
const form = reactive({ code: '', name: '', description: '', permission_ids: [] })

const permGroups = computed(() => {
  const g = {}
  for (const p of perms.value) {
    if (!g[p.module]) g[p.module] = []
    g[p.module].push(p)
  }
  return g
})

function moduleLabel(m) {
  return ({
    dashboard: '仪表盘',
    order: '工单',
    user: '用户',
    wallet: '钱包',
    withdraw: '提现',
    reconcile: '对账',
    audit: '审计',
    role: '角色/管理员',
  })[m] || m
}

async function load() {
  loading.value = true
  try {
    const [r, p] = await Promise.all([
      http.get('/admin/roles'),
      http.get('/admin/permissions'),
    ])
    roles.value = r.data || []
    perms.value = p.data || []
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.code = ''
  form.name = ''
  form.description = ''
  form.permission_ids = []
  visible.value = true
}

function openEdit(row) {
  editingId.value = row.id
  form.code = row.code
  form.name = row.name
  form.description = row.description || ''
  form.permission_ids = (row.permissions || []).map((p) => p.id)
  visible.value = true
}

async function save() {
  saving.value = true
  try {
    const body = {
      code: form.code,
      name: form.name,
      description: form.description,
      permission_ids: form.permission_ids,
    }
    if (editingId.value) {
      await http.put(`/admin/roles/${editingId.value}`, body)
    } else {
      await http.post('/admin/roles', body)
    }
    ElMessage.success('已保存')
    visible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function remove(row) {
  await ElMessageBox.confirm(`确认删除角色「${row.name}」？`, '提示')
  try {
    await http.delete(`/admin/roles/${row.id}`)
    ElMessage.success('已删除')
    load()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

onMounted(load)
</script>

<style scoped>
.perm-group { margin-bottom: 12px; }
.perm-mod { font-weight: 600; margin-bottom: 6px; color: var(--muted); font-size: 13px; }
</style>
