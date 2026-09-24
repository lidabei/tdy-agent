<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">用户钱包</h2>
      <div class="spacer" />
      <el-button @click="load">刷新</el-button>
      <el-button v-if="auth.hasPerm('wallet.adjust')" type="primary" @click="openAdjust">手动调账</el-button>
    </div>

    <div class="card" style="margin-bottom:16px">
      <el-table :data="wallets" stripe v-loading="loading">
        <el-table-column prop="user_id" label="用户ID" width="90" />
        <el-table-column prop="display_name" label="昵称" />
        <el-table-column prop="username" label="用户名" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="余额">
          <template #default="{ row }"><span class="money">¥{{ fenToYuan(row.balance) }}</span></template>
        </el-table-column>
        <el-table-column label="流水" width="100">
          <template #default="{ row }">
            <el-button link type="primary" @click="showTx(row)">查看</el-button>
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

    <div class="card" v-if="txUser">
      <h3 style="margin-top:0">{{ txUser.display_name }} 的流水</h3>
      <el-table :data="txs" stripe size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="type" label="类型" width="90" />
        <el-table-column label="金额" width="110">
          <template #default="{ row }">¥{{ fenToYuan(row.amount) }}</template>
        </el-table-column>
        <el-table-column label="余额" width="110">
          <template #default="{ row }">¥{{ fenToYuan(row.balance) }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" />
        <el-table-column prop="created_at" label="时间" min-width="160" />
      </el-table>
      <div class="row" style="margin-top:12px;justify-content:flex-end">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="txTotal"
          :page-size="txSize"
          :current-page="txPage"
          @current-change="(p) => { txPage=p; showTx(txUser) }"
        />
      </div>
    </div>

    <el-dialog v-model="adjustVisible" title="手动调账" width="420px">
      <el-form label-position="top">
        <el-form-item label="用户">
          <el-select v-model="adjust.user_id" style="width:100%">
            <el-option v-for="w in wallets" :key="w.user_id" :label="`${w.display_name} (${w.username})`" :value="w.user_id" />
          </el-select>
        </el-form-item>
        <el-form-item label="金额（元，负数为扣减）">
          <el-input-number v-model="adjust.amount_yuan" :step="1" :precision="2" style="width:100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="adjust.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="adjustVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="doAdjust">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http, { fenToYuan } from '../../api'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const wallets = ref([])
const txs = ref([])
const txUser = ref(null)
const loading = ref(false)
const saving = ref(false)
const page = ref(1)
const size = ref(20)
const total = ref(0)
const txPage = ref(1)
const txSize = ref(20)
const txTotal = ref(0)
const adjustVisible = ref(false)
const adjust = reactive({ user_id: null, amount_yuan: 10, remark: '' })

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/admin/wallets', { params: { page: page.value, size: size.value } })
    wallets.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

async function showTx(row) {
  txUser.value = row
  const { data } = await http.get('/admin/wallets/transactions', {
    params: { user_id: row.user_id, page: txPage.value, size: txSize.value },
  })
  txs.value = data.list || []
  txTotal.value = data.total || 0
}

function openAdjust() {
  adjust.user_id = wallets.value[0]?.user_id || null
  adjust.amount_yuan = 10
  adjust.remark = ''
  adjustVisible.value = true
}

async function doAdjust() {
  saving.value = true
  try {
    await http.post('/admin/wallets/adjust', adjust)
    ElMessage.success('调账成功')
    adjustVisible.value = false
    load()
    if (txUser.value?.user_id === adjust.user_id) showTx(txUser.value)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
