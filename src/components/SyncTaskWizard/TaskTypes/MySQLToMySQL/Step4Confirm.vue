<template>
  <n-space vertical :size="16">
    <n-alert type="info" title="配置确认">
      请确认以下配置信息无误后提交
    </n-alert>

    <n-descriptions bordered :column="2">
      <n-descriptions-item label="任务名称">{{ preview.taskName }}</n-descriptions-item>
      <n-descriptions-item label="选择表数">{{ preview.tableCount }}</n-descriptions-item>
      <n-descriptions-item label="并发线程">{{ preview.threadCount }}</n-descriptions-item>
      <n-descriptions-item label="批量大小">{{ preview.batchSize }}</n-descriptions-item>
    </n-descriptions>

    <n-divider>选择的表</n-divider>
    
    <n-scrollbar style="max-height: 200px">
      <n-space>
        <n-tag v-for="table in selectedTables" :key="table" type="info">{{ table }}</n-tag>
      </n-space>
    </n-scrollbar>
  </n-space>
</template>

<script setup lang="ts">
import { NSpace, NAlert, NDescriptions, NDescriptionsItem, NDivider, NScrollbar, NTag } from 'naive-ui'

defineProps<{
  preview: {
    taskName: string
    tableCount: number
    threadCount: number
    batchSize: number
  }
  selectedTables: string[]
}>()
</script>
