<template>
  <n-spin :show="loading">
    <n-space vertical :size="16">
      <!-- 操作栏 -->
      <n-space justify="space-between">
        <n-space>
          <n-button @click="loadAllTables">刷新数据</n-button>
          <n-button @click="selectAll">全选</n-button>
          <n-button @click="clearAll">清空</n-button>
          <n-input v-model:value="searchPattern" placeholder="搜索数据库或表..." clearable style="width: 200px" />
        </n-space>
        <n-text>已选择 {{ selectedTables.length }} 个表</n-text>
      </n-space>

      <!-- 穿梭框布局 -->
      <div style="display: flex; gap: 16px; height: 500px;">
        <!-- 左侧：可选项 -->
        <n-card title="可选表" style="flex: 1; overflow: hidden;">
          <n-scrollbar style="max-height: 450px;">
            <n-tree
              v-if="filteredTreeData.length > 0"
              :data="filteredTreeData"
              :checked-keys="checkedKeys"
              checkable
              cascade
              expand-on-click
              selectable
              :default-expanded-keys="defaultExpandedKeys"
              :on-update:checked-keys="handleCheckedKeysChange"
              key-field="key"
              label-field="label"
              children-field="children"
            />
            <n-empty v-else description="暂无数据" />
          </n-scrollbar>
        </n-card>

        <!-- 右侧：已选项(树形结构) -->
        <n-card title="已选表" style="flex: 1; overflow: hidden;">
          <n-scrollbar style="max-height: 450px;">
            <n-tree
              v-if="selectedTreeData.length > 0"
              :data="selectedTreeData"
              :default-expanded-keys="selectedExpandedKeys"
              :selectable="false"
            />
            <n-empty v-else description="请从左侧选择要同步的表" />
          </n-scrollbar>
        </n-card>
      </div>

      <!-- 批量设置库名 -->
      <n-card title="批量设置目标库名" size="small">
        <n-space vertical :size="12">
          <n-radio-group v-model:value="batchMode">
            <n-space>
              <n-radio value="keep">保持原库名</n-radio>
              <n-radio value="prefix">修改前缀</n-radio>
              <n-radio value="suffix">修改后缀</n-radio>
            </n-space>
          </n-radio-group>

          <n-space v-if="batchMode !== 'keep'" align="center">
            <n-space v-if="batchMode === 'prefix'" align="center">
              <n-text>源前缀:</n-text>
              <n-input
                v-model:value="batchSourcePrefix"
                placeholder="要替换的前缀"
                style="width: 150px"
              />
              <n-text>→</n-text>
              <n-text>目标前缀:</n-text>
              <n-input
                v-model:value="batchTargetPrefix"
                placeholder="新前缀"
                style="width: 150px"
              />
            </n-space>
            <n-space v-if="batchMode === 'suffix'" align="center">
              <n-text>源后缀:</n-text>
              <n-input
                v-model:value="batchSourceSuffix"
                placeholder="要替换的后缀"
                style="width: 150px"
              />
              <n-text>→</n-text>
              <n-text>目标后缀:</n-text>
              <n-input
                v-model:value="batchTargetSuffix"
                placeholder="新后缀"
                style="width: 150px"
              />
            </n-space>
            <n-button type="primary" @click="applyBatchOperation" :disabled="selectedDatabases.length === 0">
              应用
            </n-button>
          </n-space>

          <!-- 显示当前映射 -->
          <n-space v-if="selectedDatabases.length > 0" vertical :size="4">
            <n-text depth="3">当前库名映射:</n-text>
            <n-space>
              <n-tag v-for="db in selectedDatabases" :key="db.name" size="small">
                {{ db.name }} → {{ db.targetName }}
              </n-tag>
            </n-space>
          </n-space>
        </n-space>
      </n-card>
    </n-space>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, computed, watch, h } from 'vue'
import { 
  NSpin, NSpace, NButton, NText, NCard, NScrollbar, NTree, NTag, 
  NInput, NRadioGroup, NRadio, NEmpty, NIcon
} from 'naive-ui'
import { Server as DatabaseIcon, Document as TableIcon } from '@vicons/ionicons5'
import { useSyncTaskStore } from '../../../../stores/syncTask'
import type { DbNameTransform as DbNameTransformType } from '../../../../types'
import type { TreeOption } from 'naive-ui'

interface DatabaseInfo {
  name: string
  targetName: string
  tables: string[]
}

