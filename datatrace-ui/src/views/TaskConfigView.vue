<template>
  <section class="page-card">
    <div class="page-header">
      <div>
        <h2>任务配置</h2>
        <p class="page-subtitle">先完成任务的创建、查看与基础管理，后续继续迁移完整配置向导。</p>
      </div>
      <el-button type="primary" @click="createDialogVisible = true">新建同步任务</el-button>
    </div>

    <div class="summary-grid">
      <div class="metric">
        <span>任务总数</span>
        <strong>{{ tasks.length }}</strong>
      </div>
      <div class="metric">
        <span>数据源数量</span>
        <strong>{{ dataSources.length }}</strong>
      </div>
      <div class="metric">
        <span>已配置任务</span>
        <strong>{{ configuredCount }}</strong>
      </div>
      <div class="metric">
        <span>运行中任务</span>
        <strong>{{ runningCount }}</strong>
      </div>
    </div>

    <el-table v-loading="loading" :data="tasks" border style="width: 100%">
      <el-table-column prop="name" label="任务名称" min-width="180" />
      <el-table-column label="同步方向" min-width="180">
        <template #default="scope">
          {{ prettyType(scope.row.source_type) }} → {{ prettyType(scope.row.target_type) }}
        </template>
      </el-table-column>
      <el-table-column label="状态" min-width="120">
        <template #default="scope">
          <el-tag :type="statusTagType(scope.row)">{{ statusLabel(scope.row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="同步模式" min-width="120">
        <template #default="scope">
          {{ modeLabel(scope.row.sync_mode) }}
        </template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="160">
        <template #default="scope">{{ formatDate(scope.row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" min-width="320" fixed="right">
        <template #default="scope">
          <div class="action-row">
            <el-button size="small" type="primary" plain @click="openConfigPlaceholder(scope.row)">
              {{ scope.row.status === 'idle' ? '配置' : '修改配置' }}
            </el-button>
            <el-button size="small" plain @click="openView(scope.row.id)">查看</el-button>
            <el-button size="small" type="success" plain @click="gotoMonitor(scope.row.id)">监控</el-button>
            <el-button size="small" type="danger" plain @click="removeTask(scope.row.id, scope.row.name)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="createDialogVisible" title="新建同步任务" width="520px">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="任务名称">
          <el-input v-model="createForm.name" placeholder="例如：生产环境数据同步" />
        </el-form-item>
        <el-form-item label="源类型">
          <el-select v-model="createForm.source_type" style="width: 100%">
            <el-option label="MySQL" value="mysql" />
            <el-option label="Elasticsearch" value="elasticsearch" />
          </el-select>
        </el-form-item>
        <el-form-item label="目标类型">
          <el-select v-model="createForm.target_type" style="width: 100%">
            <el-option label="MySQL" value="mysql" />
            <el-option label="Elasticsearch" value="elasticsearch" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="createForm.remark" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCreate">创建并继续配置</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="viewDialogVisible" title="查看任务配置" width="900px">
      <template v-if="viewData">
        <div class="config-overview-grid">
          <div class="config-box">
            <h4>基础信息</h4>
            <p><strong>任务名称：</strong>{{ viewData.task.name }}</p>
            <p><strong>同步方向：</strong>{{ prettyType(viewData.task.source_type) }} → {{ prettyType(viewData.task.target_type) }}</p>
            <p><strong>源数据源：</strong>{{ viewData.task.source_conn?.name || '-' }}</p>
            <p><strong>目标数据源：</strong>{{ viewData.task.target_conn?.name || '-' }}</p>
          </div>
          <div class="config-box">
            <h4>同步参数</h4>
            <p><strong>同步模式：</strong>{{ viewData.config?.sync_config?.sync_mode === 'incremental' ? '增量同步' : '全量同步' }}</p>
            <p><strong>错误策略：</strong>{{ viewData.config?.sync_config?.error_strategy || '-' }}</p>
            <p><strong>表存在策略：</strong>{{ viewData.config?.sync_config?.table_exists_strategy || '-' }}</p>
            <p><strong>仅同步结构：</strong>{{ viewData.config?.sync_config?.sync_structure_only ? '是' : '否' }}</p>
          </div>
        </div>

        <div class="config-box table-list-box">
          <h4>数据库与表</h4>
          <template v-if="(viewData.config?.selected_databases || []).length > 0">
            <div v-for="db in viewData.config?.selected_databases || []" :key="`${db.source_database}-${db.database}`" class="db-block">
              <div class="db-title">
                {{ db.database }}
                <span v-if="db.is_database_modified" class="muted">(原：{{ db.source_database }})</span>
              </div>
              <ul>
                <li v-for="table in db.tables || []" :key="`${db.database}-${table.source_table}`">
                  <i v-if="tableExists(db.source_database, table.source_table)" class="bi bi-check-circle-fill text-success me-1"></i>
                  <i v-else class="bi bi-exclamation-triangle-fill text-warning me-1" title="表可能已被删除或重命名"></i>
                  {{ table.target_table || table.source_table }}
                  <span v-if="table.is_modified" class="muted">(原：{{ table.source_table }})</span>
                </li>
              </ul>
            </div>
          </template>
          <div v-else class="empty-inline">暂无表配置</div>
        </div>
      </template>
      <template #footer>
        <el-button @click="viewDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="openConfigFromView">修改配置</el-button>
      </template>
    </el-dialog>

    <TaskWizardDialog v-model="wizardVisible" :task="wizardTask" :data-sources="dataSources" @saved="handleWizardSaved" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { datasourceApi } from '../api/datasource'
import { taskConfigApi } from '../api/task-config'
import TaskWizardDialog from '../components/task-config/TaskWizardDialog.vue'
import type { CreateTaskRequest, DataSourceItem, TaskConfigViewResult } from '../types/task-config'
import type { TaskItem } from '../types/task-monitor'

const router = useRouter()

const loading = ref(false)
const createDialogVisible = ref(false)
const viewDialogVisible = ref(false)
const wizardVisible = ref(false)
const tasks = ref<TaskItem[]>([])
const dataSources = ref<DataSourceItem[]>([])
const viewData = ref<TaskConfigViewResult | null>(null)
const currentViewTaskId = ref<string>('')
const wizardTask = ref<TaskItem | null>(null)

const createForm = reactive<CreateTaskRequest>({
  name: '',
  source_type: 'mysql',
  target_type: 'mysql',
  remark: ''
})

const configuredCount = computed(() => tasks.value.filter((item) => item.status === 'configured' || item.status === 'completed').length)
const runningCount = computed(() => tasks.value.filter((item) => item.is_running).length)

async function loadPageData() {
  loading.value = true
  try {
    const [taskList, dsList] = await Promise.all([taskConfigApi.list(), datasourceApi.list()])
    tasks.value = taskList
    dataSources.value = dsList
  } finally {
    loading.value = false
  }
}

function prettyType(type: string) {
  if (type === 'mysql') return 'MySQL'
  if (type === 'elasticsearch') return 'Elasticsearch'
  return type || '-'
}

function statusLabel(task: TaskItem) {
  if (task.is_running) return '运行中'
  const map: Record<string, string> = {
    idle: '未配置',
    configured: '已配置',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败'
  }
  return map[task.status] || task.status || '未知'
}

function statusTagType(task: TaskItem) {
  if (task.is_running) return 'success'
  if (task.status === 'configured') return 'info'
  if (task.status === 'paused') return 'warning'
  if (task.status === 'completed') return 'primary'
  if (task.status === 'failed') return 'danger'
  return 'info'
}

function modeLabel(mode?: string) {
  if (mode === 'incremental') return '增量同步'
  if (mode === 'structure') return '结构同步'
  if (mode === 'full') return '全量同步'
  return mode || '-'
}

async function submitCreate() {
  if (!createForm.name.trim()) {
    ElMessage.warning('请输入任务名称')
    return
  }
  try {
    const created = await taskConfigApi.create({ ...createForm, name: createForm.name.trim() })
    createDialogVisible.value = false
    ElMessage.success('任务创建成功')
    await loadPageData()
    openWizard(created.id)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建失败')
  }
}

async function removeTask(taskId: string, taskName: string) {
  try {
    await ElMessageBox.confirm(`确定要删除任务“${taskName}”吗？此操作不可恢复。`, '删除确认', { type: 'warning' })
    await taskConfigApi.remove(taskId)
    ElMessage.success('删除成功')
    await loadPageData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error instanceof Error ? error.message : '删除失败')
    }
  }
}

async function openView(taskId: string) {
  try {
    currentViewTaskId.value = taskId
    viewData.value = await taskConfigApi.getConfigView(taskId)
    viewDialogVisible.value = true
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载失败')
  }
}

function openWizard(taskId: string) {
  const task = tasks.value.find((item) => item.id === taskId)
  if (!task) return
  wizardTask.value = task
  wizardVisible.value = true
}

function openConfigPlaceholder(task: Pick<TaskItem, 'id' | 'name'>) {
  openWizard(task.id)
}

function openConfigFromView() {
  viewDialogVisible.value = false
  openWizard(currentViewTaskId.value)
}

function gotoMonitor(taskId: string) {
  router.push({ name: 'task-monitor', params: { id: taskId } })
}

async function handleWizardSaved() {
  await loadPageData()
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function tableExists(sourceDatabase: string, sourceTable: string) {
  const rt = viewData.value?.real_time_info
  if (!rt || !rt[sourceDatabase]) return true
  return rt[sourceDatabase].includes(sourceTable)
}

onMounted(loadPageData)
</script>
