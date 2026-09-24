<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">我的工单</h2>
      <div class="spacer" />
      <el-select v-model="status" clearable placeholder="状态" style="width:140px" @change="() => { page=1; load() }">
        <el-option v-for="(v,k) in statusMap" :key="k" :label="v.label" :value="k" />
      </el-select>
      <el-button @click="load">刷新</el-button>
    </div>

    <div class="card">
      <el-table :data="orders" stripe v-loading="loading">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="title" label="标题" min-width="140" />
        <el-table-column label="奖励" width="90">
          <template #default="{ row }"><span class="money">¥{{ fenToYuan(row.reward) }}</span></template>
        </el-table-column>
        <el-table-column label="截止" width="110">
          <template #default="{ row }">
            <span :style="{color: row.overdue ? '#b91c1c' : ''}">{{ row.deadline ? String(row.deadline).slice(0,10) : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusMap[row.status]?.type" size="small">{{ statusMap[row.status]?.label }}</el-tag>
            <el-tag v-if="row.overdue" type="danger" size="small" style="margin-left:4px">超期</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="说明" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.status==='rejected'">退回：{{ row.reject_reason }}</span>
            <span v-else>{{ row.description || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status==='pending'" link type="primary" @click="accept(row)">接单</el-button>
            <el-button v-if="row.status==='accepted'||row.status==='rejected'" link type="success" @click="openSubmit(row)">提交完成</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="row" style="margin-top:12px;justify-content:flex-end">
        <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page" @current-change="(p)=>{page=p;load()}" />
      </div>
    </div>

    <el-dialog v-model="submitVisible" title="提交完成" width="480px">
      <el-input v-model="note" type="textarea" :rows="3" placeholder="完成说明" style="margin-bottom:12px" />
      <el-upload :http-request="uploadFile" :show-file-list="true" multiple>
        <el-button>上传附件</el-button>
      </el-upload>
      <p class="muted" style="font-size:12px;margin-top:8px">可先上传附件再点提交验收</p>
      <template #footer>
        <el-button @click="submitVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">提交验收</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http, { fenToYuan, statusMap } from '../../api'

const orders = ref([])
const loading = ref(false)
const saving = ref(false)
const status = ref('')
const page = ref(1)
const size = ref(20)
const total = ref(0)
const submitVisible = ref(false)
const note = ref('')
const current = ref(null)

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/orders/mine', { params: { status: status.value || undefined, page: page.value, size: size.value } })
    orders.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function accept(row) {
  await ElMessageBox.confirm(`确认接单「${row.title}」？`, '接单')
  try {
    await http.post(`/orders/${row.id}/accept`)
    ElMessage.success('已接单')
    load()
  } catch (e) { ElMessage.error(e.message) }
}

function openSubmit(row) {
  current.value = row
  note.value = ''
  submitVisible.value = true
}

async function uploadFile(opt) {
  const fd = new FormData()
  fd.append('file', opt.file)
  try {
    await http.post(`/orders/${current.value.id}/attachments`, fd)
    ElMessage.success('附件已上传')
    opt.onSuccess?.()
  } catch (e) {
    ElMessage.error(e.message)
    opt.onError?.(e)
  }
}

async function submit() {
  saving.value = true
  try {
    await http.post(`/orders/${current.value.id}/submit`, { note: note.value })
    ElMessage.success('已提交验收')
    submitVisible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
