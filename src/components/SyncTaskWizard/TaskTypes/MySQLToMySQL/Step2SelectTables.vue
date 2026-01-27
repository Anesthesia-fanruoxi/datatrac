<template>
  <n-spin :show="loading">
    <n-space vertical :size="16">
      <n-space justify="space-between">
        <n-button @click="loadAllTables">加载所有表</n-button>
        <n-text>已选择 {{ modelValue.length }} 个表</n-text>
      </n-space>

      <n-collapse>
        <n-collapse-item v-for="db in databases" :key="db.name" :title="`${db.name} (${db.tables.length})`">
          <n-checkbox-group v-model:value="localValue">
            <n-space vertical>
              <n-checkbox v-for="table in db.tables" :key="`${db.name}.${table}`" :value="`${db.name}.${table}`">
                {{ table }}
              </n-checkbox>
            </n-space>
          </n-checkbox-group>
        </n-collapse-item>
      </n-collapse>

      <n-divider />
      
      <DbNameTransform v-model="localDbNameTransform" />
    </n-space>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NSpin, NSpace, NButton, NText, NCollapse, NCollapseItem, NCheckboxGroup, NCheckbox, NDivider } from 'naive-ui'
import { useSyncTaskStore } from '../../../../stores/syncTask'
import DbNameTransform from '../../Step2/DbNameTransform.vue'
import type { DbNameTransform as DbNameTransformType } from '../../../../types'

const props = defineProps<{
  sourceId: string
  modelValue: string[]
  dbNameTransform: DbNameTransformType
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  'update:dbNameTransform': [value: DbNameTransformType]
}>()

const localValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const localDbNameTransform = computed({
  get: () => props.dbNameTransform,
  set: (val) => emit('update:dbNameTransform', val)
})

const syncTaskStore = useSyncTaskStore()
const loading = ref(false)
const databases = ref<Array<{ name: string; tables: string[] }>>([])

async function loadAllTables() {
  loading.value = true
  try {
    const dbList = await syncTaskStore.fetchDatabases(props.sourceId)
    const result = []
    for (const db of dbList) {
      const tables = await syncTaskStore.fetchTables(props.sourceId, db)
      result.push({ name: db, tables })
    }
    databases.value = result
  } finally {
    loading.value = false
  }
}

watch(() => props.sourceId, () => {
  if (props.sourceId) {
    loadAllTables()
  }
}, { immediate: true })

defineExpose({ loadAllTables })
</script>
