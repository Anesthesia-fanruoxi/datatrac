<template>
  <el-dialog v-model="visible" :title="dialogTitle" width="1100px" top="4vh" destroy-on-close @closed="handleClosed">
    <div class="wizard-layout">
      <el-steps :active="step" finish-status="success" align-center>
        <el-step title="选择数据源" />
        <el-step title="选择表" />
        <el-step title="同步配置" />
        <el-step title="确认配置" />
      </el-steps>

      <section v-if="step === 0" class="wizard-step-panel">
        <div class="wizard-grid two-cols">
          <div class="config-box">
            <h4>源数据源</h4>
            <el-select v-model="form.source_id" style="width: 100%" placeholder="请选择源数据源" @change="handleSourceChange">
              <el-option v-for="item in sourceOptions" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </div>
          <div class="config-box">
            <h4>目标数据源（可多选）</h4>
            <div class="target-card-grid">
              <div
                v-for="item in targetOptions"
                :key="item.id"
                class="target-card-item"
                :class="{ selected: form.target_ids.includes(item.id) }"
                @click="toggleTarget(item.id)"
              >
                <i v-if="form.target_ids.includes(item.id)" class="bi bi-check-circle-fill selected-icon"></i>
                <div class="card-body-inner">
                  <i class="bi bi-database"></i>
                  <div>
                    <div class="card-name">{{ item.name }}</div>
                    <small class="muted">{{ item.host }}</small>
                  </div>
                </div>
              </div>
            </div>
            <small class="muted d-block mt-2">已选 {{ form.target_ids.length }} 个</small>
          </div>
        </div>
      </section>

      <section v-else-if="step === 1" class="wizard-step-panel">
        <div v-if="databaseLoading" class="empty-state">正在加载数据库和表...</div>
        <div v-else-if="databaseOptions.length === 0" class="empty-state">请选择源数据源后继续</div>
        <div v-else class="wizard-grid two-cols stretch">
          <div class="config-box table-pick-box">
            <h4>源数据库与表</h4>
            <el-button size="small" class="mb-2" @click="selectAllLeft">全选</el-button>
            <div v-for="db in databaseOptions" :key="db.database" class="db-picker-block">
              <div class="db-picker-title">
                <button class="tree-toggle-btn" @click="toggleSourceDb(db.database)">
                  <i class="bi" :class="expandedSourceDbs[db.database] ? 'bi-chevron-down' : 'bi-chevron-right'"></i>
                </button>
                <el-checkbox
                  :model-value="isDatabaseFullySelected(db.database)"
                  :indeterminate="isDatabasePartiallySelected(db.database)"
                  @change="toggleDatabase(db.database, $event)"
                >
                  <strong>{{ db.database }}</strong> ({{ db.tables.length }})
                </el-checkbox>
              </div>
              <div v-show="expandedSourceDbs[db.database]" class="table-check-grid">
                <el-checkbox
                  v-for="table in db.tables"
                  :key="`${db.database}-${table}`"
                  :model-value="isTableSelected(db.database, table)"
                  @change="toggleTable(db.database, table, $event)"
                >
                  {{ table }}
                </el-checkbox>
              </div>
            </div>
          </div>
          <div class="config-box table-pick-box">
            <h4>已选映射</h4>
            <div class="batch-toolbar">
              <el-button size="small" @click="showBatchEdit = !showBatchEdit">批量改名</el-button>
              <el-button size="small" @click="clearAllMappings">清空</el-button>
            </div>
            <template v-if="showBatchEdit">
              <div class="batch-edit-panel">
                <el-input v-model="batchPrefix" placeholder="前缀" style="width: 100px" />
                <el-button size="small" @click="batchAddPrefix">加前缀</el-button>
                <el-input v-model="batchSuffix" placeholder="后缀" style="width: 100px" />
                <el-button size="small" @click="batchAddSuffix">加后缀</el-button>
                <el-input v-model="replaceOld" placeholder="旧前缀/后缀" style="width: 120px" />
                <el-input v-model="replaceNew" placeholder="新前缀/后缀" style="width: 120px" />
                <el-button size="small" @click="batchReplacePrefix">替换前缀</el-button>
                <el-button size="small" @click="batchReplaceSuffix">替换后缀</el-button>
              </div>
            </template>
            <div v-if="form.selected_databases.length === 0" class="empty-inline">还未选择任何表</div>
            <div v-for="db in form.selected_databases" :key="`${db.source_database}-${db.database}`" class="db-picker-block">
              <div class="mapping-db-header">
                <button class="tree-toggle-btn" @click="toggleSelectedDb(db.source_database)">
                  <i class="bi" :class="expandedSelectedDbs[db.source_database] ? 'bi-chevron-down' : 'bi-chevron-right'"></i>
                </button>
                <el-input v-model="db.database" placeholder="目标数据库名" @input="db.is_database_modified = db.database !== db.source_database" />
                <small class="muted">原：{{ db.source_database }}</small>
              </div>
              <div v-show="expandedSelectedDbs[db.source_database]" v-for="table in db.tables" :key="`${db.source_database}-${table.source_table}`" class="mapping-row">
                <span class="source-name">{{ table.source_table }}</span>
                <el-input v-model="table.target_table" placeholder="目标表名" @input="table.is_modified = table.target_table !== table.source_table" />
                <el-button type="primary" link @click="openFieldDialog(db.source_database, table.source_table)">
                  字段{{ table.selected_fields && table.selected_fields.length > 0 ? ` (${table.selected_fields.length})` : '' }}
                </el-button>
                <el-button type="danger" link @click="removeTable(db.source_database, table.source_table)">移除</el-button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="step === 2" class="wizard-step-panel">
        <div class="wizard-grid two-cols">
          <div class="config-box">
            <h4>同步模式</h4>
            <el-radio-group v-model="form.sync_config.sync_mode">
              <el-radio value="full">全量同步</el-radio>
              <el-radio value="incremental">增量同步</el-radio>
              <el-radio value="structure">结构同步</el-radio>
            </el-radio-group>
          </div>
          <div class="config-box">
            <h4>错误处理</h4>
            <el-radio-group v-model="form.sync_config.error_strategy" :disabled="form.sync_config.sync_mode === 'structure'">
              <el-radio value="skip">跳过错误</el-radio>
              <el-radio value="pause">遇错暂停</el-radio>
            </el-radio-group>
          </div>
          <div class="config-box">
            <h4>目标表策略</h4>
            <el-radio-group v-model="form.sync_config.table_exists_strategy" :disabled="form.sync_config.sync_mode === 'structure'">
              <el-radio value="drop">删除重建</el-radio>
              <el-radio value="truncate">清空数据</el-radio>
              <el-radio value="backup">备份后重建</el-radio>
            </el-radio-group>
          </div>
          <div class="config-box">
            <h4>说明</h4>
            <p><strong>智能自适应配置</strong></p>
            <p>系统将根据 CPU 核心数、内存、表大小、记录数自动优化同步参数。</p>
            <p><strong>全量同步：</strong>多线程并行 + 批次大小自适应</p>
            <p><strong>增量同步：</strong>Binlog 监听单线程 + 事件消费自动优化</p>
            <p><strong>结构同步：</strong>对比结构差异，自动执行 ALTER</p>
          </div>
        </div>
      </section>

      <section v-else class="wizard-step-panel">
        <div class="config-overview-grid">
          <div class="config-box">
            <h4>任务与数据源</h4>
            <p><strong>任务：</strong>{{ task?.name || '-' }}</p>
            <p><strong>源数据源：</strong>{{ sourceName }}</p>
            <p><strong>目标数据源：</strong>{{ targetNames.join('、') || '-' }}</p>
            <p><strong>数据库数：</strong>{{ form.selected_databases.length }}<span v-if="modifiedDbCount > 0" class="muted clickable" @click="detailDialogType = 'database'; detailDialogVisible = true">（修改 {{ modifiedDbCount }} 个）</span></p>
            <p><strong>表数量：</strong>{{ totalTables }}<span v-if="modifiedTableCount > 0" class="muted clickable" @click="detailDialogType = 'table'; detailDialogVisible = true">（修改 {{ modifiedTableCount }} 张）</span></p>
          </div>
          <div class="config-box">
            <h4>同步参数</h4>
            <p><strong>模式：</strong>{{ syncModeLabel }}</p>
            <p><strong>错误策略：</strong>{{ form.sync_config.error_strategy === 'skip' ? '跳过错误' : '遇错暂停' }}</p>
            <p><strong>目标表策略：</strong>{{ tableStrategyLabel }}</p>
            <p><strong>仅同步结构：</strong>{{ form.sync_config.sync_mode === 'structure' ? '是' : '否' }}</p>
          </div>
        </div>

        <div class="config-box table-list-box">
          <h4>配置预览</h4>
          <div v-if="form.selected_databases.length === 0" class="empty-inline">暂无选表</div>
          <div v-for="db in form.selected_databases" :key="`${db.source_database}-${db.database}`" class="db-block">
            <div class="db-title step4-db-header" @click="toggleStep4Db(db.source_database)">
              <i class="bi" :class="expandedDbs[db.source_database] ? 'bi-chevron-down' : 'bi-chevron-right'"></i>
              {{ db.database }}
              <span v-if="db.is_database_modified" class="muted">(原：{{ db.source_database }})</span>
            </div>
            <ul v-show="expandedDbs[db.source_database]">
              <li v-for="table in db.tables" :key="`${db.source_database}-${table.source_table}`">
                {{ table.target_table }}
                <span v-if="table.is_modified" class="muted">(原：{{ table.source_table }})</span>
              </li>
            </ul>
          </div>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="wizard-footer">
        <el-button @click="visible = false">取消</el-button>
        <div>
          <el-button v-if="step > 0" @click="step--">上一步</el-button>
          <el-button v-if="step < 3" type="primary" @click="nextStep">下一步</el-button>
          <el-button v-else type="success" :loading="submitting" @click="submit">完成配置</el-button>
        </div>
      </div>
    </template>
  </el-dialog>

  <el-dialog v-model="fieldDialogVisible" title="选择字段" width="700px" destroy-on-close>
    <template v-if="fieldDialogColumns.length > 0">
      <div class="batch-toolbar">
        <el-button size="small" @click="selectAllFields">全选</el-button>
        <el-button size="small" @click="clearAllFields">取消全选</el-button>
        <span class="muted">已选择 {{ fieldDialogSelected.length }} / {{ fieldDialogColumns.length }}</span>
      </div>
      <div class="field-list-box">
        <el-checkbox-group v-model="fieldDialogSelected" class="table-check-grid">
          <el-checkbox v-for="column in fieldDialogColumns" :key="column.name" :label="column.name" :disabled="column.is_primary">
            <i v-if="column.is_primary" class="bi bi-key-fill text-warning me-1"></i>
            {{ column.name }}
            <span v-if="column.is_primary" class="muted">(主键)</span>
          </el-checkbox>
        </el-checkbox-group>
      </div>
    </template>
    <template #footer>
      <el-button @click="fieldDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="applyFieldSelection">确定</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="detailDialogVisible" :title="detailDialogType === 'database' ? '修改的数据库' : '修改的表'" width="720px" destroy-on-close>
    <template v-if="detailDialogType === 'database'">
      <div class="detail-list-box">
        <div v-for="db in modifiedDatabases" :key="db.source_database" class="detail-list-item">
          <i class="bi bi-database me-2 text-primary"></i>
          <strong>{{ db.database }}</strong>
          <span class="muted ms-2">(原：{{ db.source_database }})</span>
        </div>
      </div>
    </template>
    <template v-else>
      <div class="detail-list-box">
        <div v-for="item in modifiedTables" :key="`${item.db}-${item.source}`" class="detail-list-item column">
          <div><strong>{{ item.db }}</strong></div>
          <div>
            <i class="bi bi-table me-2 text-success"></i>
            <strong>{{ item.target }}</strong>
            <span class="muted ms-2">(原：{{ item.source }})</span>
          </div>
        </div>
      </div>
    </template>
    <template #footer>
      <el-button type="primary" @click="detailDialogVisible = false">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { datasourceApi } from '../../api/datasource'
