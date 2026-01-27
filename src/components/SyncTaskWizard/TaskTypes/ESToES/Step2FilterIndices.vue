<template>
  <n-space vertical :size="16">
    <n-alert type="info">
      输入索引搜索条件，支持通配符（如 logs-*）。每个搜索条件会匹配一组索引。
    </n-alert>

    <n-space vertical :size="12">
      <n-space v-for="(group, index) in localValue" :key="index" align="center">
        <n-input
          v-model:value="group.pattern"
          placeholder="输入索引模式，如 logs-*"
          style="flex: 1"
        />
        <n-button @click="handleSearch(index)" :loading="group.loading">
          搜索 ({{ group.matchedIndices.length }})
        </n-button>
        <n-button @click="removeGroup(index)" type="error" secondary>
          删除
        </n-button>
      </n-space>

      <n-button @click="addGroup" dashed block>
        + 添加搜索条件
      </n-button>
    </n-space>

    <!-- 匹配结果预览 -->
    <n-card v-if="totalMatched > 0" title="匹配结果预览" size="small">
      <n-space vertical :size="8">
        <div v-for="(group, index) in localValue" :key="index">
          <n-text strong>{{ group.pattern }}</n-text>
          <n-text depth="3"> ({{ group.matchedIndices.length }} 个索引)</n-text>
          <n-scrollbar style="max-height: 100px">
            <n-space :size="4">
              <n-tag v-for="idx in group.matchedIndices.slice(0, 10)" :key="idx" size="small">
                {{ idx }}
              </n-tag>
              <n-text v-if="group.matchedIndices.length > 10" depth="3">
                ... 还有 {{ group.matchedIndices.length - 10 }} 个
              </n-text>
            </n-space>
          </n-scrollbar>
        </div>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { NSpace, NAlert, NInput, NButton, NCard, NText, NScrollbar, NTag } from 'naive-ui'
import { useSyncTaskStore } from '../../../../stores/syncTask'
import { handleApiError } from '../../../../utils/message'
import type { ESSearchGroup } from '../../../../types'

const props = defineProps<{
  sourceId: string
  modelValue: ESSearchGroup[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ESSearchGroup[]]
}>()

const syncTaskStore = useSyncTaskStore()

const localValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const totalMatched = computed(() => {
  return localValue.value.reduce((sum, group) => sum + group.matchedIndices.length, 0)
})

function addGroup() {
  const newGroups = [
    ...localValue.value,
    {
      pattern: '',
      matchedIndices: [],
      loading: false
    }
  ]
  localValue.value = newGroups
}

function removeGroup(index: number) {
  const newGroups = localValue.value.filter((_, i) => i !== index)
  localValue.value = newGroups
}

async function handleSearch(index: number) {
  const group = localValue.value[index]
  if (!group.pattern.trim()) return

  // 设置加载状态
  const newGroups = [...localValue.value]
  newGroups[index] = { ...group, loading: true }
  localValue.value = newGroups

  try {
    const result = await syncTaskStore.matchIndices(props.sourceId, group.pattern)
    
    // 更新匹配结果
    const updatedGroups = [...localValue.value]
    updatedGroups[index] = {
      ...group,
      matchedIndices: result.preview || [],
      loading: false
    }
    localValue.value = updatedGroups
  } catch (error) {
    handleApiError(error, '搜索索引失败')
    
    // 清除加载状态
    const updatedGroups = [...localValue.value]
    updatedGroups[index] = { ...group, loading: false }
    localValue.value = updatedGroups
  }
}

// 初始化：如果没有搜索组，添加一个空组
if (localValue.value.length === 0) {
  addGroup()
}
</script>
