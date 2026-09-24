<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">我的钱包</h2>
      <div class="spacer" />
      <el-button type="primary" @click="openWithdraw">申请提现</el-button>
      <el-button @click="load">刷新</el-button>
    </div>

    <div class="card" style="margin-bottom:16px">
      <div class="row" style="gap:32px">
        <div>
          <div class="muted">当前余额</div>
          <div style="font-size:32px;margin-top:6px" class="money">¥{{ fenToYuan(wallet?.balance) }}</div>
        </div>
        <div>
          <div class="muted">可用余额</div>
          <div style="font-size:24px;margin-top:10px" class="money">¥{{ fenToYuan(wallet?.available) }}</div>
        </div>
        <div>
          <div class="muted">待审提现占用</div>
          <div style="font-size:24px;margin-top:10px">¥{{ fenToYuan(wallet?.pending_withdraw) }}</div>
        </div>
      </div>
    </div>

    <div class="card" style="margin-bottom:16px">
      <h3 style="margin-top:0">提现记录</h3>
      <el-table :data="withdrawals" stripe size="small">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="金额" width="110">
          <template #default="{ row }">¥{{ fenToYuan(row.amount) }}</template>
        </el-table-column>
        <el-table-column prop="account_info" label="收款信息" min-width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="withdrawStatusMap[row.status]?.type" size="small">{{ withdrawStatusMap[row.status]?.label }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="admin_remark" label="审核备注" min-width="120" />
        <el-table-column prop="created_at" label="时间" min-width="160" />
      </el-table>
    </div>

    <div class="card">
      <h3 style="margin-top:0">收支流水</h3>
      <el-table :data="txs" stripe v-loading="loading">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="类型" width="90">
          <template #default="{ row }">{{ row.type === 'credit' ? '入账' : '扣减' }}</template>
        </el-table-column>
        <el-table-column label="金额" width="110">
          <template #default="{ row }">
            <span :style="{ color: row.type==='credit' ? 'var(--accent)' : '#b91c1c' }">
              {{ row.type==='credit' ? '+' : '-' }}¥{{ fenToYuan(row.amount) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="余额" width="110">
          <template #default="{ row }">¥{{ fenToYuan(row.balance) }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="180" />
        <el-table-column prop="created_at" label="时间" min-width="160" />
      </el-table>
      <div class="row" style="margin-top:12px;justify-content:flex-end">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="txTotal"
          :page-size="txSize"
          :current-page="txPage"
          @current-change="(p) => { txPage=p; loadTx() }"
        />
      </div>
    </div>

    <el-dialog v-model="withdrawVisible" title="申请提现" width="440px">
      <el-form label-position="top">
        <el-form-item label="提现金额（元）">
          <el-input-number v-model="wForm.amount_yuan" :min="0.01" :precision="2" :step="1" style="width:100%" />
        </el-form-item>
        <el-form-item label="收款信息（支付宝/银行卡等）">
          <el-input v-model="wForm.account_info" placeholder="如：支付宝 138xxxx 张三" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="wForm.user_remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="withdrawVisible=false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="applyWithdraw">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http, { fenToYuan, withdrawStatusMap } from '../../api'

const wallet = ref(null)
const txs = ref([])
const withdrawals = ref([])
const loading = ref(false)
const saving = ref(false)
const withdrawVisible = ref(false)
const txPage = ref(1)
const txSize = ref(20)
const txTotal = ref(0)
const wForm = reactive({ amount_yuan: 10, account_info: '', user_remark: '' })

async function loadTx() {
  const t = await http.get('/wallet/transactions', { params: { page: txPage.value, size: txSize.value } })
  txs.value = t.data.list || []
  txTotal.value = t.data.total || 0
}

async function load() {
  loading.value = true
  try {
    const [w, wd] = await Promise.all([
      http.get('/wallet'),
      http.get('/withdrawals/mine', { params: { page: 1, size: 50 } }),
    ])
    wallet.value = w.data
    withdrawals.value = wd.data.list || []
    await loadTx()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function openWithdraw() {
  wForm.amount_yuan = Math.min(10, Number(fenToYuan(wallet.value?.available || 0)) || 0.01)
  wForm.account_info = ''
  wForm.user_remark = ''
  withdrawVisible.value = true
}

async function applyWithdraw() {
  saving.value = true
  try {
    await http.post('/withdrawals', wForm)
    ElMessage.success('已提交提现申请')
    withdrawVisible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
