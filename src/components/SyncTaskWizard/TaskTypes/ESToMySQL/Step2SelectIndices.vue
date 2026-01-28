<template>
  <n-space vertical :size="16">
    <n-form-item label="索引匹配模式">
      <n-input
        v-model:value="localPattern"
        placeholder="输入索引名或通配符，如 logs-*"
        @blur="handleMatch"
      >
        <template #suffix>
          <n-button text @click="handleMatch">匹配</n-button>
        </template>
      </n-input>
    </n-form-item>

    <n-alert v-if="matchedIndices.length > 0" type="success">
      匹配到 {{ matchedIndices.length }} 个索引
    </n-alert>

    <n-scrollbar v-if="matchedIndices.length > 0" style="max-height: 300px">
      <n-space>
        <n-tag v-for="index in matchedIndices" :key="index" type="info">{{ index }}</n-tag>
      </n-space>
    </n-scrollbar>
  </n-space>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NSpace, NFormItem, NInput, NButton, NAlert, NScrollbar, NTag } from 'naive-ui'
import { useSyncTaskStore } from '../../../../stores/syncTask'

const props = defineProps<{
  sourceId: string
  pattern: string
  matchedIndices: string[]
}>()

const emit = defineEmits<{
  'update:pattern': [value: string]
  'update:matchedIndices': [value: string[]]
}>()

const syncTaskStore = useSyncTaskStore()

const localPattern = computed({
  get: () => props.pattern,
  set: (val) => emit('update:pattern', val)
})

const matchedIndices = computed({
  get: () => props.matchedIndices,
  set: (val) => emit('update:matchedIndices', val)
})

async function handleMatch() {
  if (!localPattern.value) return
  
  try {
    const result = await syncTaskStore.matchIndices(props.sourceId, localPattern.value)
    matchedIndices.value = result.preview
  } catch (error) {
    console.error('匹配索引失败:', error)
  }
}
</script>
