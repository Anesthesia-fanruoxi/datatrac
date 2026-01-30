<template>
  <n-modal
    :show="true"
    preset="card"
    :title="modalTitle"
    style="width: 900px"
    :closable="true"
    :mask-closable="false"
    @close="$emit('close')"
  >
    <!-- 第一阶段：任务名称与类型选择 -->
    <template v-if="!isTypeConfirmed">
      <Step1SelectDataSourceType
        v-model:task-name="taskModel.name"
        v-model:source-type="taskModel.sourceType"
        v-model:target-type="taskModel.targetType"
        :is-edit="!!taskId"
      />
    </template>

    <!-- 第二阶段：分步配置向导 -->
    <template v-else>
      <n-steps :current="currentStep" :status="stepStatus" style="margin-bottom: 24px">
        <n-step title="数据源" description="选择具体实例" />
        <n-step title="读取配置" description="选择数据范围" />
        <n-step title="写入配置" description="设置写入策略" />
        <n-step title="性能配置" description="设置并发与策略" />
        <n-step title="预览确认" description="核对配置" />
      </n-steps>

      <div class="step-content">
        <!-- 步骤1：数据源实例选择（已锁定类型） -->
        <div v-if="currentStep === 1">
          <n-form label-placement="left" label-width="100">
            <n-form-item label="任务名称">
              <n-input v-model:value="taskModel.name" disabled />
            </n-form-item>
            <n-grid :cols="2" :x-gap="24">
              <n-form-item-gi :label="`源 ${taskModel.sourceType.toUpperCase()}`" required>
                <n-select
                  v-model:value="taskModel.sourceId"
                  :options="sourceOptions"
                  placeholder="请选择源实例"
                />
              </n-form-item-gi>
              <n-form-item-gi :label="`目标 ${taskModel.targetType.toUpperCase()}`" required>
                <n-select
                  v-model:value="taskModel.targetId"
                  :options="targetOptions"
                  placeholder="请选择目标实例"
                />
              </n-form-item-gi>
            </n-grid>
          </n-form>
        </div>

        <!-- 后续步骤保持不变 -->
        <div v-else-if="currentStep === 2">
          <component
            :is="sourceComponent"
            :data-source-id="taskModel.sourceId"
            v-model="sourceConfig"
          />
        </div>

        <div v-else-if="currentStep === 3">
          <component
            :is="targetComponent"
            :data-source-id="taskModel.targetId"
            v-model="targetConfig"
          />
        </div>

        <div v-else-if="currentStep === 4">
          <PipelineConfig v-model="pipelineConfig" />
        </div>

        <div v-else-if="currentStep === 5">
          <n-descriptions title="任务预览" bordered :column="2">
            <n-descriptions-item label="任务名称">{{ taskModel.name }}</n-descriptions-item>
            <n-descriptions-item label="同步模式">
              <n-tag type="info">{{ taskModel.sourceType.toUpperCase() }}</n-tag>
              <span style="margin: 0 8px">→</span>
              <n-tag type="warning">{{ taskModel.targetType.toUpperCase() }}</n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="同步对象数量">
              {{ selectedObjectsCount }} 个单元
            </n-descriptions-item>
            <n-descriptions-item label="并行线程">{{ pipelineConfig.threadCount }}</n-descriptions-item>
            <n-descriptions-item label="批次大小">{{ pipelineConfig.batchSize }}</n-descriptions-item>
            <n-descriptions-item label="错误策略">{{ pipelineConfig.errorStrategy === 'skip' ? '跳过' : '暂停' }}</n-descriptions-item>
          </n-descriptions>
        </div>
      </div>
    </template>

    <template #footer>
      <div class="footer-actions">
        <n-button @click="$emit('close')">取消</n-button>
        <n-space>
          <!-- 类型选择阶段按钮 -->
          <template v-if="!isTypeConfirmed">
            <n-button 
              type="primary" 
              :disabled="!canConfirmType" 
              :loading="creating"
              @click="handleCreateBaseTask"
            >
              创建任务并配置
            </n-button>
          </template>
          
          <!-- 向导阶段按钮 -->
          <template v-else>
            <n-button v-if="currentStep === 1" @click="isTypeConfirmed = false" disabled>
              类型已锁定
            </n-button>
            <n-button v-if="currentStep > 1" @click="currentStep--">上一步</n-button>
            <n-button v-if="currentStep < 5" type="primary" @click="handleNext">下一步</n-button>
            <n-button v-if="currentStep === 5" type="primary" :loading="submitting" @click="handleFinish">
              保存配置
            </n-button>
          </template>
        </n-space>
      </div>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, defineAsyncComponent } from 'vue';
import { 
  NModal, NSteps, NStep, NForm, NFormItem, NFormItemGi, NGrid, 
  NInput, NSelect, NButton, NSpace, NTag, NDescriptions, NDescriptionsItem 
} from 'naive-ui';
import { datasourceApi } from '../../api/datasource';
import { taskApi } from '../../api/task';
import type { DataSource, DataSourceType } from '../../types';
import { showSuccess, showError } from '../../utils/message';
import PipelineConfig from './shared/PipelineConfig.vue';
import Step1SelectDataSourceType from './Step1SelectDataSourceType.vue';

const props = defineProps<{
  taskId?: string;
}>();

const emit = defineEmits(['close', 'success', 'created']);

const currentStep = ref(1);
const stepStatus = ref<any>('process');
const submitting = ref(false);
const creating = ref(false);
const dataSources = ref<DataSource[]>([]);
const isTypeConfirmed = ref(false);
const currentTaskId = ref<string | undefined>(props.taskId);

// 任务基本信息
const taskModel = reactive({
  name: '',
  sourceId: '',
  targetId: '',
  sourceType: '' as DataSourceType | '',
  targetType: '' as DataSourceType | '',
  status: 'idle',
  createdAt: '',
  updatedAt: ''
});

