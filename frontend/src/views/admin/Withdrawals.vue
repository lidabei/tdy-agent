<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">提现审核</h2>
      <div class="spacer" />
      <el-select v-model="status" clearable placeholder="状态" style="width:140px" @change="() => { page=1; load() }">
        <el-option v-for="(v,k) in withdrawStatusMap" :key="k" :label="v.label" :value="k" />
      </el-select>
      <el-button v-if="auth.hasPerm('withdraw.review')" type="success" :disabled="!selected.length" @click="batchApprove">批量通过</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <div class="card">
      <el-table :data="list" stripe v-loading="loading" @selection-change="(rows)=>selected=rows">
        <el-table-column v-if="auth.hasPerm('withdraw.review')" type="selection" width="48" :selectable="(r)=>r.status==='pending'" />
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="用户" width="140">
          <template #default="{ row }">{{ row.user?.display_name || row.user_id }}</template>
        </el-table-column>
        <el-table-column label="金额" width="110">
          <template #default="{ row }"><span class="money">¥{{ fenToYuan(row.amount) }}</span></template>
        </el-table-column>
        <el-table-column prop="account_info" label="收款信息" min-width="160" show-overflow-tooltip />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="withdrawStatusMap[row.status]?.type" size="small">{{ withdrawStatusMap[row.status]?.label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="申请时间" min-width="160" />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <template v-if="auth.hasPerm('withdraw.review') && row.status==='pending'">
              <el-button link type="success" @click="approve(row)">通过</el-button>
              <el-button link type="danger" @click="reject(row)">拒绝</el-button>
            </template>
            <span v-else class="muted">{{ row.admin_remark || '-' }}</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="row" style="margin-top:12px;justify-content:flex-end">
        <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page" @current-change="(p)=>{page=p;load()}" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http, { fenToYuan, withdrawStatusMap } from '../../api'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()

const list = ref([])
const selected = ref([])
const loading = ref(false)
const status = ref('pending')
const page = ref(1)
const size = ref(20)
const total = ref(0)

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/admin/withdrawals', { params: { status: status.value || undefined, page: page.value, size: size.value } })
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function approve(row) {
  await ElMessageBox.confirm(`通过提现 ¥${fenToYuan(row.amount)}？`, '审核')
  try {
    await http.post(`/admin/withdrawals/${row.id}/approve`, { admin_remark: '已打款' })
    ElMessage.success('已通过')
    load()
  } catch (e) { ElMessage.error(e.message) }
}

async function batchApprove() {
  await ElMessageBox.confirm(`批量通过 ${selected.value.length} 笔？`, '批量审核')
  try {
    const { data } = await http.post('/admin/withdrawals/batch-approve', {
      ids: selected.value.map(r => r.id),
      admin_remark: '批量已打款',
    })
    ElMessage.success(`成功 ${data.ok}，失败 ${data.fail}`)
    load()
  } catch (e) { ElMessage.error(e.message) }
}

async function reject(row) {
  const { value } = await ElMessageBox.prompt('拒绝原因', '拒绝', { inputPattern: /.+/, inputErrorMessage: '必填' })
  try {
    await http.post(`/admin/withdrawals/${row.id}/reject`, { admin_remark: value })
    ElMessage.success('已拒绝')
    load()
  } catch (e) { ElMessage.error(e.message) }
}

onMounted(load)
</script>
