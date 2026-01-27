<template>
  <n-space vertical :size="16">
    <n-form-item :label="sourceLabel" required>
      <n-select
        v-model:value="localSourceId"
        :options="sourceOptions"
        placeholder="请选择源数据源"
        filterable
      />
    </n-form-item>

    <n-form-item :label="targetLabel" required>
      <n-select
        v-model:value="localTargetId"
        :options="targetOptions"
        placeholder="请选择目标数据源"
        filterable
      />
    </n-form-item>
  </n-space>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NSpace, NFormItem, NSelect } from 'naive-ui'
import { useDataSourceStore } from '../../../stores/dataSource'
import type { DataSourceType } from '../../../types'

const props = defineProps<{
  sourceId?: string
  targetId?: string
  sourceType: DataSourceType
  targetType: DataSourceType
  sourceLabel?: string
  targetLabel?: string
}>()

const emit = defineEmits<{
  'update:sourceId': [value: string]
  'update:targetId': [value: string]
}>()

const dataSourceStore = useDataSourceStore()

const localSourceId = computed({
  get: () => props.sourceId,
  set: (val) => emit('update:sourceId', val || '')
})

const localTargetId = computed({
  get: () => props.targetId,
  set: (val) => emit('update:targetId', val || '')
})

// 源数据源选项（根据类型过滤）
const sourceOptions = computed(() => {
  return dataSourceStore.dataSources
    .filter(ds => ds.type === props.sourceType)
    .map(ds => ({
      label: ds.name,
      value: ds.id
    }))
})

// 目标数据源选项（根据类型过滤，排除已选择的源）
const targetOptions = computed(() => {
  return dataSourceStore.dataSources
    .filter(ds => ds.type === props.targetType && ds.id !== props.sourceId)
    .map(ds => ({
      label: ds.name,
      value: ds.id
    }))
})
</script>
