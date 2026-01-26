<template>
  <n-card title="配置预览" :bordered="false">
    <n-descriptions :column="2" bordered>
      <n-descriptions-item label="任务名称">
        {{ preview.taskName }}
      </n-descriptions-item>
      <n-descriptions-item label="数据信息">
        {{ preview.databaseInfo }}
      </n-descriptions-item>
      <n-descriptions-item label="源数据源">
        {{ preview.sourceName }} ({{ preview.sourceType }})
      </n-descriptions-item>
      <n-descriptions-item label="目标数据源">
        {{ preview.targetName }} ({{ preview.targetType }})
      </n-descriptions-item>
      <n-descriptions-item label="线程数">
        {{ preview.threadCount }}
      </n-descriptions-item>
      <n-descriptions-item label="批量大小">
        {{ preview.batchSize }}
      </n-descriptions-item>
      <n-descriptions-item label="错误策略" :span="2">
        {{ preview.errorStrategy }}
      </n-descriptions-item>
    </n-descriptions>

    <!-- 详细表列表 -->
    <n-divider />
    <div v-if="sourceType === 'mysql' && selectedTables.length > 0">
      <h4>选择的表：</h4>
      <n-space>
        <n-tag
          v-for="table in selectedTables"
          :key="table"
          type="info"
          size="small"
        >
          {{ table }}
        </n-tag>
      </n-space>
    </div>
  </n-card>
</template>

<script setup lang="ts">
import { NCard, NDescriptions, NDescriptionsItem, NDivider, NSpace, NTag } from 'naive-ui'

defineProps<{
  preview: {
    taskName: string
    sourceName: string
    sourceType: string
    targetName: string
    targetType: string
    databaseInfo: string
    threadCount: number
    batchSize: number
    errorStrategy: string
  }
  sourceType: 'mysql' | 'elasticsearch'
  selectedTables: string[]
}>()
</script>
