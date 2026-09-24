<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">仪表盘</h2>
      <div class="spacer" />
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="stats" v-loading="loading">
      <div
        class="stat card"
        v-for="s in cards"
        :key="s.label"
        :style="s.to ? 'cursor:pointer' : ''"
        @click="s.to && $router.push(s.to)"
      >
        <div class="muted">{{ s.label }}</div>
        <div class="num" :class="{ money: s.money }">{{ s.value }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http, { fenToYuan } from '../../api'

const loading = ref(false)
const data = ref({})

const cards = computed(() => [
  { label: '待验收工单', value: data.value.pending_review ?? '-', to: '/admin/orders?status=submitted' },
  { label: '待审提现', value: data.value.pending_withdraw ?? '-', to: '/admin/withdrawals' },
  { label: '进行中工单', value: data.value.open_orders ?? '-', to: '/admin/orders' },
  { label: '超期工单', value: data.value.overdue_orders ?? '-', to: '/admin/orders?overdue=1' },
  { label: '启用用户', value: data.value.active_users ?? '-', to: '/admin/users' },
  { label: '本月入账', value: '¥' + fenToYuan(data.value.month_credit), money: true },
  { label: '本月扣减', value: '¥' + fenToYuan(data.value.month_debit), money: true },
])

async function load() {
  loading.value = true
  try {
    const { data: d } = await http.get('/admin/dashboard')
    data.value = d
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}
.stat .num {
  font-size: 28px;
  font-weight: 700;
  margin-top: 8px;
}
.stat:hover { border-color: var(--accent); }
</style>
