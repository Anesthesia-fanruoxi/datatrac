<template>
  <div class="stepper">
    <div v-for="step in stepStates" :key="step.key" class="step-item" :class="step.className">
      <div class="step-dot">{{ step.symbol }}</div>
      <div class="step-label">{{ step.name }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  syncMode: string
  currentStep?: string
  taskStatus?: string
  isRunning?: boolean
}>()

const stepStates = computed(() => {
  const steps = [
    { key: 'initialize', name: '初始化' },
    { key: 'sync_data', name: '全量同步' },
    { key: props.syncMode === 'incremental' ? 'incremental' : 'completed', name: props.syncMode === 'incremental' ? '增量同步' : '完成' }
  ]

  const currentIndex = steps.findIndex((item) => item.key === props.currentStep)
  const done = props.taskStatus === 'completed'

  return steps.map((step, index) => {
    const active = props.isRunning && props.currentStep === step.key
    const completed = done ? true : props.syncMode !== 'incremental' && index < currentIndex
    return {
      ...step,
      className: active ? 'active' : completed ? 'completed' : '',
      symbol: active ? '•' : completed ? '✓' : String(index + 1)
    }
  })
})
</script>
