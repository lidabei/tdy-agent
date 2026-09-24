<template>
  <el-popover placement="bottom-end" :width="360" trigger="click" @show="load">
    <template #reference>
      <el-badge :value="unread || ''" :hidden="!unread" class="bell">
        <el-button link>通知</el-button>
      </el-badge>
    </template>
    <div class="head row">
      <strong>通知</strong>
      <div class="spacer" />
      <el-button link type="primary" @click="readAll">全部已读</el-button>
    </div>
    <div v-if="!list.length" class="muted" style="padding:12px 0">暂无通知</div>
    <div v-for="n in list" :key="n.id" class="item" :class="{ unread: !n.read }" @click="open(n)">
      <div class="t">{{ n.title }}</div>
      <div class="muted c">{{ n.content }}</div>
    </div>
  </el-popover>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api'

const router = useRouter()
const list = ref([])
const unread = ref(0)
let timer

async function refreshCount() {
  try {
    const { data } = await http.get('/notifications/unread-count')
    unread.value = data.unread || 0
  } catch {}
}

async function load() {
  try {
    const { data } = await http.get('/notifications', { params: { page: 1, size: 20 } })
    list.value = data.list || []
    unread.value = data.unread || 0
  } catch {}
}

async function open(n) {
  if (!n.read) await http.post(`/notifications/${n.id}/read`)
  if (n.link) router.push(n.link)
  load()
}

async function readAll() {
  await http.post('/notifications/read-all')
  load()
}

onMounted(() => {
  refreshCount()
  timer = setInterval(refreshCount, 30000)
})
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.item { padding: 8px 4px; border-bottom: 1px solid var(--line); cursor: pointer; }
.item.unread .t { font-weight: 700; color: var(--accent); }
.c { font-size: 12px; margin-top: 2px; }
.head { margin-bottom: 8px; }
</style>
