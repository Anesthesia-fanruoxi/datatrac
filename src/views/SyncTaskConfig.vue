<template>
  <div class="page-container">
    <div class="header-actions">
      <n-space>
        <n-radio-group v-model:value="viewMode" size="medium">
          <n-radio-button value="card">
            <n-icon><GridIcon /></n-icon>
          </n-radio-button>
          <n-radio-button value="list">
            <n-icon><ListIcon /></n-icon>
          </n-radio-button>
        </n-radio-group>
        <n-button type="primary" @click="handleCreate">
          <template #icon>
            <n-icon><AddIcon /></n-icon>
          </template>
          创建任务
        </n-button>
      </n-space>
    </div>

    <!-- 卡片视图 -->
    <n-grid v-if="viewMode === 'card'" :cols="3" :x-gap="16" :y-gap="16">
      <n-grid-item v-for="item in tasks" :key="item.id">
        <n-card 
          :title="item.name" 
          hoverable 
          class="task-card"
          @click="handleEdit(item)"
        >
          <template #header-extra>
            <n-tag :type="getStatusType(item.status)" size="small">
              {{ getStatusText(item.status) }}
            </n-tag>
          </template>
          
          <div class="card-content">
            <div class="sync-direction">
              <n-tag type="info" size="small" round>{{ item.sourceType.toUpperCase() }}</n-tag>
              <n-icon size="18"><ArrowIcon /></n-icon>
              <n-tag type="warning" size="small" round>{{ item.targetType.toUpperCase() }}</n-tag>
            </div>
            <div class="info-item">
              <n-icon><TimeIcon /></n-icon>
              <n-text depth="3">创建于 {{ formatDate(item.createdAt) }}</n-text>
            </div>
          </div>

          <template #action>
            <div class="card-actions" @click.stop>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-button size="small" quaternary @click="handleMonitor(item)">
                    <template #icon><n-icon><MonitorIcon /></n-icon></template>
                  </n-button>
                </template>
                任务监控
              </n-tooltip>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-button size="small" quaternary @click="handleEdit(item)">
                    <template #icon><n-icon><EditIcon /></n-icon></template>
                  </n-button>
                </template>
                编辑任务
              </n-tooltip>
              <n-popconfirm @positive-click="handleDelete(item.id)">
                <template #trigger>
                  <n-button size="small" quaternary type="error" @click.stop>
                    <template #icon><n-icon><TrashIcon /></n-icon></template>
                  </n-button>
                </template>
                确定要删除该任务吗？
              </n-popconfirm>
            </div>
          </template>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- 列表视图 -->
    <n-data-table
      v-else
      :columns="columns"
      :data="tasks"
      :loading="loading"
      :pagination="pagination"
      :bordered="false"
      @row-click="handleEdit"
      class="task-table"
    />

    <!-- 任务配置向导 -->
    <SyncTaskWizard
      v-if="showWizard"
      :task-id="editingTaskId"
      @close="showWizard = false"
      @success="handleWizardSuccess"
      @created="loadTasks"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue';
import { useRouter } from 'vue-router';
import { 
  NButton, NDataTable, NIcon, NTag, NSpace, NPopconfirm, 
  NGrid, NGridItem, NCard, NText, NRadioGroup, NRadioButton, NTooltip 
} from 'naive-ui';
import { 
  Add as AddIcon, 
  Grid as GridIcon,
  List as ListIcon,
  Sync as ArrowIcon,
  Time as TimeIcon,
  StatsChart as MonitorIcon,
  Create as EditIcon,
  Trash as TrashIcon
} from '@vicons/ionicons5';
import { taskApi } from '../api/task';
import type { SyncTask } from '../types';
import { showSuccess, showError } from '../utils/message';
import { formatDate } from '../utils/format';
import SyncTaskWizard from '../components/SyncTaskWizard/index.vue';

const tasks = ref<SyncTask[]>([]);
const viewMode = ref<'card' | 'list'>('card');
const loading = ref(false);
const showWizard = ref(false);
const editingTaskId = ref<string | undefined>();
const router = useRouter();

const pagination = {
  pageSize: 10
};

const getStatusType = (status: string): any => {
  const typeMap: Record<string, string> = {
    idle: 'default',
    running: 'primary',
    paused: 'warning',
    completed: 'success',
    failed: 'error'
  };
  return typeMap[status] || 'default';
};

const getStatusText = (status: string): string => {
  const textMap: Record<string, string> = {
    idle: '待机',
    running: '运行中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败'
  };
  return textMap[status] || status;
};

const columns = [
  { title: '任务名称', key: 'name' },
  { 
    title: '同步方向', 
    key: 'direction',
    render(row: SyncTask) {
      return h(NSpace, null, {
        default: () => [
          h(NTag, { type: 'info', size: 'small' }, { default: () => row.sourceType.toUpperCase() }),
          h(NIcon, { size: 16, style: 'vertical-align: middle' }, { default: () => h(ArrowIcon) }),
          h(NTag, { type: 'warning', size: 'small' }, { default: () => row.targetType.toUpperCase() })
        ]
      });
    }
  },
  { 
    title: '状态', 
    key: 'status',
    render(row: SyncTask) {
      return h(NTag, { type: getStatusType(row.status) }, { default: () => getStatusText(row.status) });
    }
  },
  { 
    title: '创建时间', 
    key: 'createdAt',
    render: (row: SyncTask) => formatDate(row.createdAt)
  },
  {
    title: '操作',
    key: 'actions',
    render(row: SyncTask) {
      return h(NSpace, null, {
        default: () => [
          h(NButton, { size: 'small', quaternary: true, onClick: (e) => { e.stopPropagation(); handleMonitor(row); } }, { icon: () => h(MonitorIcon) }),
          h(NButton, { size: 'small', quaternary: true, onClick: (e) => { e.stopPropagation(); handleEdit(row); } }, { icon: () => h(EditIcon) }),
          h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
            trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error', onClick: (e) => e.stopPropagation() }, { icon: () => h(TrashIcon) }),
            default: () => '确定要删除该任务吗？'
          })
        ]
      });
    }
  }
];

async function loadTasks() {
  loading.value = true;
  try {
    tasks.value = await taskApi.list();
  } catch (e: any) {
    showError('加载任务列表失败: ' + e);
  } finally {
    loading.value = false;
  }
}

function handleCreate() {
  editingTaskId.value = undefined;
  showWizard.value = true;
}

function handleEdit(task: SyncTask) {
  editingTaskId.value = task.id;
  showWizard.value = true;
}

function handleMonitor(task: SyncTask) {
  router.push({ name: 'TaskMonitor', query: { taskId: task.id } });
}

async function handleDelete(id: string) {
  try {
    await taskApi.delete(id);
    showSuccess('删除成功');
    loadTasks();
  } catch (e: any) {
    showError('删除失败: ' + e);
  }
}

function handleWizardSuccess() {
  showWizard.value = false;
  loadTasks();
}

onMounted(loadTasks);
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.header-actions {
  display: flex;
  justify-content: flex-end;
}

.task-card {
  cursor: pointer;
  transition: all 0.3s ease;
}

.task-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.card-content {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sync-direction {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: center;
  padding: 8px;
  background: #f5f7f9;
  border-radius: 8px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.task-table :deep(.n-data-table-tr) {
  cursor: pointer;
}
</style>
