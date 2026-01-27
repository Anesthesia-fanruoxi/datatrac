<template>
  <n-space vertical :size="16">
    <n-alert type="info">
      从搜索结果中选择要同步的索引，可以配置索引名称转换规则。
    </n-alert>

    <!-- 索引选择 -->
    <n-card title="选择索引" size="small">
      <n-scrollbar style="max-height: 400px">
        <n-checkbox-group v-model:value="localSelectedIndices">
          <n-space vertical>
            <div v-for="group in searchGroups" :key="group.pattern">
              <n-divider>{{ group.pattern }} ({{ group.matchedIndices.length }})</n-divider>
              <n-space>
                <n-checkbox
                  v-for="index in group.matchedIndices"
                  :key="index"
                  :value="index"
                  :label="index"
                />
              </n-space>
            </div>
          </n-space>
        </n-checkbox-group>
      </n-scrollbar>

      <template #footer>
        <n-space justify="space-between">
          <n-text>已选择 {{ localSelectedIndices.length }} 个索引</n-text>
          <n-space>
            <n-button size="small" @click="selectAll">全选</n-button>
            <n-button size="small" @click="clearSelection">清空</n-button>
          </n-space>
        </n-space>
      </template>
    </n-card>

    <!-- 索引名称转换 -->
    <n-card title="索引名称转换（可选）" size="small">
      <n-space vertical :size="12">
        <n-checkbox v-model:checked="localTransform.enabled">
          启用索引名称转换
        </n-checkbox>

        <template v-if="localTransform.enabled">
          <n-form-item label="转换模式">
            <n-radio-group v-model:value="localTransform.mode">
              <n-space>
                <n-radio value="prefix">前缀替换</n-radio>
                <n-radio value="suffix">后缀替换</n-radio>
              </n-space>
            </n-radio-group>
          </n-form-item>

          <n-form-item label="源模式">
            <n-input
              v-model:value="localTransform.sourcePattern"
              placeholder="例如：logs-"
            />
          </n-form-item>

          <n-form-item label="目标模式">
            <n-input
              v-model:value="localTransform.targetPattern"
              placeholder="例如：backup-logs-"
            />
          </n-form-item>

          <!-- 转换预览 -->
          <n-alert v-if="transformPreview" type="success" title="转换预览">
            {{ transformPreview }}
          </n-alert>
        </template>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  NSpace,
  NAlert,
  NCard,
  NScrollbar,
  NCheckboxGroup,
  NCheckbox,
  NDivider,
  NText,
  NButton,
  NFormItem,
  NRadioGroup,
  NRadio,
  NInput
} from 'naive-ui'
import type { ESSearchGroup, IndexNameTransform } from '../../../../types'

const props = defineProps<{
  searchGroups: ESSearchGroup[]
  modelValue: string[]
  indexNameTransform: IndexNameTransform
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  'update:indexNameTransform': [value: IndexNameTransform]
}>()

const localSelectedIndices = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const localTransform = ref<IndexNameTransform>({ ...props.indexNameTransform })

// 监听转换配置变化，同步到父组件
watch(localTransform, (newVal) => {
  emit('update:indexNameTransform', newVal)
}, { deep: true })

// 转换预览
const transformPreview = computed(() => {
  if (!localTransform.value.enabled || !localTransform.value.sourcePattern || !localTransform.value.targetPattern) {
    return ''
  }

  const example = localSelectedIndices.value[0] || 'logs-2024-01'
  const { mode, sourcePattern, targetPattern } = localTransform.value

  if (mode === 'prefix') {
    if (example.startsWith(sourcePattern)) {
      return `${example} → ${example.replace(sourcePattern, targetPattern)}`
    }
  } else if (mode === 'suffix') {
    if (example.endsWith(sourcePattern)) {
      return `${example} → ${example.replace(new RegExp(sourcePattern + '$'), targetPattern)}`
    }
  }

  return `示例：${example} → (无匹配)`
})

function selectAll() {
  const allIndices = props.searchGroups.flatMap(group => group.matchedIndices)
  localSelectedIndices.value = allIndices
}

function clearSelection() {
  localSelectedIndices.value = []
}
</script>
