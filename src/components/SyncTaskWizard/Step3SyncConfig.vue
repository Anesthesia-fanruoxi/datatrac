<template>
  <n-form :model="formData" label-placement="left" label-width="140">
    <n-form-item label="线程数" path="syncConfig.threadCount">
      <n-input-number
        v-model:value="formData.syncConfig!.threadCount"
        :min="1"
        :max="32"
        style="width: 100%"
      />
    </n-form-item>

    <n-form-item label="批量大小" path="syncConfig.batchSize">
      <n-input-number
        v-model:value="formData.syncConfig!.batchSize"
        :min="100"
        :max="10000"
        style="width: 100%"
      />
    </n-form-item>

    <n-form-item label="错误策略" path="syncConfig.errorStrategy">
      <n-radio-group v-model:value="formData.syncConfig!.errorStrategy">
        <n-space>
          <n-radio value="skip">跳过错误</n-radio>
          <n-radio value="pause">遇错暂停</n-radio>
        </n-space>
      </n-radio-group>
    </n-form-item>

    <n-divider />

    <n-form-item label="目标表存在策略" path="syncConfig.tableExistsStrategy">
      <n-radio-group v-model:value="formData.syncConfig!.tableExistsStrategy">
        <n-space vertical>
          <n-radio value="drop">
            <span>删除重建</span>
            <span style="margin-left: 8px; color: #999; font-size: 12px;">删除原表/索引后重新创建</span>
          </n-radio>
          <n-radio value="truncate">
            <span>清空数据</span>
            <span style="margin-left: 8px; color: #999; font-size: 12px;">保留表结构，仅清空数据</span>
          </n-radio>
          <n-radio value="backup">
            <span>备份后重建</span>
            <span style="margin-left: 8px; color: #999; font-size: 12px;">备份原表/索引后重新创建</span>
          </n-radio>
        </n-space>
      </n-radio-group>
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { NForm, NFormItem, NInputNumber, NRadioGroup, NRadio, NSpace, NDivider } from 'naive-ui'
import type { SyncTask } from '../../types'

defineProps<{
  formData: Partial<SyncTask> & { targetDatabase?: string }
  targetType: 'mysql' | 'elasticsearch'
}>()
</script>
