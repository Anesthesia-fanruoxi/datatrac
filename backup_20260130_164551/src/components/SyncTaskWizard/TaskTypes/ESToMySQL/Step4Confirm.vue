<template>
  <n-space vertical :size="16">
    <n-alert type="info" title="配置确认">
      请确认以下配置信息无误后提交
    </n-alert>

    <n-descriptions bordered :column="2">
      <n-descriptions-item label="任务名称">{{ preview.taskName }}</n-descriptions-item>
      <n-descriptions-item label="索引数量">{{ preview.indexCount }}</n-descriptions-item>
      <n-descriptions-item label="并发线程">{{ preview.threadCount }}</n-descriptions-item>
      <n-descriptions-item label="批量大小">{{ preview.batchSize }}</n-descriptions-item>
    </n-descriptions>

    <n-divider>匹配的索引</n-divider>
    
    <n-scrollbar style="max-height: 200px">
      <n-space>
        <n-tag v-for="index in matchedIndices" :key="index" type="info">{{ index }}</n-tag>
      </n-space>
    </n-scrollbar>
  </n-space>
</template>

<script setup lang="ts">
import { NSpace, NAlert, NDescriptions, NDescriptionsItem, NDivider, NScrollbar, NTag } from 'naive-ui'

defineProps<{
  preview: {
    taskName: string
    indexCount: number
    threadCount: number
    batchSize: number
  }
  matchedIndices: string[]
}>()
</script>
