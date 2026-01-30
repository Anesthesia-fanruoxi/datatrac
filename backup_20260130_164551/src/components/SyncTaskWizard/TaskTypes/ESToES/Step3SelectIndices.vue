<template>
  <n-space vertical :size="16">
    <n-alert type="info">
      从搜索结果中选择要同步的索引，可以配置索引名称转换规则。
    </n-alert>

    <!-- 自定义穿梭框 -->
    <n-card title="选择索引" size="small">
      <div class="custom-transfer">
        <!-- 左侧：树形结构 -->
        <div class="transfer-panel left-panel">
          <div class="panel-header">
            <n-text>可选索引 ({{ totalIndices }})</n-text>
            <n-input
              v-model:value="leftSearchText"
              placeholder="搜索索引..."
              size="small"
              clearable
              style="margin-top: 8px"
            >
              <template #prefix>
                <n-icon :component="SearchOutline" />
              </template>
            </n-input>
          </div>
          <n-scrollbar class="panel-content">
            <div v-for="(group, index) in filteredGroups" :key="index" class="group-item">
              <div class="group-header">
                <n-button
                  text
                  size="tiny"
                  @click="toggleGroupCollapse(index)"
                  class="collapse-btn"
                >
                  <template #icon>
                    <n-icon :component="collapsedGroups.has(index) ? ChevronForwardOutline : ChevronDownOutline" />
                  </template>
                </n-button>
                <n-checkbox
                  :checked="isGroupAllSelected(group)"
                  :indeterminate="isGroupIndeterminate(group)"
                  @update:checked="(checked) => toggleGroup(group, checked)"
                >
                  <n-text strong style="color: #18a058">
                    {{ group.pattern }} ({{ group.matchedIndices.length }})
                  </n-text>
                </n-checkbox>
              </div>
              <div v-show="!collapsedGroups.has(index)" class="group-children">
                <n-checkbox
                  v-for="index in group.matchedIndices"
                  :key="index"
                  :checked="localSelectedIndices.includes(index)"
                  :label="index"
                  @update:checked="(checked) => toggleIndex(index, checked)"
                  class="index-checkbox"
                />
              </div>
            </div>
          </n-scrollbar>
          <div class="panel-footer">
            <n-text depth="3">{{ leftSelectedCount }} / {{ totalIndices }} 项</n-text>
          </div>
        </div>

        <!-- 中间：操作按钮 -->
        <div class="transfer-actions">
          <n-space vertical :size="8">
            <n-button
              size="small"
              @click="selectAll"
              :disabled="totalIndices === 0"
            >
              全选 →
            </n-button>
            <n-button
              size="small"
              @click="clearSelection"
              :disabled="localSelectedIndices.length === 0"
            >
              ← 清空
            </n-button>
          </n-space>
        </div>

        <!-- 右侧：已选列表 -->
        <div class="transfer-panel right-panel">
          <div class="panel-header">
            <n-text>已选索引 ({{ localSelectedIndices.length }})</n-text>
            <n-input
              v-model:value="rightSearchText"
              placeholder="搜索已选索引..."
              size="small"
              clearable
              style="margin-top: 8px"
            >
              <template #prefix>
                <n-icon :component="SearchOutline" />
              </template>
            </n-input>
          </div>
          <n-scrollbar class="panel-content">
            <div class="selected-list">
              <div
                v-for="index in filteredSelectedIndices"
                :key="index"
                class="selected-item"
              >
                <n-text>{{ index }}</n-text>
                <n-button
                  text
                  size="tiny"
                  @click="removeIndex(index)"
                  style="margin-left: auto"
                >
                  <template #icon>
                    <n-icon :component="CloseOutline" />
                  </template>
                </n-button>
              </div>
            </div>
          </n-scrollbar>
          <div class="panel-footer">
            <n-text depth="3">{{ localSelectedIndices.length }} 项</n-text>
          </div>
        </div>
      </div>
      
      <template #footer>
        <n-text>已选择 {{ localSelectedIndices.length }} 个索引</n-text>
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
  NText,
  NFormItem,
  NRadioGroup,
  NRadio,
  NInput,
  NCheckbox,
  NButton,
  NScrollbar,
  NIcon
} from 'naive-ui'
import { SearchOutline, CloseOutline, ChevronDownOutline, ChevronForwardOutline } from '@vicons/ionicons5'
import type { ESSearchGroup, IndexNameTransform } from '../../../../types'

const props = defineProps<{
  searchGroups: ESSearchGroup[]
  modelValue: string[]
  indexNameTransform: IndexNameTransform
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  'update:indexNameTransform': [value: IndexNameTransform]
  'update:searchGroups': [value: ESSearchGroup[]]
}>()

const leftSearchText = ref('')
const rightSearchText = ref('')
const collapsedGroups = ref<Set<number>>(new Set())

// 本地选中的索引列表
const localSelectedIndices = computed({
  get: () => {
    // 优先使用传入的 modelValue
    if (props.modelValue && props.modelValue.length > 0) {
      return props.modelValue
    }
    // 否则从 searchGroups 中提取已选中的索引
    const selected: string[] = []
    props.searchGroups.forEach(group => {
      if (group.selectedIndices && group.selectedIndices.length > 0) {
        selected.push(...group.selectedIndices)
      }
    })
    return selected
  },
  set: (val) => {
    emit('update:modelValue', val)
    // 同时更新 searchGroups 中的 selectedIndices
    updateSearchGroupsSelection(val)
  }
})