const props = defineProps<{
  sourceId: string
  modelValue: string[]
  dbNameTransform: DbNameTransformType
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  'update:dbNameTransform': [value: DbNameTransformType]
}>()

const syncTaskStore = useSyncTaskStore()
const loading = ref(false)
const databases = ref<Array<{ name: string; tables: string[] }>>([])
const searchPattern = ref('')
const checkedKeys = ref<string[]>([])
const targetDbNames = ref<Record<string, string>>({})
const batchMode = ref<'keep' | 'prefix' | 'suffix'>('keep')
const batchSourcePrefix = ref('')
const batchTargetPrefix = ref('')
const batchSourceSuffix = ref('')
const batchTargetSuffix = ref('')

// 构建树形数据
const treeData = computed<TreeOption[]>(() => {
  return databases.value.map(db => ({
    key: db.name,
    label: `${db.name} (${db.tables.length})`,
    isLeaf: false,
    prefix: () => h(NIcon, { color: '#18a058' }, { default: () => h(DatabaseIcon) }),
    children: db.tables.map(table => ({
      key: `${db.name}.${table}`,
      label: table,
      isLeaf: true,
      prefix: () => h(NIcon, { color: '#2080f0' }, { default: () => h(TableIcon) })
    }))
  }))
})

// 过滤后的树形数据
const filteredTreeData = computed<TreeOption[]>(() => {
  if (!searchPattern.value) return treeData.value
  
  const pattern = searchPattern.value.toLowerCase()
  return treeData.value
    .map(db => {
      const dbMatches = db.label?.toString().toLowerCase().includes(pattern) || false
      const filteredChildren = db.children?.filter(table => 
        table.label?.toString().toLowerCase().includes(pattern)
      ) || []
      
      if (dbMatches || filteredChildren.length > 0) {
        return {
          ...db,
          children: dbMatches ? db.children : filteredChildren
        }
      }
      return null
    })
    .filter(Boolean) as TreeOption[]
})

// 默认展开的节点
const defaultExpandedKeys = computed(() => {
  // 返回空数组,默认不展开
  return []
})

// 已选择的表
const selectedTables = computed(() => {
  return checkedKeys.value.filter(key => key.includes('.'))
})

// 已选择的数据库及其表
const selectedDatabases = computed<DatabaseInfo[]>(() => {
  const dbMap = new Map<string, string[]>()
  
  selectedTables.value.forEach(fullName => {
    const [dbName, tableName] = fullName.split('.')
    if (!dbMap.has(dbName)) {
      dbMap.set(dbName, [])
    }
    dbMap.get(dbName)!.push(tableName)
  })
  
  return Array.from(dbMap.entries()).map(([name, tables]) => ({
    name,
    targetName: targetDbNames.value[name] || name,
    tables
  }))
})

// 右侧树形数据(已选择的)
const selectedTreeData = computed<TreeOption[]>(() => {
  return selectedDatabases.value.map(db => ({
    key: `selected-${db.name}`,
    label: `${db.name} → ${db.targetName}`,
    isLeaf: false,
    prefix: () => h(NIcon, { color: '#18a058' }, { default: () => h(DatabaseIcon) }),
    children: db.tables.map(table => ({
      key: `selected-${db.name}.${table}`,
      label: table,
      isLeaf: true,
      prefix: () => h(NIcon, { color: '#2080f0' }, { default: () => h(TableIcon) })
    }))
  }))
})

// 右侧默认不展开
const selectedExpandedKeys = computed(() => {
  return []
})

// 处理选中变化
function handleCheckedKeysChange(keys: string[]) {
  console.log('选中的 keys:', keys)
  checkedKeys.value = keys
  
  // 使用 setTimeout 延迟触发,避免响应式更新冲突
  setTimeout(() => {
    const tables = keys.filter(key => key.includes('.'))
    console.log('选中的表:', tables)
    emit('update:modelValue', tables)
  }, 0)
}

// 全选
function selectAll() {
  const allKeys: string[] = []
  databases.value.forEach(db => {
    allKeys.push(db.name)
    db.tables.forEach(table => {
      allKeys.push(`${db.name}.${table}`)
    })
  })
  checkedKeys.value = allKeys
  
  // 使用 nextTick 确保 DOM 更新完成后再触发 emit
  setTimeout(() => {
    const tables = allKeys.filter(key => key.includes('.'))
    emit('update:modelValue', tables)
  }, 0)
}

