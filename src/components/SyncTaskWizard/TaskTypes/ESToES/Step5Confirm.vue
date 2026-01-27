<template>
  <n-space vertical :size="16">
    <n-alert type="success" title="任务配置完成">
      请确认以下配置信息，确认无误后点击"保存任务"按钮。
    </n-alert>

    <!-- 任务信息 -->
    <n-card title="任务信息" size="small">
      <n-descriptions :column="2" bordered>
        <n-descriptions-item label="任务名称">
          {{ preview.taskName }}
        </n-descriptions-item>
        <n-descriptions-item label="搜索条件组数">
          {{ preview.searchGroupsCount }}
        </n-descriptions-item>
        <n-descriptions-item label="选择的索引数">
          {{ preview.selectedIndicesCount }}
        </n-descriptions-item>
        <n-descriptions-item label="线程数">
          {{ preview.threadCount }}
        </n-descriptions-item>
        <n-descriptions-item label="批量大小">
          {{ preview.batchSize }}
        </n-descriptions-item>
      </n-descriptions>
    </n-card>

    <!-- 选择的索引列表 -->
    <n-card title="选择的索引" size="small">
      <n-scrollbar style="max-height: 300px">
        <n-space :size="8">
          <n-tag v-for="index in selectedIndices" :key="index" type="info">
            {{ index }}
          </n-tag>
        </n-space>
      </n-scrollbar>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { NSpace, NAlert, NCard, NDescriptions, NDescriptionsItem, NScrollbar, NTag } from 'naive-ui'

defineProps<{
  preview: {
    taskName: string
    searchGroupsCount: number
    selectedIndicesCount: number
    threadCount: number
    batchSize: number
  }
  selectedIndices: string[]
}>()
</script>