// 源配置
const sourceConfig = ref<any>({
  database: '',
  selectedTables: [],
  selectedIndices: [],
  dbNameTransform: { enabled: false, mode: 'prefix', sourcePattern: '', targetPattern: '' }
});

// 目标配置
const targetConfig = ref<any>({
  tableExistsStrategy: 'truncate',
  bulkInsertSize: 1000,
  writeMode: 'index',
  shards: 1,
  replicas: 1
});

// 流水线配置
const pipelineConfig = ref({
  batchSize: 1000,
  threadCount: 4,
  errorStrategy: 'skip',
  maxRetries: 3
});

// 动态组件加载
const sourceComponent = computed(() => {
  if (taskModel.sourceType === 'mysql') return defineAsyncComponent(() => import('./SourceConfig/MySqlSource.vue'));
  if (taskModel.sourceType === 'elasticsearch') return defineAsyncComponent(() => import('./SourceConfig/ElasticSource.vue'));
  return null;
});

const targetComponent = computed(() => {
  if (taskModel.targetType === 'mysql') return defineAsyncComponent(() => import('./TargetConfig/MySqlTarget.vue'));
  if (taskModel.targetType === 'elasticsearch') return defineAsyncComponent(() => import('./TargetConfig/ElasticTarget.vue'));
  return null;
});

const modalTitle = computed(() => {
  if (currentTaskId.value) return '配置同步任务';
  return '创建同步任务';
});

const canConfirmType = computed(() => {
  return taskModel.name.trim() && taskModel.sourceType && taskModel.targetType;
});

// 过滤后的选项：只显示对应类型的数据源
const sourceOptions = computed(() => 
  dataSources.value
    .filter(d => d.type === taskModel.sourceType)
    .map(d => ({ label: d.name, value: d.id }))
);

const targetOptions = computed(() => 
  dataSources.value
    .filter(d => d.type === taskModel.targetType)
    .map(d => ({ label: d.name, value: d.id }))
);

const selectedObjectsCount = computed(() => {
  if (taskModel.sourceType === 'mysql') return sourceConfig.value.selectedTables.length;
  if (taskModel.sourceType === 'elasticsearch') return sourceConfig.value.selectedIndices.length;
  return 0;
});

async function loadDataSources() {
  try {
    dataSources.value = await datasourceApi.list();
    if (currentTaskId.value) {
      isTypeConfirmed.value = true; // 编辑模式直接进入配置
      const allTasks = await taskApi.list();
      const task = allTasks.find(t => t.id === currentTaskId.value);
      if (task) {
        Object.assign(taskModel, {
          name: task.name,
          sourceId: task.sourceId,
          targetId: task.targetId,
          sourceType: task.sourceType,
          targetType: task.targetType,
          status: task.status,
          createdAt: task.createdAt,
          updatedAt: task.updatedAt
        });
        const config = JSON.parse(task.config || '{}');
        Object.assign(pipelineConfig.value, config);
        Object.assign(sourceConfig.value, config.sourceConfig || {});
        Object.assign(targetConfig.value, config.targetConfig || {});
      }
    }
  } catch (e) {
    showError('获取初始化数据失败');
  }
}

async function handleCreateBaseTask() {
  creating.value = true;
  try {
    const now = new Date().toISOString();
    const taskData: any = {
      id: '',
      name: taskModel.name,
      sourceId: '',
      targetId: '',
      sourceType: taskModel.sourceType,
      targetType: taskModel.targetType,
      status: 'idle',
      config: JSON.stringify({
        ...pipelineConfig.value,
        sourceConfig: sourceConfig.value,
        targetConfig: targetConfig.value
      }),
      createdAt: now,
      updatedAt: now
    };
    
    const id = await taskApi.create(taskData);
    currentTaskId.value = id;
    taskModel.status = 'idle';
    taskModel.createdAt = now;
    taskModel.updatedAt = now;
    isTypeConfirmed.value = true;
    emit('created');
    showSuccess('任务已创建，请继续完善配置');
  } catch (e: any) {
    showError('创建失败: ' + e);
  } finally {
    creating.value = false;
  }
}

function handleNext() {
  if (currentStep.value === 1) {
    if (!taskModel.sourceId || !taskModel.targetId) {
      showError('请选择源和目标数据源实例');
      return;
    }
  }
  if (currentStep.value === 2) {
    if (selectedObjectsCount.value === 0) {
      showError('请至少选择一个同步对象');
      return;
    }
  }
  currentStep.value++;
}

async function handleFinish() {
  submitting.value = true;
  try {
    const finalConfig = {
      ...pipelineConfig.value,
      sourceConfig: sourceConfig.value,
      targetConfig: targetConfig.value,
      selection: taskModel.sourceType === 'mysql' ? sourceConfig.value.selectedTables : sourceConfig.value.selectedIndices
    };

    const taskData: any = {
      id: currentTaskId.value,
      name: taskModel.name,
      sourceId: taskModel.sourceId,
      targetId: taskModel.targetId,
      sourceType: taskModel.sourceType,
      targetType: taskModel.targetType,
      status: taskModel.status || 'idle',
      config: JSON.stringify(finalConfig),
      createdAt: taskModel.createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };

    if (currentTaskId.value) {
      await taskApi.update(currentTaskId.value, taskData);
      showSuccess('更新成功');
    }
    emit('success');
  } catch (e: any) {
    showError('保存失败: ' + e);
  } finally {
    submitting.value = false;
  }
}

onMounted(loadDataSources);
</script>

<style scoped>
.step-content {
  margin-top: 16px;
  min-height: 400px;
}
.footer-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 24px;
}
</style>
