<template>
  <div class="sync-task-config">
    <n-space vertical :size="16">
      <!-- 操作栏 -->
      <n-space justify="space-between">
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <n-icon><AddIcon /></n-icon>
          </template>
          创建任务
        </n-button>
        <n-button @click="loadTasks">
          <template #icon>
            <n-icon><RefreshIcon /></n-icon>
          </template>
          刷新
        </n-button>
      </n-space>

      <!-- 任务列表 -->
      <n-data-table
        :columns="columns"
        :data="syncTaskStore.tasks"
        :loading="syncTaskStore.loading"
        :pagination="pagination"
        :bordered="false"
      />
    </n-space>

    <!-- 添加/编辑任务对话框 -->
    <n-modal
      v-model:show="showModal"
      :title="isEdit ? '编辑任务' : '创建任务'"
      preset="card"
      style="width: 800px"
      :segmented="{ content: 'soft', footer: 'soft' }"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="120"
      >
        <n-form-item label="任务名称" path="name">
          <n-input v-model:value="formData.name" placeholder="请输入任务名称" />
        </n-form-item>

        <n-form-item label="源数据源" path="sourceId">
          <n-select
            v-model:value="formData.sourceId"
            :options="dataSourceOptions"
            placeholder="请选择源数据源"
            @update:value="handleSourceChange"
          />
        </n-form-item>

        <n-form-item label="目标数据源" path="targetId">
          <n-select
            v-model:value="formData.targetId"
            :options="dataSourceOptions"
            placeholder="请选择目标数据源"
            @update:value="handleTargetChange"
          />
        </n-form-item>

        <!-- MySQL 数据库和表选择 -->
        <template v-if="sourceType === 'mysql'">
          <n-form-item label="选择数据库和表">
            <n-space vertical style="width: 100%">
              <n-button @click="loadDatabases" :loading="loadingDatabases">
                加载数据库列表
              </n-button>
              <n-collapse v-if="databases.length > 0">
                <n-collapse-item
                  v-for="db in databases"
                  :key="db"
                  :title="db"
                >
                  <n-space vertical>
                    <n-button
                      size="small"
                      @click="loadTables(db)"
                      :loading="loadingTables[db]"
                    >
                      加载表列表
                    </n-button>
                    <n-checkbox-group
                      v-if="tables[db]"
                      v-model:value="selectedTables[db]"
                    >
                      <n-space vertical>
                        <n-checkbox
                          v-for="table in tables[db]"
                          :key="table"
                          :value="table"
                          :label="table"
                        />
                      </n-space>
                    </n-checkbox-group>
                  </n-space>
                </n-collapse-item>
              </n-collapse>
            </n-space>
          </n-form-item>
        </template>

        <!-- ES 索引选择 -->
        <template v-if="sourceType === 'elasticsearch'">
          <n-form-item label="索引选择">
            <n-space vertical style="width: 100%">
              <n-input
                v-model:value="indexPattern"
                placeholder="输入索引名或通配符（如 logs-*）"
              >
                <template #suffix>
                  <n-button
                    text
                    @click="matchIndices"
                    :loading="loadingIndices"
                  >
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
          </n-form-item>
        </template>

        <n-divider />

        <!-- 同步配置 -->
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
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" @click="handleSubmit">
            {{ isEdit ? '保存' : '创建' }}
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 删除确认对话框 -->
    <n-modal
      v-model:show="showDeleteModal"
      preset="dialog"
      title="确认删除"
      content="确定要删除这个任务吗？此操作不可恢复。"
      positive-text="删除"
      negative-text="取消"
      @positive-click="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, h, computed, onMounted, reactive } from 'vue'
import {
  NSpace,
  NButton,
  NDataTable,
  NModal,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NRadioGroup,
  NRadio,
  NIcon,
  NTag,
  NCollapse,
  NCollapseItem,
  NCheckboxGroup,
  NCheckbox,
  NAlert,
  NDivider,
  type DataTableColumns,
  type FormInst,
  type FormRules
} from 'naive-ui'
import {
  Add as AddIcon,
  Refresh as RefreshIcon,
  Create as EditIcon,
  Trash as DeleteIcon
} from '@vicons/ionicons5'
import { useDataSourceStore } from '../stores/dataSource'
import { useSyncTaskStore } from '../stores/syncTask'
import { showSuccess, showError, handleApiError } from '../utils/message'
import type { SyncTask } from '../types'

