<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">对账</h2>
      <div class="spacer" />
      <el-button v-if="auth.hasPerm('reconcile.export')" @click="exportXlsx">导出 Excel</el-button>
      <el-button type="primary" @click="load">重新对账</el-button>
    </div>

    <div class="card" style="margin-bottom:16px" v-if="summary">
      <div class="row">
        <div>当前总余额：<span class="money">¥{{ fenToYuan(summary.total_balance) }}</span></div>
        <div>验收奖励合计：¥{{ fenToYuan(summary.total_approved) }}</div>
        <div>入账合计：¥{{ fenToYuan(summary.total_credit) }}</div>
        <div>扣减合计：¥{{ fenToYuan(summary.total_debit) }}</div>
        <el-tag :type="summary.ledger_ok ? 'success' : 'danger'">
          {{ summary.ledger_ok ? '账本平衡' : '账本异常' }}
        </el-tag>
      </div>
      <p class="muted" style="margin:10px 0 0">校验规则：余额 = 入账合计 − 扣减合计</p>
    </div>

    <div class="card">
      <el-table :data="users" stripe v-loading="loading">
        <el-table-column prop="user_id" label="ID" width="70" />
        <el-table-column prop="display_name" label="昵称" />
        <el-table-column label="余额" width="110">
          <template #default="{ row }">¥{{ fenToYuan(row.wallet_balance) }}</template>
        </el-table-column>
        <el-table-column label="验收奖励" width="110">
          <template #default="{ row }">¥{{ fenToYuan(row.approved_reward) }}</template>
        </el-table-column>
        <el-table-column label="入账" width="100">
          <template #default="{ row }">¥{{ fenToYuan(row.credit_total) }}</template>
        </el-table-column>
        <el-table-column label="扣减" width="100">
          <template #default="{ row }">¥{{ fenToYuan(row.debit_total) }}</template>
        </el-table-column>
        <el-table-column label="差额" width="100">
          <template #default="{ row }">¥{{ fenToYuan(row.diff) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.ok ? 'success' : 'danger'" size="small">{{ row.ok ? '正常' : '异常' }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http, { fenToYuan } from '../../api'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const loading = ref(false)
const summary = ref(null)
const users = ref([])

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/admin/reconcile')
    summary.value = data.summary
    users.value = data.users
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function exportXlsx() {
  try {
    const res = await http.get('/admin/reconcile/export', { responseType: 'blob' })
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = 'reconcile.xlsx'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    ElMessage.error(e.message || '导出失败')
  }
}

onMounted(load)
</script>
