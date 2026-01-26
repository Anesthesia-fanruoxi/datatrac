<template>
  <div style="display: flex; gap: 16px;">
    <!-- 左侧：源表（树形结构） -->
    <div style="flex: 1; border: 1px solid #e0e0e0; border-radius: 4px; padding: 12px;">
      <div style="margin-bottom: 8px; display: flex; justify-content: space-between; align-items: center;">
        <span style="font-weight: 500;">源表</span>
        <n-space>
          <n-button size="small" @click="handleSelectAll">全选</n-button>
          <n-button size="small" @click="handleUnselectAll">取消全选</n-button>
        </n-space>
      </div>
      <n-input 
        v-model:value="sourceFilter" 
        placeholder="搜索数据库或表" 
        clearable
        style="margin-bottom: 12px;"
      />
      <n-tree
        :data="filteredSourceTreeData"
        checkable
        cascade
        :checked-keys="checkedKeys"
        @update:checked-keys="handleCheckedKeysChange"
        :pattern="sourceFilter"
        :show-irrelevant-nodes="false"
        block-line
        expand-on-click
        key-field="key"
        label-field="label"
        children-field="children"
        :render-prefix="renderPrefix"
        style="max-height: 400px; overflow-y: auto;"
      />
    </div>

    <!-- 右侧：已选表（树形结构） -->
    <div style="flex: 1; border: 1px solid #e0e0e0; border-radius: 4px; padding: 12px;">
      <div style="margin-bottom: 8px; font-weight: 500;">
        已选表 ({{ selectedTablesList.length }})
      </div>
      <n-input 
        v-model:value="targetFilter" 
        placeholder="搜索已选表" 
        clearable
        style="margin-bottom: 12px;"
      />
      <n-tree
        :data="filteredSelectedTreeData"
        :show-line="false"
        block-line
        expand-on-click
        key-field="key"
        label-field="label"
        children-field="children"
        :render-prefix="renderPrefix"
        :render-suffix="renderSuffix"
        style="max-height: 400px; overflow-y: auto;"
      />
      <n-empty 
        v-if="selectedTablesList.length === 0" 
        description="暂无选择"
        size="small"
        style="margin-top: 20px;"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { NTree, NInput, NButton, NSpace, NEmpty, NIcon, type TreeOption } from 'naive-ui'
import { Close as CloseIcon, Server as DatabaseIcon, List as TableIcon } from '@vicons/ionicons5'

const props = defineProps<{
  treeData: TreeOption[]
  checkedKeys: string[]
  dbNameTransform?: {
    enabled: boolean
    mode: 'prefix' | 'suffix'
    sourcePattern: string
    targetPattern: string
  }
}>()

const emit = defineEmits<{
  'update:checkedKeys': [keys: string[]]
}>()

const sourceFilter = ref('')
const targetFilter = ref('')

// 过滤后的源树数据
const filteredSourceTreeData = computed(() => {
  console.log('源树数据:', props.treeData)
  if (!sourceFilter.value) return props.treeData
  
  const filter = sourceFilter.value.toLowerCase()
  return props.treeData.map(db => {
    const dbMatches = db.label?.toString().toLowerCase().includes(filter)
    const filteredChildren = db.children?.filter(table => 
      table.label?.toString().toLowerCase().includes(filter)
    ) || []
    
    if (dbMatches || filteredChildren.length > 0) {
      return {
        ...db,
        children: dbMatches ? db.children : filteredChildren
      }
    }
    return null
  }).filter(Boolean) as TreeOption[]
})

// 已选表列表（只包含表，不包含数据库）
const selectedTablesList = computed(() => {
  return props.checkedKeys.filter(key => key.includes('.'))
})

// 转换数据库名称
function transformDbName(dbName: string): string {
  if (!props.dbNameTransform || !props.dbNameTransform.enabled) {
    return dbName
  }
  
  const { mode, sourcePattern, targetPattern } = props.dbNameTransform
  
  if (!sourcePattern || !targetPattern) {
    return dbName
  }
  
  if (mode === 'prefix') {
    if (dbName.startsWith(sourcePattern)) {
      return targetPattern + dbName.slice(sourcePattern.length)
    }
  } else {
    if (dbName.endsWith(sourcePattern)) {
      return dbName.slice(0, -sourcePattern.length) + targetPattern
    }
  }
  
  return dbName
}

// 已选表的树形数据（按数据库分组）
const selectedTreeData = computed(() => {
  const dbMap = new Map<string, string[]>()
  
  selectedTablesList.value.forEach(fullKey => {
    const [db, table] = fullKey.split('.')
    if (!dbMap.has(db)) {
      dbMap.set(db, [])
    }
    dbMap.get(db)!.push(table)
  })
  
  return Array.from(dbMap.entries()).map(([db, tables]) => {
    const transformedDb = transformDbName(db)
    const label = transformedDb !== db 
      ? `${db} → ${transformedDb}`
      : db
    
    return {
      key: db,
      label,
      isTable: false,
      children: tables.map(table => ({
        key: `${db}.${table}`,
        label: table,
        isTable: true
      }))
    }
  })
})

// 过滤后的已选表树
const filteredSelectedTreeData = computed(() => {
  if (!targetFilter.value) return selectedTreeData.value
  
  const filter = targetFilter.value.toLowerCase()
  return selectedTreeData.value.map(db => {
    const dbMatches = db.label?.toString().toLowerCase().includes(filter)
    const filteredChildren = db.children?.filter(table => 
      table.label?.toString().toLowerCase().includes(filter)
    ) || []
    
    if (dbMatches || filteredChildren.length > 0) {
      return {
        ...db,
        children: dbMatches ? db.children : filteredChildren
      }
    }
    return null
  }).filter(Boolean) as TreeOption[]
})

function handleCheckedKeysChange(keys: string[]) {
  emit('update:checkedKeys', keys)
}

function handleSelectAll() {
  const allKeys: string[] = []
  props.treeData.forEach(db => {
    allKeys.push(db.key as string)
    if (db.children) {
      db.children.forEach(table => {
        allKeys.push(table.key as string)
      })
    }
  })
  emit('update:checkedKeys', allKeys)
}

function handleUnselectAll() {
  emit('update:checkedKeys', [])
}

function handleRemoveTable(tableKey: string) {
  const newKeys = props.checkedKeys.filter(key => key !== tableKey)
  emit('update:checkedKeys', newKeys)
}

// 渲染前缀图标
function renderPrefix({ option }: { option: any }) {
  return h(
    NIcon,
    { 
      color: option.isTable ? '#666' : '#0ea5e9',
      style: { marginRight: '4px' }
    },
    {
      default: () => h(option.isTable ? TableIcon : DatabaseIcon)
    }
  )
}

// 渲染后缀（删除按钮）
function renderSuffix({ option }: { option: any }) {
  if (!option.isTable) return null
  
  return h(
    NButton,
    {
      text: true,
      size: 'small',
      onClick: (e: Event) => {
        e.stopPropagation()
        handleRemoveTable(option.key as string)
      }
    },
    {
      icon: () => h(NIcon, null, { default: () => h(CloseIcon) })
    }
  )
}
</script>