const dataSourceStore = useDataSourceStore()
const syncTaskStore = useSyncTaskStore()

// 表格配置
const pagination = {
  pageSize: 10
}

// 表格列定义
const columns: DataTableColumns<SyncTask> = [
  {
    title: '任务名称',
    key: 'name',
    width: 150
  },
  {
    title: '源类型',
    key: 'sourceType',
    width: 120,
    render: (row) => {
      return h(
        NTag,
        { type: row.sourceType === 'mysql' ? 'info' : 'success' },
        { default: () => row.sourceType === 'mysql' ? 'MySQL' : 'Elasticsearch' }
      )
    }
  },
  {
    title: '目标类型',
    key: 'targetType',
    width: 120,
    render: (row) => {
      return h(
        NTag,
        { type: row.targetType === 'mysql' ? 'info' : 'success' },
        { default: () => row.targetType === 'mysql' ? 'MySQL' : 'Elasticsearch' }
      )
    }
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => {
      const statusMap: Record<string, { type: any; label: string }> = {
        idle: { type: 'default', label: '空闲' },
        running: { type: 'info', label: '运行中' },
        paused: { type: 'warning', label: '已暂停' },
        completed: { type: 'success', label: '已完成' },
        failed: { type: 'error', label: '失败' }
      }
      const status = statusMap[row.status] || statusMap.idle
      return h(NTag, { type: status.type }, { default: () => status.label })
    }
  },
  {
    title: '线程数',
    key: 'syncConfig.threadCount',
    width: 80,
    render: (row) => row.syncConfig.threadCount
  },
  {
    title: '批量大小',
    key: 'syncConfig.batchSize',
    width: 100,
    render: (row) => row.syncConfig.batchSize
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 180,
    render: (row) => new Date(row.createdAt).toLocaleString()
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row) => {
      return h(
        NSpace,
        {},
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                onClick: () => handleEdit(row)
              },
              { default: () => '编辑', icon: () => h(NIcon, null, { default: () => h(EditIcon) }) }
            ),
            h(
              NButton,
              {
                size: 'small',
                type: 'error',
                onClick: () => handleDelete(row)
              },
              { default: () => '删除', icon: () => h(NIcon, null, { default: () => h(DeleteIcon) }) }
            )
          ]
        }
      )
    }
  }
]

// 表单相关
const showModal = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInst | null>(null)
const formData = ref<Partial<SyncTask>>({
  name: '',
  sourceId: '',
  targetId: '',
  sourceType: 'mysql',
  targetType: 'mysql',
  mysqlConfig: {
    databases: []
  },
  esConfig: {
    indices: []
  },
  syncConfig: {
    threadCount: 4,
    batchSize: 1000,
    errorStrategy: 'skip'
  },
  status: 'idle'
})

const rules: FormRules = {
  name: [
    { required: true, message: '请输入任务名称', trigger: 'blur' }
  ],
  sourceId: [
    { required: true, message: '请选择源数据源', trigger: 'change' }
  ],
  targetId: [
    { required: true, message: '请选择目标数据源', trigger: 'change' }
  ]
}

// 数据源选项
const dataSourceOptions = computed(() => {
  return dataSourceStore.dataSources.map(ds => ({
    label: `${ds.name} (${ds.type})`,
    value: ds.id
  }))
})

// 源和目标类型
const sourceType = ref<'mysql' | 'elasticsearch'>('mysql')
const targetType = ref<'mysql' | 'elasticsearch'>('mysql')

// MySQL 相关
const databases = ref<string[]>([])
const tables = reactive<Record<string, string[]>>({})
const selectedTables = reactive<Record<string, string[]>>({})
const loadingDatabases = ref(false)
const loadingTables = reactive<Record<string, boolean>>({})

// ES 相关
const indexPattern = ref('')
const matchedIndices = ref<string[]>([])
const loadingIndices = ref(false)

// 删除相关
const showDeleteModal = ref(false)
const deleteTarget = ref<SyncTask | null>(null)

// 方法
function handleAdd() {
  isEdit.value = false
  formData.value = {
    name: '',
    sourceId: '',
    targetId: '',
    sourceType: 'mysql',
    targetType: 'mysql',
    mysqlConfig: {
      databases: []
    },
    esConfig: {
      indices: []
    },
    syncConfig: {
      threadCount: 4,
      batchSize: 1000,
      errorStrategy: 'skip'
    },
    status: 'idle'
  }
  databases.value = []
  Object.keys(tables).forEach(key => delete tables[key])
  Object.keys(selectedTables).forEach(key => delete selectedTables[key])
  indexPattern.value = ''
  matchedIndices.value = []
  showModal.value = true
}

