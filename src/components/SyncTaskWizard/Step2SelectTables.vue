<template>
  <div>
    <!-- MySQL -->
    <template v-if="sourceType === 'mysql'">
      <n-space vertical style="width: 100%">
        <n-alert 
          v-if="databaseTreeData.length > 0" 
          type="info" 
          :show-icon="false"
          style="margin-bottom: 16px"
        >
          已加载 {{ databaseTreeData.length }} 个数据库，共 {{ totalTableCount }} 个表
        </n-alert>

        <n-alert 
          v-if="loading" 
          type="info" 
          :show-icon="false"
        >
          正在加载数据库和表...
        </n-alert>

        <!-- 表选择器 -->
        <TableSelector
          v-if="databaseTreeData.length > 0"
          :tree-data="databaseTreeData"
          :checked-keys="checkedKeys"
          :db-name-transform="dbNameTransform"
          @update:checked-keys="handleCheckedKeysChange"
        />

        <!-- 数据库名称转换配置 -->
        <DbNameTransform
          v-if="databaseTreeData.length > 0"
          v-model="dbNameTransform"
        />
      </n-space>
    </template>

    <!-- Elasticsearch -->
    <template v-if="sourceType === 'elasticsearch'">
      <n-space vertical style="width: 100%">
        <n-input
          v-model:value="indexPattern"
          placeholder="输入索引名或通配符（如 logs-*）"
        >
          <template #suffix>
            <n-button text @click="handleMatchIndices" :loading="loading">
              匹配
            </n-button>
          </template>
        </n-input>
        
        <n-alert
          v-if="matchedIndices.length > 0"
          type="info"
          :title="`匹配到 ${matchedIndices.length} 个索引`"
        >
          <n-space>
            <n-tag
              v-for="index in matchedIndices.slice(0, 10)"
              :key="index"
              size="small"
            >
              {{ index }}
            </n-tag>
            <span v-if="matchedIndices.length > 10">
              ...还有 {{ matchedIndices.length - 10 }} 个
            </span>
          </n-space>
        </n-alert>
      </n-space>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NSpace, NAlert, NInput, NTag, NButton, type TreeOption } from 'naive-ui'
import { useSyncTaskStore } from '../../stores/syncTask'
import { handleApiError } from '../../utils/message'
import TableSelector from './Step2/TableSelector.vue'
import DbNameTransform from './Step2/DbNameTransform.vue'

const props = defineProps<{
  sourceId: string
  sourceType: 'mysql' | 'elasticsearch'
  modelValue: string[]
  indexPattern?: string
  matchedIndices?: string[]
  dbNameTransform?: {
    enabled: boolean
    mode: 'prefix' | 'suffix'
    sourcePattern: string
    targetPattern: string
  }
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  'update:indexPattern': [value: string]
  'update:matchedIndices': [value: string[]]
  'update:dbNameTransform': [value: {
    enabled: boolean
    mode: 'prefix' | 'suffix'
    sourcePattern: string
    targetPattern: string
  }]
}>()

const syncTaskStore = useSyncTaskStore()
const loading = ref(false)
const databaseTreeData = ref<TreeOption[]>([])
const checkedKeys = ref<string[]>([])
const dbNameTransform = ref({
  enabled: props.dbNameTransform?.enabled || false,
  mode: (props.dbNameTransform?.mode || 'prefix') as 'prefix' | 'suffix',
  sourcePattern: props.dbNameTransform?.sourcePattern || '',
  targetPattern: props.dbNameTransform?.targetPattern || ''
})

// MySQL 系统库列表
const SYSTEM_DATABASES = ['information_schema', 'mysql', 'performance_schema', 'sys']

const indexPattern = computed({
  get: () => props.indexPattern || '',
  set: (val) => emit('update:indexPattern', val)
})

const matchedIndices = computed({
  get: () => props.matchedIndices || [],
  set: (val) => emit('update:matchedIndices', val)
})

// 计算总表数
const totalTableCount = computed(() => {
  return databaseTreeData.value.reduce((sum, db) => {
    return sum + (db.children?.length || 0)
  }, 0)
})

// 已选表列表（只包含表，不包含数据库）
const selectedTablesList = computed(() => {
  return checkedKeys.value.filter(key => key.includes('.'))
})

// 同步选中的表到父组件
watch(selectedTablesList, (newTables) => {
  emit('update:modelValue', newTables)
}, { deep: true })

// 同步数据库名称转换配置到父组件
watch(dbNameTransform, (newValue) => {
  emit('update:dbNameTransform', newValue)
}, { deep: true })

// 初始化已选表（只在初始化时执行一次）
watch(() => props.modelValue, (newValue) => {
  if (newValue && newValue.length > 0 && checkedKeys.value.length === 0) {
    checkedKeys.value = [...newValue]
  }
}, { immediate: true })

// 监听 props.dbNameTransform 的变化，更新内部状态
watch(() => props.dbNameTransform, (newValue) => {
  if (newValue && JSON.stringify(newValue) !== JSON.stringify(dbNameTransform.value)) {
    dbNameTransform.value = {
      enabled: newValue.enabled,
      mode: newValue.mode,
      sourcePattern: newValue.sourcePattern,
      targetPattern: newValue.targetPattern
    }
  }
}, { deep: true, immediate: true })

async function loadAllDatabaseTables() {
  if (!props.sourceId) return
  
  loading.value = true
  databaseTreeData.value = []
  
  try {
    // 加载所有数据库
    const databases = await syncTaskStore.fetchDatabases(props.sourceId)
    
    // 过滤掉系统库
    const userDatabases = databases.filter(db => !SYSTEM_DATABASES.includes(db))
    
    // 并发加载所有数据库的表
    const promises = userDatabases.map(async (database) => {
      try {
        const tables = await syncTaskStore.fetchTables(props.sourceId, database)
        return {
          key: database,
          label: database,
          isTable: false,
          children: tables.map(table => ({
            key: `${database}.${table}`,
            label: table,
            isTable: true
          }))
        }
      } catch (error) {
        console.error(`加载数据库 ${database} 的表失败:`, error)
        return {
          key: database,
          label: database,
          isTable: false,
          children: []
        }
      }
    })
    
    const results = await Promise.all(promises)
    databaseTreeData.value = results
  } catch (error) {
    handleApiError(error, '加载数据库列表失败')
  } finally {
    loading.value = false
  }
}

function handleCheckedKeysChange(keys: string[]) {
  checkedKeys.value = keys
}

async function handleMatchIndices() {
  if (!props.sourceId) return
  
  if (!indexPattern.value) {
    return
  }
  
  loading.value = true
  try {
    const result = await syncTaskStore.matchIndices(props.sourceId, indexPattern.value)
    matchedIndices.value = result.preview
  } catch (error) {
    handleApiError(error, '匹配索引失败')
  } finally {
    loading.value = false
  }
}

defineExpose({
  loadAllDatabaseTables
})
</script>