// 清空所有选择
function clearAll() {
  checkedKeys.value = []
  targetDbNames.value = {}
  batchMode.value = 'keep'
  batchSourcePrefix.value = ''
  batchTargetPrefix.value = ''
  batchSourceSuffix.value = ''
  batchTargetSuffix.value = ''
  emit('update:modelValue', [])
  updateDbNameTransform()
}

// 批量操作
function applyBatchOperation() {
  selectedDatabases.value.forEach(db => {
    let newName = db.name
    
    switch (batchMode.value) {
      case 'keep':
        newName = db.name
        break
      case 'prefix':
        // 修改前缀：替换源前缀为目标前缀
        if (batchSourcePrefix.value && db.name.startsWith(batchSourcePrefix.value)) {
          newName = batchTargetPrefix.value + db.name.substring(batchSourcePrefix.value.length)
        } else {
          newName = batchTargetPrefix.value + db.name
        }
        break
      case 'suffix':
        // 修改后缀：替换源后缀为目标后缀
        if (batchSourceSuffix.value && db.name.endsWith(batchSourceSuffix.value)) {
          newName = db.name.substring(0, db.name.length - batchSourceSuffix.value.length) + batchTargetSuffix.value
        } else {
          newName = db.name + batchTargetSuffix.value
        }
        break
    }
    
    targetDbNames.value[db.name] = newName
  })
  
  updateDbNameTransform()
}

// 更新数据库名称转换配置
function updateDbNameTransform() {
  const dbs = selectedDatabases.value
  if (dbs.length === 0) {
    emit('update:dbNameTransform', {
      enabled: false,
      mode: 'prefix',
      sourcePattern: '',
      targetPattern: ''
    })
    return
  }
  
  // 检查是否使用了前缀或后缀
  if (batchMode.value === 'prefix' && (batchSourcePrefix.value || batchTargetPrefix.value)) {
    emit('update:dbNameTransform', {
      enabled: true,
      mode: 'prefix',
      sourcePattern: batchSourcePrefix.value,
      targetPattern: batchTargetPrefix.value
    })
  } else if (batchMode.value === 'suffix' && (batchSourceSuffix.value || batchTargetSuffix.value)) {
    emit('update:dbNameTransform', {
      enabled: true,
      mode: 'suffix',
      sourcePattern: batchSourceSuffix.value,
      targetPattern: batchTargetSuffix.value
    })
  } else {
    emit('update:dbNameTransform', {
      enabled: false,
      mode: 'prefix',
      sourcePattern: '',
      targetPattern: ''
    })
  }
}

// 加载所有表
async function loadAllTables() {
  loading.value = true
  try {
    const dbList = await syncTaskStore.fetchDatabases(props.sourceId)
    
    // 过滤系统库
    const systemDatabases = ['information_schema', 'mysql', 'performance_schema', 'sys']
    const userDbList = dbList.filter(db => !systemDatabases.includes(db.toLowerCase()))
    
    const result = []
    for (const db of userDbList) {
      const tables = await syncTaskStore.fetchTables(props.sourceId, db)
      result.push({ name: db, tables })
    }
    databases.value = result
  } catch (error) {
    console.error('加载表失败:', error)
  } finally {
    loading.value = false
  }
}

// 监听 sourceId 变化
watch(() => props.sourceId, () => {
  if (props.sourceId) {
    loadAllTables()
  }
}, { immediate: true })

// 恢复已选择的表
watch(() => props.modelValue, (newValue) => {
  if (newValue && newValue.length > 0) {
    checkedKeys.value = [...newValue]
    
    // 恢复目标库名
    const dbMap = new Map<string, string>()
    newValue.forEach(fullName => {
      const [dbName] = fullName.split('.')
      if (!dbMap.has(dbName)) {
        dbMap.set(dbName, dbName)
      }
    })
    
    // 如果没有保存的目标库名，使用源库名
    dbMap.forEach((_, dbName) => {
      if (!targetDbNames.value[dbName]) {
        targetDbNames.value[dbName] = dbName
      }
    })
  }
}, { immediate: true })

// 监听批量模式变化,自动应用
watch([batchMode, batchSourcePrefix, batchTargetPrefix, batchSourceSuffix, batchTargetSuffix], () => {
  if (batchMode.value === 'keep') {
    // 保持原库名
    selectedDatabases.value.forEach(db => {
      targetDbNames.value[db.name] = db.name
    })
    updateDbNameTransform()
  }
})

defineExpose({ loadAllTables })
</script>
