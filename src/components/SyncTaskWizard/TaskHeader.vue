<template>
  <div class="task-header">
    <n-space align="center" :size="16">
      <!-- 任务名称（可编辑） -->
      <div class="task-name-section">
        <n-input
          v-if="isEditingName"
          v-model:value="localName"
          size="large"
          @blur="handleNameBlur"
          @keyup.enter="handleNameBlur"
          ref="nameInputRef"
          style="width: 300px"
        />
        <n-space v-else align="center" :size="8">
          <n-text strong style="font-size: 18px">{{ taskName }}</n-text>
          <n-button text @click="startEditName">
            <template #icon>
              <n-icon><EditIcon /></n-icon>
            </template>
          </n-button>
        </n-space>
      </div>

      <n-divider vertical />

      <!-- 数据源类型（不可编辑） -->
      <n-space align="center" :size="12">
        <div class="type-display">
          <n-icon :size="32" :color="getSourceColor()">
            <component :is="getSourceIcon()" />
          </n-icon>
          <n-text depth="3" style="font-size: 12px">{{ getSourceLabel() }}</n-text>
        </div>

        <n-icon :size="24" color="#999">
          <ArrowForwardIcon />
        </n-icon>

        <div class="type-display">
          <n-icon :size="32" :color="getTargetColor()">
            <component :is="getTargetIcon()" />
          </n-icon>
          <n-text depth="3" style="font-size: 12px">{{ getTargetLabel() }}</n-text>
        </div>
      </n-space>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { NSpace, NText, NInput, NButton, NIcon, NDivider } from 'naive-ui'
import { Create as EditIcon, Server as ServerIcon, Cloud as CloudIcon, ArrowForward as ArrowForwardIcon } from '@vicons/ionicons5'
import type { DataSourceType } from '../../types'

const props = defineProps<{
  taskName: string
  sourceType: DataSourceType
  targetType: DataSourceType
}>()

const emit = defineEmits<{
  'update:taskName': [value: string]
}>()

const isEditingName = ref(false)
const localName = ref(props.taskName)
const nameInputRef = ref()

function startEditName() {
  localName.value = props.taskName
  isEditingName.value = true
  nextTick(() => {
    nameInputRef.value?.focus()
  })
}

function handleNameBlur() {
  if (localName.value.trim()) {
    emit('update:taskName', localName.value.trim())
  } else {
    localName.value = props.taskName
  }
  isEditingName.value = false
}

function getSourceIcon() {
  return props.sourceType === 'mysql' ? ServerIcon : CloudIcon
}

function getTargetIcon() {
  return props.targetType === 'mysql' ? ServerIcon : CloudIcon
}

function getSourceLabel() {
  return props.sourceType === 'mysql' ? 'MySQL' : 'Elasticsearch'
}

function getTargetLabel() {
  return props.targetType === 'mysql' ? 'MySQL' : 'Elasticsearch'
}

function getSourceColor() {
  return props.sourceType === 'mysql' ? '#18a058' : '#2080f0'
}

function getTargetColor() {
  return props.targetType === 'mysql' ? '#18a058' : '#2080f0'
}
</script>

<style scoped>
.task-header {
  padding: 16px 24px;
  background: #fafafa;
  border-radius: 4px;
  margin-bottom: 24px;
}

.task-name-section {
  min-width: 200px;
}

.type-display {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}
</style>
