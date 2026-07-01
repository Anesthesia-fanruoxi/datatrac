<template>
  <section class="tm-card log-panel">
    <header class="tm-card-header">
      <h3>任务日志</h3>
      <div class="log-tabs">
        <button v-for="item in categories" :key="item" class="log-tab" :class="{ active: modelValue === item }" @click="$emit('update:modelValue', item)">
          {{ labels[item] }}
        </button>
      </div>
    </header>
    <div ref="logBodyRef" class="log-body">
      <div v-if="logs.length === 0" class="empty-state dark">暂无日志</div>
      <div v-for="(log, index) in logs" :key="`${index}-${log.Time || log.time || ''}`" class="log-line">
        <span class="time">{{ log.Time || log.time || '-' }}</span>
        <span>{{ log.Message || log.message || '(空消息)' }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { LogCategory, TaskLogEntry } from '../../types/task-monitor'

const props = defineProps<{ logs: TaskLogEntry[]; modelValue: LogCategory }>()
defineEmits<{ 'update:modelValue': [value: LogCategory] }>()

const logBodyRef = ref<HTMLElement | null>(null)

const categories: LogCategory[] = ['all', 'initialize', 'complete']
const labels: Record<LogCategory, string> = {
  all: '全部',
  initialize: '初始化',
  complete: '完成'
}

watch(
  () => props.logs.length,
  async () => {
    await nextTick()
    if (logBodyRef.value) {
      logBodyRef.value.scrollTop = logBodyRef.value.scrollHeight
    }
  }
)
</script>