function handleEdit(row: SyncTask) {
  isEdit.value = true
  formData.value = { ...row }
  
  // 恢复 MySQL 配置
  if (row.mysqlConfig) {
    row.mysqlConfig.databases.forEach(dbConfig => {
      selectedTables[dbConfig.database] = dbConfig.tables
    })
  }
  
  // 恢复 ES 配置
  if (row.esConfig && row.esConfig.indices.length > 0) {
    indexPattern.value = row.esConfig.indices[0].pattern
    matchedIndices.value = row.esConfig.indices[0].matchedIndices || []
  }
  
  showModal.value = true
}

function handleSourceChange(value: string) {
  const ds = dataSourceStore.dataSources.find(d => d.id === value)
  if (ds) {
    sourceType.value = ds.type
    formData.value.sourceType = ds.type
  }
}

function handleTargetChange(value: string) {
  const ds = dataSourceStore.dataSources.find(d => d.id === value)
  if (ds) {
    targetType.value = ds.type
    formData.value.targetType = ds.type
  }
}

async function loadDatabases() {
  if (!formData.value.sourceId) {
    showError('请先选择源数据源')
    return
  }
  
  loadingDatabases.value = true
  try {
    databases.value = await syncTaskStore.fetchDatabases(formData.value.sourceId)
  } catch (error) {
    handleApiError(error, '加载数据库列表失败')
  } finally {
    loadingDatabases.value = false
  }
}

async function loadTables(database: string) {
  if (!formData.value.sourceId) return
  
  loadingTables[database] = true
  try {
    const tableList = await syncTaskStore.fetchTables(formData.value.sourceId, database)
    tables[database] = tableList
  } catch (error) {
    handleApiError(error, '加载表列表失败')
  } finally {
    loadingTables[database] = false
  }
}

async function matchIndices() {
  if (!formData.value.sourceId) {
    showError('请先选择源数据源')
    return
  }
  
  if (!indexPattern.value) {
    showError('请输入索引名或通配符')
    return
  }
  
  loadingIndices.value = true
  try {
    const result = await syncTaskStore.matchIndices(formData.value.sourceId, indexPattern.value)
    matchedIndices.value = result.preview
    showSuccess(`匹配到 ${result.count} 个索引`)
  } catch (error) {
    handleApiError(error, '匹配索引失败')
  } finally {
    loadingIndices.value = false
  }
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
    
    // 构建 MySQL 配置
    if (sourceType.value === 'mysql') {
      formData.value.mysqlConfig = {
        databases: Object.keys(selectedTables)
          .filter(db => selectedTables[db].length > 0)
          .map(db => ({
            database: db,
            tables: selectedTables[db]
          }))
      }
    }
    
    // 构建 ES 配置
    if (sourceType.value === 'elasticsearch') {
      formData.value.esConfig = {
        indices: [{
          pattern: indexPattern.value,
          matchedIndices: matchedIndices.value
        }]
      }
    }
    
    if (isEdit.value && formData.value.id) {
      await syncTaskStore.updateTask(formData.value.id, formData.value as SyncTask)
      showSuccess('任务更新成功')
    } else {
      await syncTaskStore.createTask(formData.value as SyncTask)
      showSuccess('任务创建成功')
    }
    
    showModal.value = false
    await loadTasks()
  } catch (error) {
    handleApiError(error, isEdit.value ? '更新任务失败' : '创建任务失败')
  }
}

function handleDelete(row: SyncTask) {
  deleteTarget.value = row
  showDeleteModal.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  
  try {
    await syncTaskStore.deleteTask(deleteTarget.value.id)
    showSuccess('任务删除成功')
    await loadTasks()
  } catch (error) {
    handleApiError(error, '删除任务失败')
  } finally {
    deleteTarget.value = null
  }
}

async function loadTasks() {
  try {
    await syncTaskStore.fetchTasks()
  } catch (error) {
    handleApiError(error, '加载任务列表失败')
  }
}

onMounted(async () => {
  await dataSourceStore.fetchDataSources()
  await loadTasks()
})
</script>

<style scoped>
.sync-task-config {
  height: 100%;
}
</style>
