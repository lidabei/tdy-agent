<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">工单管理</h2>
      <div class="spacer" />
      <el-select v-model="status" clearable placeholder="状态" style="width:130px" @change="() => { page=1; load() }">
        <el-option v-for="(v,k) in statusMap" :key="k" :label="v.label" :value="k" />
      </el-select>
      <el-checkbox v-model="onlyOverdue" @change="() => { page=1; load() }">仅超期</el-checkbox>
      <el-button v-if="auth.hasPerm('order.create')" @click="openBatch">批量派单</el-button>
      <el-button v-if="auth.hasPerm('order.create')" type="primary" @click="openCreate">发布工单</el-button>
    </div>

    <div class="card">
      <el-table :data="orders" stripe v-loading="loading">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="title" label="标题" min-width="140" />
        <el-table-column label="用户" width="110">
          <template #default="{ row }">{{ row.assignee?.display_name || row.assignee_id }}</template>
        </el-table-column>
        <el-table-column label="奖励" width="90">
          <template #default="{ row }"><span class="money">¥{{ fenToYuan(row.reward) }}</span></template>
        </el-table-column>
        <el-table-column label="截止" width="110">
          <template #default="{ row }">
            <span :style="{color: row.overdue ? '#b91c1c' : ''}">{{ row.deadline ? String(row.deadline).slice(0,10) : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusMap[row.status]?.type" size="small">{{ statusMap[row.status]?.label }}</el-tag>
            <el-tag v-if="row.overdue" type="danger" size="small" style="margin-left:4px">超期</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="附件" width="90">
          <template #default="{ row }">
            <el-button v-if="row.attachments?.length" link type="primary" @click="showFiles(row)">
              {{ row.attachments.length }} 个
            </el-button>
            <span v-else class="muted">0</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button v-if="auth.hasPerm('order.approve') && row.status==='submitted'" link type="success" @click="approve(row)">通过</el-button>
            <el-button v-if="auth.hasPerm('order.approve') && row.status==='submitted'" link type="danger" @click="reject(row)">退回</el-button>
            <el-button v-if="auth.hasPerm('order.reassign') && (row.status==='pending'||row.status==='accepted'||row.status==='rejected')" link @click="reassign(row)">改派</el-button>
            <el-button v-if="auth.hasPerm('order.cancel') && (row.status==='pending'||row.status==='accepted')" link type="warning" @click="cancel(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="row" style="margin-top:12px;justify-content:flex-end">
        <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page" @current-change="(p)=>{page=p;load()}" />
      </div>
    </div>

    <el-dialog v-model="createVisible" :title="batchMode ? '批量派单' : '发布工单'" width="480px">
      <el-form label-position="top">
        <el-form-item label="标题" required><el-input v-model="form.title" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="奖励（元）" required>
          <el-input-number v-model="form.reward_yuan" :min="0.01" :precision="2" style="width:100%" />
        </el-form-item>
        <el-form-item label="截止日期">
          <el-date-picker v-model="form.deadline" type="date" value-format="YYYY-MM-DD" style="width:100%" />
        </el-form-item>
        <el-form-item v-if="!batchMode" label="指定用户" required>
          <el-select v-model="form.assignee_id" filterable style="width:100%">
            <el-option v-for="u in enabledUsers" :key="u.id" :label="`${u.display_name} (${u.username})`" :value="u.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-else label="指定多个用户" required>
          <el-select v-model="form.assignee_ids" multiple filterable style="width:100%">
            <el-option v-for="u in enabledUsers" :key="u.id" :label="`${u.display_name} (${u.username})`" :value="u.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="create">发布</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="filesVisible" title="工单附件" width="480px">
      <el-table :data="fileList" size="small">
        <el-table-column prop="file_name" label="文件名" min-width="180" />
        <el-table-column label="大小" width="100">
          <template #default="{ row }">{{ (row.size/1024).toFixed(1) }} KB</template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button link type="primary" @click="downloadFile(row)">下载</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import http, { fenToYuan, statusMap } from '../../api'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const orders = ref([])
const users = ref([])
const loading = ref(false)
const saving = ref(false)
const status = ref(route.query.status ? String(route.query.status) : '')
const onlyOverdue = ref(route.query.overdue === '1')
const page = ref(1)
const size = ref(20)
const total = ref(0)
const createVisible = ref(false)
const batchMode = ref(false)
const filesVisible = ref(false)
const fileList = ref([])
const form = reactive({ title: '', description: '', reward_yuan: 50, assignee_id: null, assignee_ids: [], deadline: '' })
const enabledUsers = computed(() => users.value.filter(u => u.enabled !== false))

async function showFiles(row) {
  fileList.value = row.attachments || []
  if (!fileList.value.length) {
    const { data } = await http.get(`/orders/${row.id}/attachments`)
    fileList.value = data || []
  }
  filesVisible.value = true
}

async function downloadFile(row) {
  try {
    const res = await http.get(`/attachments/${row.id}/download`, { responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = row.file_name || 'file'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    ElMessage.error(e.message || '下载失败')
  }
}

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/admin/orders', {
      params: { status: status.value || undefined, overdue: onlyOverdue.value ? 1 : undefined, page: page.value, size: size.value },
    })
    orders.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function loadUsers() {
  const { data } = await http.get('/admin/users', { params: { role: 'user', all: 1 } })
  users.value = Array.isArray(data) ? data : (data.list || [])
}

function openCreate() {
  batchMode.value = false
  Object.assign(form, { title: '', description: '', reward_yuan: 50, assignee_id: enabledUsers.value[0]?.id || null, assignee_ids: [], deadline: '' })
  createVisible.value = true
}
function openBatch() {
  batchMode.value = true
  Object.assign(form, { title: '', description: '', reward_yuan: 50, assignee_id: null, assignee_ids: [], deadline: '' })
  createVisible.value = true
}

async function create() {
  saving.value = true
  try {
    if (batchMode.value) {
      await http.post('/admin/orders/batch', {
        title: form.title, description: form.description, reward_yuan: form.reward_yuan,
        assignee_ids: form.assignee_ids, deadline: form.deadline || undefined,
      })
    } else {
      await http.post('/admin/orders', {
        title: form.title, description: form.description, reward_yuan: form.reward_yuan,
        assignee_id: form.assignee_id, deadline: form.deadline || undefined,
      })
    }
    ElMessage.success('已发布')
    createVisible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

async function approve(row) {
  await ElMessageBox.confirm(`确认验收通过并入账 ¥${fenToYuan(row.reward)}？`, '验收')
  try {
    await http.post(`/admin/orders/${row.id}/approve`)
    ElMessage.success('已入账')
    load()
  } catch (e) { ElMessage.error(e.message) }
}
async function reject(row) {
  const { value } = await ElMessageBox.prompt('退回原因', '退回', { inputPattern: /.+/, inputErrorMessage: '必填' })
  try {
    await http.post(`/admin/orders/${row.id}/reject`, { reason: value })
    ElMessage.success('已退回')
    load()
  } catch (e) { ElMessage.error(e.message) }
}
async function cancel(row) {
  await ElMessageBox.confirm('确认取消？', '取消')
  try {
    await http.post(`/admin/orders/${row.id}/cancel`)
    ElMessage.success('已取消')
    load()
  } catch (e) { ElMessage.error(e.message) }
}
async function reassign(row) {
  const options = enabledUsers.value.map(u => `${u.id}:${u.display_name}`).join('\n')
  const { value } = await ElMessageBox.prompt(`输入新用户ID\n可选：\n${options}`, '改派', { inputPattern: /^\d+$/, inputErrorMessage: '请输入数字ID' })
  try {
    await http.post(`/admin/orders/${row.id}/reassign`, { assignee_id: Number(value) })
    ElMessage.success('已改派')
    load()
  } catch (e) { ElMessage.error(e.message) }
}

onMounted(async () => { await loadUsers(); await load() })
</script>
