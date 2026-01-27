<template>
  <n-space vertical :size="24">
    <!-- 任务名称 -->
    <n-form-item label="任务名称" required>
      <n-input
        :value="taskName"
        @update:value="$emit('update:taskName', $event)"
        placeholder="请输入任务名称"
        clearable
      />
    </n-form-item>

    <!-- 数据源类型选择 - 左右布局 -->
    <n-space :size="24">
      <n-form-item label="源数据源类型" required style="flex: 1">
        <n-button
          :type="sourceType ? 'primary' : 'default'"
          size="large"
          block
          :disabled="isEdit"
          @click="!isEdit && (showSourceSelector = true)"
        >
          <template #icon>
            <n-icon v-if="sourceType">
              <component :is="getIcon(sourceType)" />
            </n-icon>
          </template>
          {{ sourceType ? getLabel(sourceType) : '点击选择源数据源类型' }}
        </n-button>
      </n-form-item>

      <div style="display: flex; align-items: center; padding-top: 34px">
        <n-icon :size="32" color="#999">
          <ArrowForwardIcon />
        </n-icon>
      </div>

      <n-form-item label="目标数据源类型" required style="flex: 1">
        <n-button
          :type="targetType ? 'primary' : 'default'"
          size="large"
          block
          :disabled="isEdit"
          @click="!isEdit && (showTargetSelector = true)"
        >
          <template #icon>
            <n-icon v-if="targetType">
              <component :is="getIcon(targetType)" />
            </n-icon>
          </template>
          {{ targetType ? getLabel(targetType) : '点击选择目标数据源类型' }}
        </n-button>
      </n-form-item>
    </n-space>

    <!-- 任务类型提示 -->
    <n-alert v-if="sourceType && targetType" type="success">
      <template #icon>
        <n-icon><CheckmarkCircleIcon /></n-icon>
      </template>
      任务类型：{{ getTaskTypeLabel() }}
      <span v-if="isEdit" style="margin-left: 8px; color: #666;">（已锁定，不可修改）</span>
    </n-alert>
  </n-space>

  <!-- 源数据源类型选择对话框 -->
  <DataSourceTypeSelector
    v-model="showSourceSelector"
    title="选择源数据源类型"
    :selected-type="sourceType"
    @select="handleSourceTypeSelect"
  />

  <!-- 目标数据源类型选择对话框 -->
  <DataSourceTypeSelector
    v-model="showTargetSelector"
    title="选择目标数据源类型"
    :selected-type="targetType"
    @select="handleTargetTypeSelect"
  />
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { NSpace, NFormItem, NInput, NButton, NIcon, NAlert } from 'naive-ui'
import { Server as ServerIcon, Cloud as CloudIcon, ArrowForward as ArrowForwardIcon, CheckmarkCircle as CheckmarkCircleIcon } from '@vicons/ionicons5'
import DataSourceTypeSelector from './DataSourceTypeSelector.vue'
import type { DataSourceType } from '../../types'

const props = defineProps<{
  taskName: string
  sourceType?: DataSourceType
  targetType?: DataSourceType
  isEdit?: boolean
}>()

const emit = defineEmits<{
  'update:taskName': [value: string]
  'update:sourceType': [value: DataSourceType]
  'update:targetType': [value: DataSourceType]
}>()

const showSourceSelector = ref(false)
const showTargetSelector = ref(false)

function handleSourceTypeSelect(type: DataSourceType) {
  emit('update:sourceType', type)
}

function handleTargetTypeSelect(type: DataSourceType) {
  emit('update:targetType', type)
}

function getIcon(type: DataSourceType) {
  return type === 'mysql' ? ServerIcon : CloudIcon
}

function getLabel(type: DataSourceType) {
  return type === 'mysql' ? 'MySQL' : 'Elasticsearch'
}

function getTaskTypeLabel() {
  if (!props.sourceType || !props.targetType) return ''
  
  const labels: Record<string, string> = {
    'mysql-mysql': 'MySQL → MySQL 数据同步',
    'mysql-elasticsearch': 'MySQL → Elasticsearch 数据同步',
    'elasticsearch-mysql': 'Elasticsearch → MySQL 数据同步',
    'elasticsearch-elasticsearch': 'Elasticsearch → Elasticsearch 索引同步'
  }
  
  return labels[`${props.sourceType}-${props.targetType}`] || ''
}
</script>