import { taskConfigApi } from '../../api/task-config'
import type {
  DataSourceItem,
  DatabaseSelectionItem,
  DatabaseTablesNode,
  SyncConfigPayload,
  TableColumnInfo,
  TaskConfigViewResult,
  UpdateTaskConfigPayload
} from '../../types/task-config'
import type { TaskItem } from '../../types/task-monitor'

const props = defineProps<{
  modelValue: boolean
  task: TaskItem | null
  dataSources: DataSourceItem[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

const step = ref(0)
const submitting = ref(false)
const databaseLoading = ref(false)
const databaseOptions = ref<DatabaseTablesNode[]>([])
const showBatchEdit = ref(false)
const batchPrefix = ref('')
const batchSuffix = ref('')
const replaceOld = ref('')
const replaceNew = ref('')
const expandedSourceDbs = reactive<Record<string, boolean>>({})
const expandedSelectedDbs = reactive<Record<string, boolean>>({})
const fieldDialogVisible = ref(false)
const fieldDialogColumns = ref<TableColumnInfo[]>([])
const fieldDialogDb = ref('')
const fieldDialogTable = ref('')
const fieldDialogSelected = ref<string[]>([])
const detailDialogVisible = ref(false)
const detailDialogType = ref<'database' | 'table'>('database')

const form = reactive<UpdateTaskConfigPayload>({
  source_id: '',
  target_id: '',
  target_ids: [],
  selected_databases: [],
  sync_config: {
    sync_mode: 'full',
    error_strategy: 'skip',
    table_exists_strategy: 'truncate',
    sync_structure_only: false
  }
})

const sourceOptions = computed(() => props.dataSources.filter((item) => item.type === props.task?.source_type))
const targetOptions = computed(() => props.dataSources.filter((item) => item.type === props.task?.target_type && item.id !== form.source_id))
const sourceName = computed(() => props.dataSources.find((item) => item.id === form.source_id)?.name || '-')
const targetNames = computed(() => props.dataSources.filter((item) => form.target_ids.includes(item.id)).map((item) => item.name))
const totalTables = computed(() => form.selected_databases.reduce((sum, db) => sum + db.tables.length, 0))
const syncModeLabel = computed(() => {
  if (form.sync_config.sync_mode === 'incremental') return '增量同步'
  if (form.sync_config.sync_mode === 'structure') return '结构同步'
  return '全量同步'
})
const dialogTitle = computed(() => `${props.task?.name || '任务'} - 配置向导`)
const modifiedDbCount = computed(() => form.selected_databases.filter((db) => db.is_database_modified).length)
const modifiedTableCount = computed(() => form.selected_databases.reduce((sum, db) => sum + db.tables.filter((t) => t.is_modified).length, 0))
const modifiedDatabases = computed(() => form.selected_databases.filter((db) => db.is_database_modified))
const modifiedTables = computed(() => {
  const list: Array<{ db: string; source: string; target: string }> = []
  form.selected_databases.forEach((db) => {
    db.tables.filter((t) => t.is_modified).forEach((t) => {
      list.push({ db: db.database, source: t.source_table, target: t.target_table })
    })
  })
  return list
})
const tableStrategyLabel = computed(() => {
  const map: Record<string, string> = { drop: '删除重建', truncate: '清空数据', backup: '备份后重建' }
  return map[form.sync_config.table_exists_strategy] || form.sync_config.table_exists_strategy
})
const expandedDbs = reactive<Record<string, boolean>>({})

function toggleStep4Db(dbKey: string) {
  expandedDbs[dbKey] = !expandedDbs[dbKey]
}

function toggleTarget(id: string) {
  const idx = form.target_ids.indexOf(id)
  if (idx >= 0) {
    form.target_ids.splice(idx, 1)
  } else {
    form.target_ids.push(id)
  }
}

watch(
  () => visible.value,
  async (value) => {
    if (value) {
      await initialize()
    }
  }
)

watch(
  () => form.sync_config.sync_mode,
  (mode) => {
    form.sync_config.sync_structure_only = mode === 'structure'
  }
)

async function initialize() {
  step.value = 0
  databaseOptions.value = []
  Object.keys(expandedSourceDbs).forEach((key) => delete expandedSourceDbs[key])
  Object.keys(expandedSelectedDbs).forEach((key) => delete expandedSelectedDbs[key])
  form.source_id = ''
  form.target_id = ''
  form.target_ids = []
  form.selected_databases = []
  form.sync_config = defaultSyncConfig()

  if (!props.task?.id) return
  try {
    const detail = await taskConfigApi.getConfigView(props.task.id)
    applyConfig(detail)
    if (form.source_id) {
      await loadDatabaseTables()
    }
  } catch {
    // 初次未配置时允许空白开始
  }
}

function defaultSyncConfig(): SyncConfigPayload {
  return {
    sync_mode: 'full',
    error_strategy: 'skip',
    table_exists_strategy: 'truncate',
    sync_structure_only: false
  }
}

function applyConfig(detail: TaskConfigViewResult) {
  const config = detail.config
  form.source_id = config?.source_id || ''
  form.target_ids = config?.target_ids?.length ? [...config.target_ids] : config?.target_id ? [config.target_id] : []
  form.target_id = form.target_ids[0] || ''
  form.selected_databases = (config?.selected_databases || []).map((db) => ({
    database: db.database,
    source_database: db.source_database,
    is_database_modified: !!db.is_database_modified,
    tables: (db.tables || []).map((table) => ({
      source_table: table.source_table,
      target_table: table.target_table,
      is_modified: !!table.is_modified,
      selected_fields: table.selected_fields || []
    }))
  }))
  form.sync_config = {
    ...defaultSyncConfig(),
    ...(config?.sync_config || {})
  }
}

async function handleSourceChange() {
  form.target_ids = form.target_ids.filter((id) => id !== form.source_id)
  form.selected_databases = []
  await loadDatabaseTables()
}

async function loadDatabaseTables() {
  if (!form.source_id) return
  databaseLoading.value = true
  try {
    const data = await datasourceApi.getDatabaseTables(form.source_id)
    databaseOptions.value = (data || []).filter((item) => item.tables && item.tables.length > 0)
    databaseOptions.value.forEach((item) => {
      if (expandedSourceDbs[item.database] === undefined) expandedSourceDbs[item.database] = false
    })
  } finally {
    databaseLoading.value = false
  }
}

function toggleSourceDb(database: string) {
  expandedSourceDbs[database] = !expandedSourceDbs[database]
}

function toggleSelectedDb(database: string) {
  expandedSelectedDbs[database] = !expandedSelectedDbs[database]
}

function isTableSelected(database: string, table: string) {
  return !!form.selected_databases.find((db) => db.source_database === database)?.tables.find((item) => item.source_table === table)
}

function isDatabaseFullySelected(database: string) {
  const dbNode = databaseOptions.value.find((item) => item.database === database)
  if (!dbNode || dbNode.tables.length === 0) return false
  return dbNode.tables.every((table) => isTableSelected(database, table))
}

function isDatabasePartiallySelected(database: string) {
  const dbNode = databaseOptions.value.find((item) => item.database === database)
  if (!dbNode || dbNode.tables.length === 0) return false
  const selected = dbNode.tables.filter((table) => isTableSelected(database, table))
  return selected.length > 0 && selected.length < dbNode.tables.length
}

function toggleDatabase(database: string, checked: string | number | boolean) {
  const dbNode = databaseOptions.value.find((item) => item.database === database)
  if (!dbNode) return
  if (checked) {
    expandedSourceDbs[database] = true
    expandedSelectedDbs[database] = true
    dbNode.tables.forEach((table) => {
      const db = ensureDatabaseSelection(database)
      const exists = db.tables.findIndex((item) => item.source_table === table)
      if (exists < 0) {
        db.tables.push({ source_table: table, target_table: table, is_modified: false, selected_fields: [] })
      }
    })
  } else {
    form.selected_databases = form.selected_databases.filter((item) => item.source_database !== database)
  }
}

function selectAllLeft() {
  databaseOptions.value.forEach((db) => {
    toggleDatabase(db.database, true)
  })
}

function clearAllMappings() {
  form.selected_databases = []
}

function batchAddPrefix() {
  if (!batchPrefix.value) return
  form.selected_databases.forEach((db) => {
    db.tables.forEach((table) => {
      table.target_table = batchPrefix.value + table.source_table
      table.is_modified = true
    })
  })
}

function batchAddSuffix() {
  if (!batchSuffix.value) return
  form.selected_databases.forEach((db) => {
    db.tables.forEach((table) => {
      table.target_table = table.source_table + batchSuffix.value
      table.is_modified = true
    })
  })
}

function batchReplacePrefix() {
  if (!replaceOld.value) return
  form.selected_databases.forEach((db) => {
    db.tables.forEach((table) => {
      if (table.source_table.startsWith(replaceOld.value)) {
        table.target_table = replaceNew.value + table.source_table.substring(replaceOld.value.length)
        table.is_modified = table.target_table !== table.source_table
      }
    })
  })
}

function batchReplaceSuffix() {
  if (!replaceOld.value) return
  form.selected_databases.forEach((db) => {
    db.tables.forEach((table) => {
      if (table.source_table.endsWith(replaceOld.value)) {
        table.target_table = table.source_table.substring(0, table.source_table.length - replaceOld.value.length) + replaceNew.value
        table.is_modified = table.target_table !== table.source_table
      }
    })
  })
}

async function openFieldDialog(database: string, table: string) {
  if (!form.source_id) {
    ElMessage.warning('请先选择源数据源')
    return
  }
  const db = form.selected_databases.find((item) => item.source_database === database)
  const target = db?.tables.find((item) => item.source_table === table)
  if (!target) return
  try {
    fieldDialogColumns.value = await datasourceApi.getTableColumns(form.source_id, database, table)
    fieldDialogDb.value = database
    fieldDialogTable.value = table
    const primaryFields = fieldDialogColumns.value.filter((item) => item.is_primary).map((item) => item.name)
    fieldDialogSelected.value = Array.from(new Set([...(target.selected_fields || []), ...primaryFields]))
    fieldDialogVisible.value = true
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '获取字段失败')
  }
}

function selectAllFields() {
  fieldDialogSelected.value = fieldDialogColumns.value.map((item) => item.name)
}

function clearAllFields() {
  fieldDialogSelected.value = fieldDialogColumns.value.filter((item) => item.is_primary).map((item) => item.name)
}

function applyFieldSelection() {
  const db = form.selected_databases.find((item) => item.source_database === fieldDialogDb.value)
  const target = db?.tables.find((item) => item.source_table === fieldDialogTable.value)
  if (target) {
    target.selected_fields = [...fieldDialogSelected.value]
  }
  fieldDialogVisible.value = false
}

function toggleTable(database: string, table: string, checked: string | number | boolean) {
  const db = ensureDatabaseSelection(database)
  const exists = db.tables.findIndex((item) => item.source_table === table)
  if (checked && exists < 0) {
    db.tables.push({
      source_table: table,
      target_table: table,
      is_modified: false,
      selected_fields: []
    })
  }
  if (!checked && exists >= 0) {
    db.tables.splice(exists, 1)
  }
  if (db.tables.length === 0) {
    form.selected_databases = form.selected_databases.filter((item) => item.source_database !== database)
  }
}

function ensureDatabaseSelection(database: string): DatabaseSelectionItem {
  let current = form.selected_databases.find((item) => item.source_database === database)
  if (!current) {
    current = {
      database,
      source_database: database,
      is_database_modified: false,
      tables: []
    }
    form.selected_databases.push(current)
  }
  expandedSelectedDbs[database] = true
  return current
}

function removeTable(database: string, table: string) {
  const db = form.selected_databases.find((item) => item.source_database === database)
  if (!db) return
  db.tables = db.tables.filter((item) => item.source_table !== table)
  if (db.tables.length === 0) {
    form.selected_databases = form.selected_databases.filter((item) => item.source_database !== database)
  }
}

function nextStep() {
  if (step.value === 0) {
    if (!form.source_id) {
      ElMessage.warning('请选择源数据源')
      return
    }
    if (form.target_ids.length === 0) {
      ElMessage.warning('请至少选择一个目标数据源')
      return
    }
    form.target_id = form.target_ids[0]
  }
  if (step.value === 1 && totalTables.value === 0) {
    ElMessage.warning('请至少选择一张表')
    return
  }
  step.value += 1
}

async function submit() {
  if (!props.task?.id) return
  submitting.value = true
  try {
    form.target_id = form.target_ids[0] || ''
    await taskConfigApi.updateConfig(props.task.id, {
      ...form,
      sync_config: {
        ...form.sync_config,
        sync_structure_only: form.sync_config.sync_mode === 'structure'
      }
    })
    ElMessage.success('配置保存成功')
    emit('saved')
    visible.value = false
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    submitting.value = false
  }
}

function handleClosed() {
  step.value = 0
}
</script>
