<template>
  <n-form label-placement="left" label-width="120">
    <n-form-item label="并发线程数" required>
      <n-input-number
        v-model:value="localConfig.threadCount"
        :min="1"
        :max="32"
        style="width: 100%"
        placeholder="1-32"
      >
        <template #suffix>
          <n-text depth="3" style="font-size: 12px">个线程</n-text>
        </template>
      </n-input-number>
      <template #feedback>
        <n-text depth="3" style="font-size: 12px">
          建议范围：1-32，默认4。线程越多速度越快，但占用资源越多
        </n-text>
      </template>
    </n-form-item>

    <n-form-item label="批量大小" required>
      <n-input-number
        v-model:value="localConfig.batchSize"
        :min="100"
        :max="10000"
        :step="100"
        style="width: 100%"
        placeholder="100-10000"
      >
        <template #suffix>
          <n-text depth="3" style="font-size: 12px">条/批次</n-text>
        </template>
      </n-input-number>
      <template #feedback>
        <n-text depth="3" style="font-size: 12px">
          建议范围：100-10000，默认2500。批量越大速度越快，但占用内存越多
        </n-text>
      </template>
    </n-form-item>

    <n-form-item label="错误处理策略" required>
      <n-radio-group v-model:value="localConfig.errorStrategy">
        <n-space>
          <n-radio value="skip">跳过错误继续</n-radio>
          <n-radio value="pause">遇错暂停</n-radio>
        </n-space>
      </n-radio-group>
    </n-form-item>

    <n-form-item label="目标表存在时" required>
      <n-radio-group v-model:value="localConfig.tableExistsStrategy">
        <n-space vertical>
          <n-radio value="drop">删除重建</n-radio>
          <n-radio value="truncate">清空数据</n-radio>
          <n-radio value="backup">备份后重建</n-radio>
        </n-space>
      </n-radio-group>
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NForm, NFormItem, NInputNumber, NRadioGroup, NRadio, NSpace, NText } from 'naive-ui'
import type { SyncConfig } from '../../../types'

const props = defineProps<{
  syncConfig: SyncConfig
}>()

const emit = defineEmits<{
  'update:syncConfig': [value: SyncConfig]
}>()

const localConfig = computed({
  get: () => ({
    threadCount: props.syncConfig.threadCount,
    batchSize: props.syncConfig.batchSize,
    errorStrategy: props.syncConfig.errorStrategy,
    tableExistsStrategy: props.syncConfig.tableExistsStrategy
  }),
  set: (val) => {
    emit('update:syncConfig', {
      threadCount: val.threadCount,
      batchSize: val.batchSize,
      errorStrategy: val.errorStrategy,
      tableExistsStrategy: val.tableExistsStrategy
    })
  }
})
</script>
