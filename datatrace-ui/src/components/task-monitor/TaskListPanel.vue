<template>
  <section class="tm-card tm-list-panel">
    <header class="tm-card-header gradient">
      <h2>任务执行监控</h2>
    </header>
    <TaskMonitorTabs :model-value="selectedTab" @update:model-value="onSwitchTab" />
    <div class="tm-card-body task-grid-wrap">
      <div v-if="loading" class="empty-state">加载中...</div>
      <div v-else-if="tasks.length === 0" class="empty-state">暂无任务</div>
      <div v-else class="task-grid">
        <article
          v-for="task in tasks"
          :key="task.id"
          class="task-card"
          :class="{ active: currentTaskId === task.id }"
          @click="$emit('select', task.id)"
        >
          <div class="task-card-title">{{ task.name }}</div>
          <div class="task-card-meta">
            <el-tag size="small" :type="tagType(task)">{{ statusText(task) }}</el-tag>
            <span>{{ task.source_type }} → {{ task.target_type }}</span>
          </div>
          <div class="task-card-actions">
            <el-button v-if="!task.is_running" size="small" type="success" plain @click.stop="$emit('start', task.id)">启动</el-button>
            <el-button v-if="task.is_running" size="small" type="warning" plain @click.stop="$emit('pause', task.id)">暂停</el-button>
            <el-button v-if="task.is_running" size="small" type="danger" plain @click.stop="$emit('stop', task.id)">停止</el-button>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { TaskItem } from '../../types/task-monitor'
import TaskMonitorTabs from './TaskMonitorTabs.vue'

const props = defineProps<{
  tasks: TaskItem[]
  loading: boolean
  selectedTab: 'full' | 'incremental' | 'structure'
  currentTaskId?: string
}>()

const emit = defineEmits<{
  select: [taskId: string]
  start: [taskId: string]
  pause: [taskId: string]
  stop: [taskId: string]
  tabChange: [tab: 'full' | 'incremental' | 'structure']
}>()

function onSwitchTab(tab: 'full' | 'incremental' | 'structure') {
  emit('tabChange', tab)
}

function statusText(task: TaskItem) {
  if (task.is_running && task.current_step === 'initialize') return '初始化'
  if (task.is_running && task.current_step === 'sync_data') return '全量同步'
  if (task.is_running && task.current_step === 'incremental') return '增量同步'
  if (!task.is_running && (task.status === 'completed' || task.current_step === 'completed')) return '已完成'
  return '未运行'
}

function tagType(task: TaskItem) {
  if (task.is_running && task.current_step === 'incremental') return 'warning'
  if (task.is_running) return 'primary'
  if (!task.is_running && (task.status === 'completed' || task.current_step === 'completed')) return 'success'
  return 'info'
}
</script>
