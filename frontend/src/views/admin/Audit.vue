<template>
  <div>
    <div class="row" style="margin-bottom:14px">
      <h2 style="margin:0">操作审计</h2>
      <div class="spacer" />
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="card">
      <el-table :data="list" stripe v-loading="loading">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="actor_name" label="操作人" width="120" />
        <el-table-column prop="action" label="动作" width="160" />
        <el-table-column prop="target_type" label="对象" width="100" />
        <el-table-column prop="target_id" label="对象ID" width="90" />
        <el-table-column prop="detail" label="详情" min-width="180" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="130" />
        <el-table-column prop="created_at" label="时间" min-width="160" />
      </el-table>
      <div class="row" style="margin-top:12px;justify-content:flex-end">
        <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page" @current-change="(p)=>{page=p;load()}" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../../api'

const list = ref([])
const loading = ref(false)
const page = ref(1)
const size = ref(20)
const total = ref(0)

async function load() {
  loading.value = true
  try {
    const { data } = await http.get('/admin/audit-logs', { params: { page: page.value, size: size.value } })
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