const localTransform = ref<IndexNameTransform>({ ...props.indexNameTransform })

// 更新 searchGroups 中每个组的 selectedIndices
function updateSearchGroupsSelection(selectedList: string[]) {
  const selectedSet = new Set(selectedList)
  // 创建新的 searchGroups 副本，避免直接修改 props
  const updatedGroups = props.searchGroups.map(group => ({
    ...group,
    selectedIndices: group.matchedIndices.filter(idx => selectedSet.has(idx))
  }))
  // 通知父组件更新
  emit('update:searchGroups', updatedGroups)
}

// 监听 modelValue 变化，确保 searchGroups 同步更新
watch(() => props.modelValue, (newVal) => {
  if (newVal && newVal.length >= 0) {
    updateSearchGroupsSelection(newVal)
  }
}, { immediate: true })

// 总索引数
const totalIndices = computed(() => {
  return props.searchGroups.reduce((sum, group) => sum + group.matchedIndices.length, 0)
})

// 左侧已选数量
const leftSelectedCount = computed(() => {
  return localSelectedIndices.value.length
})

// 过滤后的分组（左侧搜索）
const filteredGroups = computed(() => {
  if (!leftSearchText.value.trim()) {
    return props.searchGroups
  }
  const searchLower = leftSearchText.value.toLowerCase()
  return props.searchGroups.map(group => ({
    ...group,
    matchedIndices: group.matchedIndices.filter(idx => 
      idx.toLowerCase().includes(searchLower)
    )
  })).filter(group => group.matchedIndices.length > 0)
})

// 过滤后的已选索引（右侧搜索）
const filteredSelectedIndices = computed(() => {
  if (!rightSearchText.value.trim()) {
    return localSelectedIndices.value
  }
  const searchLower = rightSearchText.value.toLowerCase()
  return localSelectedIndices.value.filter(idx => 
    idx.toLowerCase().includes(searchLower)
  )
})

// 判断组是否全选
function isGroupAllSelected(group: ESSearchGroup): boolean {
  if (group.matchedIndices.length === 0) return false
  return group.matchedIndices.every(idx => localSelectedIndices.value.includes(idx))
}

// 判断组是否部分选中
function isGroupIndeterminate(group: ESSearchGroup): boolean {
  const selectedCount = group.matchedIndices.filter(idx => 
    localSelectedIndices.value.includes(idx)
  ).length
  return selectedCount > 0 && selectedCount < group.matchedIndices.length
}

// 切换组折叠状态
function toggleGroupCollapse(groupIndex: number) {
  if (collapsedGroups.value.has(groupIndex)) {
    collapsedGroups.value.delete(groupIndex)
  } else {
    collapsedGroups.value.add(groupIndex)
  }
  // 触发响应式更新
  collapsedGroups.value = new Set(collapsedGroups.value)
}

// 切换组选择
function toggleGroup(group: ESSearchGroup, checked: boolean) {
  const newSelected = new Set(localSelectedIndices.value)
  group.matchedIndices.forEach(idx => {
    if (checked) {
      newSelected.add(idx)
    } else {
      newSelected.delete(idx)
    }
  })
  localSelectedIndices.value = Array.from(newSelected)
}

// 切换单个索引
function toggleIndex(index: string, checked: boolean) {
  const newSelected = new Set(localSelectedIndices.value)
  if (checked) {
    newSelected.add(index)
  } else {
    newSelected.delete(index)
  }
  localSelectedIndices.value = Array.from(newSelected)
}

// 移除索引
function removeIndex(index: string) {
  localSelectedIndices.value = localSelectedIndices.value.filter(idx => idx !== index)
}

// 全选
function selectAll() {
  const allIndices = props.searchGroups.flatMap(group => group.matchedIndices)
  localSelectedIndices.value = allIndices
}

// 清空
function clearSelection() {
  localSelectedIndices.value = []
}

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
</script>

<style scoped>
.custom-transfer {
  display: flex;
  gap: 16px;
  height: 450px;
}

.transfer-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  border: 1px solid #e0e0e6;
  border-radius: 4px;
  overflow: hidden;
}

.panel-header {
  padding: 12px;
  border-bottom: 1px solid #e0e0e6;
  background-color: #fafafc;
}

.panel-content {
  flex: 1;
  padding: 8px;
}

.panel-footer {
  padding: 8px 12px;
  border-top: 1px solid #e0e0e6;
  background-color: #fafafc;
}

.transfer-actions {
  display: flex;
  align-items: center;
  justify-content: center;
}

.group-item {
  margin-bottom: 12px;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 8px;
}

.collapse-btn {
  padding: 0;
  min-width: 20px;
}

.group-children {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-left: 24px;
}

.index-checkbox {
  padding: 4px 0;
}

.selected-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.selected-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-radius: 4px;
  background-color: #f5f5f5;
  transition: background-color 0.3s;
}

.selected-item:hover {
  background-color: #e8e8e8;
}
</style>
