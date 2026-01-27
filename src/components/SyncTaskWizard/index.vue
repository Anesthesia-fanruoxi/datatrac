<template>
  <n-modal
    v-model:show="visible"
    :title="modalTitle"
    preset="card"
    :style="modalStyle"
    :segmented="{ content: 'soft', footer: 'soft' }"
    :closable="false"
    :mask-closable="false"
  >
    <!-- 前置配置：任务名称和数据源类型（不算步骤） -->
    <template v-if="!taskType">
      <Step1SelectDataSourceType
        v-model:task-name="formData.name"
        v-model:source-type="sourceTypeSelected"
        v-model:target-type="targetTypeSelected"
        :is-edit="isEdit"
      />
    </template>

    <!-- 步骤条和内容（选择类型后显示） -->
    <template v-else>
      <!-- 任务头部：显示任务名称和类型 -->
      <TaskHeader
        :task-name="formData.name || ''"
        :source-type="sourceTypeFromTask"
        :target-type="targetTypeFromTask"
        @update:task-name="handleTaskNameUpdate"
      />

      <!-- 步骤条 -->
      <n-steps 
        :current="currentStep" 
        style="margin-bottom: 24px"
        @update:current="handleStepClick"
      >
        <n-step title="选择数据源" />
        <template v-if="taskType === 'elasticsearch-elasticsearch'">
          <n-step title="筛选索引" />
          <n-step title="选择索引" />
          <n-step title="同步配置" />
          <n-step title="确认配置" />
        </template>
        <template v-else>
          <n-step title="选择表/索引" />
          <n-step title="同步配置" />
          <n-step title="确认配置" />
        </template>
      </n-steps>

      <!-- 步骤内容 -->
      <div :style="contentStyle">
        <!-- 步骤1：选择具体数据源 -->
        <DataSourceSelector
          v-if="currentStep === 1"
          v-model:source-id="formData.sourceId"
          v-model:target-id="formData.targetId"
          :source-type="sourceTypeFromTask"
          :target-type="targetTypeFromTask"
          :source-label="sourceLabel"
          :target-label="targetLabel"
        />

        <!-- 步骤2+：动态加载对应类型的步骤组件 -->
        <component
          v-else-if="currentStepComponent"
          :is="currentStepComponent"
          v-bind="currentStepProps"
        />
      </div>
    </template>

    <template #footer>
      <n-space justify="space-between">
        <n-button @click="handleCancel">取消</n-button>
        <n-space>
          <!-- 前置配置阶段的按钮 -->
          <n-button 
            v-if="!taskType" 
            type="primary" 
            @click="handleStartConfig"
            :disabled="!canStartConfig"
          >
            创建任务
          </n-button>
          
          <!-- 步骤阶段的按钮 -->
          <template v-else>
            <n-button v-if="currentStep > 1" @click="prevStep">上一步</n-button>
            <n-button v-if="currentStep < maxSteps" type="primary" @click="nextStep" :disabled="!canGoNext">
              下一步
            </n-button>
            <n-button v-if="currentStep === maxSteps" type="primary" @click="handleSubmit">
              {{ isEdit ? '保存' : '保存任务' }}
            </n-button>
          </template>
        </n-space>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, shallowRef } from 'vue'
import { NModal, NSteps, NStep, NSpace, NButton } from 'naive-ui'
import TaskHeader from './TaskHeader.vue'
import Step1SelectDataSourceType from './Step1SelectDataSourceType.vue'
import DataSourceSelector from './shared/DataSourceSelector.vue'

// MySQL→MySQL 步骤组件
import MySQLToMySQLStep2 from './TaskTypes/MySQLToMySQL/Step2SelectTables.vue'
import MySQLToMySQLStep3 from './TaskTypes/MySQLToMySQL/Step3SyncConfig.vue'
import MySQLToMySQLStep4 from './TaskTypes/MySQLToMySQL/Step4Confirm.vue'

// MySQL→ES 步骤组件
import MySQLToESStep2 from './TaskTypes/MySQLToES/Step2SelectTables.vue'
import MySQLToESStep3 from './TaskTypes/MySQLToES/Step3SyncConfig.vue'
import MySQLToESStep4 from './TaskTypes/MySQLToES/Step4Confirm.vue'

// ES→MySQL 步骤组件
import ESToMySQLStep2 from './TaskTypes/ESToMySQL/Step2SelectIndices.vue'
import ESToMySQLStep3 from './TaskTypes/ESToMySQL/Step3SyncConfig.vue'
import ESToMySQLStep4 from './TaskTypes/ESToMySQL/Step4Confirm.vue'

// ES→ES 步骤组件
import ESToESStep2 from './TaskTypes/ESToES/Step2FilterIndices.vue'
import ESToESStep3 from './TaskTypes/ESToES/Step3SelectIndices.vue'
import ESToESStep4 from './TaskTypes/ESToES/Step4SyncConfig.vue'
import ESToESStep5 from './TaskTypes/ESToES/Step5Confirm.vue'

import type { SyncTask, SyncTaskType, DataSourceType, ESSearchGroup, IndexNameTransform, DbNameTransform } from '../../types'

const props = defineProps<{
  modelValue: boolean
  isEdit: boolean
  formData: Partial<SyncTask>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'submit': [data: Partial<SyncTask>]
  'create': [data: { name: string; sourceType: DataSourceType; targetType: DataSourceType }]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

// 动态标题
const modalTitle = computed(() => {
  if (props.isEdit) {
    return '配置任务' // 编辑模式就是配置任务
  }
  return '创建任务'
})

const taskType = ref<SyncTaskType | ''>('')
const sourceTypeSelected = ref<DataSourceType | ''>('')
const targetTypeSelected = ref<DataSourceType | ''>('')
const currentStep = ref(1)

// 不再自动设置 taskType，只在编辑模式时设置
// 新建模式下，taskType 保持为空，直到用户点击"创建任务"按钮

// 各类型的状态数据
const selectedTables = ref<string[]>([])
const dbNameTransform = ref<DbNameTransform>({
  enabled: false,
  mode: 'prefix',
  sourcePattern: '',
  targetPattern: ''
})
const indexPattern = ref('')
const matchedIndices = ref<string[]>([])
const esSearchGroups = ref<ESSearchGroup[]>([])
const esSelectedIndices = ref<string[]>([])
const esIndexNameTransform = ref<IndexNameTransform>({
  enabled: false,
  mode: 'prefix',
  sourcePattern: '',
  targetPattern: ''
})

// 从任务类型解析源和目标类型
const sourceTypeFromTask = computed<DataSourceType>(() => {
  if (!taskType.value) return 'mysql'
  return taskType.value.split('-')[0] as DataSourceType
})

const targetTypeFromTask = computed<DataSourceType>(() => {
  if (!taskType.value) return 'mysql'
  return taskType.value.split('-')[1] as DataSourceType
})

const sourceLabel = computed(() => {
  const typeMap: Record<DataSourceType, string> = {
    'mysql': '源 MySQL 数据库',
    'elasticsearch': '源 Elasticsearch'
  }
  return typeMap[sourceTypeFromTask.value] || '源数据源'
})

const targetLabel = computed(() => {
  const typeMap: Record<DataSourceType, string> = {
    'mysql': '目标 MySQL 数据库',
    'elasticsearch': '目标 Elasticsearch'
  }
  return typeMap[targetTypeFromTask.value] || '目标数据源'
})

// 当前步骤的组件
const currentStepComponent = computed(() => {
  if (!taskType.value || currentStep.value === 1) return null
  
  const componentMap: Record<string, any> = {
    'mysql-mysql-2': MySQLToMySQLStep2,
    'mysql-mysql-3': MySQLToMySQLStep3,
    'mysql-mysql-4': MySQLToMySQLStep4,
    'mysql-elasticsearch-2': MySQLToESStep2,
    'mysql-elasticsearch-3': MySQLToESStep3,
    'mysql-elasticsearch-4': MySQLToESStep4,
    'elasticsearch-mysql-2': ESToMySQLStep2,
    'elasticsearch-mysql-3': ESToMySQLStep3,
    'elasticsearch-mysql-4': ESToMySQLStep4,
    'elasticsearch-elasticsearch-2': ESToESStep2,
    'elasticsearch-elasticsearch-3': ESToESStep3,
    'elasticsearch-elasticsearch-4': ESToESStep4,
    'elasticsearch-elasticsearch-5': ESToESStep5
  }
  
  const key = `${taskType.value}-${currentStep.value}`
  const component = componentMap[key]
  
  return component || null
})

// 当前步骤组件的 props
const currentStepProps = computed(() => {
  if (!taskType.value || currentStep.value === 1) return {}
  
  // 步骤2的props
  if (currentStep.value === 2) {
    if (taskType.value === 'mysql-mysql') {
      return {
        sourceId: props.formData.sourceId,
        modelValue: selectedTables.value,
        'onUpdate:modelValue': (val: string[]) => { selectedTables.value = val },
        dbNameTransform: dbNameTransform.value,
        'onUpdate:dbNameTransform': (val: DbNameTransform) => { dbNameTransform.value = val }
      }
    } else if (taskType.value === 'mysql-elasticsearch') {
      return {
        sourceId: props.formData.sourceId,
        modelValue: selectedTables.value,
        'onUpdate:modelValue': (val: string[]) => { selectedTables.value = val }
      }
    } else if (taskType.value === 'elasticsearch-mysql') {
      return {
        sourceId: props.formData.sourceId,
        pattern: indexPattern.value,
        'onUpdate:pattern': (val: string) => { indexPattern.value = val },
        matchedIndices: matchedIndices.value,
        'onUpdate:matchedIndices': (val: string[]) => { matchedIndices.value = val }
      }
    } else if (taskType.value === 'elasticsearch-elasticsearch') {
      return {
        sourceId: props.formData.sourceId,
        modelValue: esSearchGroups.value,
        'onUpdate:modelValue': (val: ESSearchGroup[]) => { esSearchGroups.value = val }
      }
    }
  }
  
  // 步骤3的props
  if (currentStep.value === 3) {
    if (taskType.value === 'elasticsearch-elasticsearch') {
      return {
        searchGroups: esSearchGroups.value,
        modelValue: esSelectedIndices.value,
        'onUpdate:modelValue': (val: string[]) => { esSelectedIndices.value = val },
        indexNameTransform: esIndexNameTransform.value,
        'onUpdate:indexNameTransform': (val: IndexNameTransform) => { esIndexNameTransform.value = val }
      }
    } else {
      return {
        name: props.formData.name || '',
        syncConfig: props.formData.syncConfig || {
          threadCount: 4,
          batchSize: 2500,
          errorStrategy: 'skip' as const,
          tableExistsStrategy: 'drop' as const
        },
        'onUpdate:name': (val: string) => { props.formData.name = val },
        'onUpdate:syncConfig': (val: any) => { props.formData.syncConfig = val }
      }
    }
  }
  
  // 步骤4的props
  if (currentStep.value === 4) {
    if (taskType.value === 'elasticsearch-elasticsearch') {
      return {
        name: props.formData.name || '',
        syncConfig: props.formData.syncConfig || {
          threadCount: 4,
          batchSize: 2500,
          errorStrategy: 'skip' as const,
          tableExistsStrategy: 'drop' as const
        },
        'onUpdate:name': (val: string) => { props.formData.name = val },
        'onUpdate:syncConfig': (val: any) => { props.formData.syncConfig = val }
      }
    } else if (taskType.value.startsWith('mysql')) {
      return {
        preview: {
          taskName: props.formData.name || '未命名',
          tableCount: selectedTables.value.length,
          threadCount: props.formData.syncConfig?.threadCount || 4,
          batchSize: props.formData.syncConfig?.batchSize || 1000
        },
        selectedTables: selectedTables.value
      }
    } else if (taskType.value === 'elasticsearch-mysql') {
      return {
        preview: {
          taskName: props.formData.name || '未命名',
          indexCount: matchedIndices.value.length,
          threadCount: props.formData.syncConfig?.threadCount || 4,
          batchSize: props.formData.syncConfig?.batchSize || 1000
        },
        matchedIndices: matchedIndices.value
      }
    }
  }
  
  // 步骤5的props (仅ES→ES)
  if (currentStep.value === 5 && taskType.value === 'elasticsearch-elasticsearch') {
    return {
      preview: {
        taskName: props.formData.name || '未命名',
        searchGroupsCount: esSearchGroups.value.length,
        selectedIndicesCount: esSelectedIndices.value.length,
        threadCount: props.formData.syncConfig?.threadCount || 4,
        batchSize: props.formData.syncConfig?.batchSize || 1000
      },
      selectedIndices: esSelectedIndices.value
    }
  }
  
  return {}
})

const maxSteps = computed(() => {
  return taskType.value === 'elasticsearch-elasticsearch' ? 5 : 4
})

// 前置配置是否可以开始
const canStartConfig = computed(() => {
  return !!(props.formData.name && props.formData.name.trim() && sourceTypeSelected.value && targetTypeSelected.value)
})

const canGoNext = computed(() => {
  // 步骤1：选择具体数据源
  if (currentStep.value === 1) {
    return !!(props.formData.sourceId && props.formData.targetId)
  }
  
  if (taskType.value === 'elasticsearch-elasticsearch') {
    switch (currentStep.value) {
      case 2: return esSearchGroups.value.length > 0
      case 3: return esSelectedIndices.value.length > 0
      case 4: return !!(props.formData.syncConfig)
      default: return true
    }
  } else {
    switch (currentStep.value) {
      case 2:
        if (taskType.value.startsWith('mysql')) {
          return selectedTables.value.length > 0
        } else {
          return matchedIndices.value.length > 0
        }
      case 3: return !!(props.formData.syncConfig)
      default: return true
    }
  }
})

function handleTaskNameUpdate(newName: string) {
  formData.value.name = newName
  // 如果任务已创建（有 id），立即保存名称修改
  if (formData.value.id) {
    // 这里可以调用 API 更新任务名称，但为了简化，我们在最终提交时一起保存
    // 或者可以添加一个 debounce 的自动保存功能
  }
}

function handleStartConfig() {
  // 前置配置完成，创建任务记录（只保存名称和类型）
  // 调用后端API创建任务，然后关闭对话框
  emit('create', {
    name: props.formData.name,
    sourceType: sourceTypeSelected.value,  // 直接使用选择的值
    targetType: targetTypeSelected.value   // 直接使用选择的值
  })
  // 不进入步骤1，而是关闭对话框
  // 用户需要在任务列表中点击"编辑"按钮才能进入配置步骤
}

watch(() => props.modelValue, (newValue) => {
  if (newValue && props.isEdit) {
    // 编辑模式：从 formData 恢复任务类型和配置数据
    const { sourceType, targetType, mysqlConfig, esConfig } = props.formData
    
    console.log('编辑模式 - 恢复任务数据:', props.formData)
    
    if (sourceType && targetType) {
      taskType.value = `${sourceType}-${targetType}` as SyncTaskType
      sourceTypeSelected.value = sourceType
      targetTypeSelected.value = targetType
      currentStep.value = 1 // 直接进入步骤1
      
      // 恢复 MySQL 配置
      if (mysqlConfig && mysqlConfig.databases) {
        selectedTables.value = []
        mysqlConfig.databases.forEach(db => {
          db.tables.forEach(table => {
            selectedTables.value.push(`${db.database}.${table}`)
          })
        })
        console.log('编辑模式 - 恢复 selectedTables:', selectedTables.value)
      }
      
      // 恢复数据库名称转换配置
      if (props.formData.syncConfig?.dbNameTransform) {
        dbNameTransform.value = props.formData.syncConfig.dbNameTransform
        console.log('编辑模式 - 恢复 dbNameTransform:', dbNameTransform.value)
      }
      
      // 恢复 ES 配置
      if (esConfig) {
        // ES→ES 配置
        if (esConfig.searchGroups) {
          esSearchGroups.value = esConfig.searchGroups
          console.log('编辑模式 - 恢复 esSearchGroups:', esSearchGroups.value)
        }
        if (esConfig.selectedIndices) {
          esSelectedIndices.value = esConfig.selectedIndices
          console.log('编辑模式 - 恢复 esSelectedIndices:', esSelectedIndices.value)
        }
        if (esConfig.indexNameTransform) {
          esIndexNameTransform.value = esConfig.indexNameTransform
          console.log('编辑模式 - 恢复 esIndexNameTransform:', esIndexNameTransform.value)
        }
        
        // ES→MySQL 配置
        if (esConfig.indices && esConfig.indices.length > 0) {
          const firstIndex = esConfig.indices[0]
          indexPattern.value = firstIndex.pattern || ''
          matchedIndices.value = firstIndex.matchedIndices || []
          console.log('编辑模式 - 恢复 indexPattern:', indexPattern.value)
          console.log('编辑模式 - 恢复 matchedIndices:', matchedIndices.value)
        }
      }
    }
  } else if (newValue && !props.isEdit) {
    // 新建模式：重置到前置配置
    taskType.value = ''
    sourceTypeSelected.value = ''
    targetTypeSelected.value = ''
    currentStep.value = 1
    
    // 重置所有状态变量
    selectedTables.value = []
    dbNameTransform.value = {
      enabled: false,
      mode: 'prefix',
      sourcePattern: '',
      targetPattern: ''
    }
    indexPattern.value = ''
    matchedIndices.value = []
    esSearchGroups.value = []
    esSelectedIndices.value = []
    esIndexNameTransform.value = {
      enabled: false,
      mode: 'prefix',
      sourcePattern: '',
      targetPattern: ''
    }
  }
})

const modalStyle = computed(() => ({
  width: '80vw',
  maxWidth: '1400px',
  minWidth: '800px'
}))

const contentStyle = {
  height: '55vh',
  minHeight: '400px',
  overflowY: 'auto' as const
}

function nextStep() {
  if (currentStep.value < maxSteps.value) {
    currentStep.value++
  }
}

function prevStep() {
  if (currentStep.value > 1) {
    currentStep.value--
  }
}

function handleStepClick(newStep: number) {
  // 只有在编辑模式下才允许跳转
  if (!props.isEdit) {
    return
  }
  
  // 验证目标步骤的数据完整性
  // 步骤1：必须有数据源
  if (newStep === 1) {
    currentStep.value = 1
    return
  }
  
  // 步骤2：必须有数据源ID
  if (newStep === 2) {
    if (props.formData.sourceId && props.formData.targetId) {
      currentStep.value = 2
    }
    return
  }
  
  // 步骤3：必须有步骤2的数据
  if (newStep === 3) {
    if (props.formData.sourceId && props.formData.targetId) {
      // 检查步骤2的数据是否存在
      if (taskType.value === 'elasticsearch-elasticsearch') {
        if (esSearchGroups.value.length > 0) {
          currentStep.value = 3
        }
      } else if (taskType.value.startsWith('mysql')) {
        if (selectedTables.value.length > 0) {
          currentStep.value = 3
        }
      } else if (taskType.value === 'elasticsearch-mysql') {
        if (matchedIndices.value.length > 0) {
          currentStep.value = 3
        }
      }
    }
    return
  }
  
  // 步骤4：必须有步骤3的数据
  if (newStep === 4) {
    if (props.formData.sourceId && props.formData.targetId && props.formData.syncConfig) {
      if (taskType.value === 'elasticsearch-elasticsearch') {
        if (esSearchGroups.value.length > 0 && esSelectedIndices.value.length > 0) {
          currentStep.value = 4
        }
      } else if (taskType.value.startsWith('mysql')) {
        if (selectedTables.value.length > 0) {
          currentStep.value = 4
        }
      } else if (taskType.value === 'elasticsearch-mysql') {
        if (matchedIndices.value.length > 0) {
          currentStep.value = 4
        }
      }
    }
    return
  }
  
  // 步骤5（ES→ES）：必须有步骤4的数据
  if (newStep === 5 && taskType.value === 'elasticsearch-elasticsearch') {
    if (props.formData.sourceId && 
        props.formData.targetId && 
        props.formData.syncConfig &&
        esSearchGroups.value.length > 0 &&
        esSelectedIndices.value.length > 0) {
      currentStep.value = 5
    }
    return
  }
}

function handleCancel() {
  visible.value = false
  // 重置状态
  taskType.value = ''
  sourceTypeSelected.value = ''
  targetTypeSelected.value = ''
  currentStep.value = 1
}

function handleSubmit() {
  const submitData: Partial<SyncTask> = {
    ...props.formData,
    sourceType: sourceTypeFromTask.value,
    targetType: targetTypeFromTask.value
  }
  
  console.log('handleSubmit - 开始构建提交数据')
  console.log('handleSubmit - taskType:', taskType.value)
  console.log('handleSubmit - formData:', props.formData)
  
  // 根据任务类型构建配置
  if (taskType.value === 'elasticsearch-elasticsearch') {
    submitData.esConfig = {
      searchGroups: esSearchGroups.value,
      selectedIndices: esSelectedIndices.value,
      indexNameTransform: esIndexNameTransform.value
    }
    console.log('handleSubmit - ES→ES 配置:', submitData.esConfig)
  } else if (taskType.value.startsWith('mysql-')) {
    const databaseMap = new Map<string, string[]>()
    selectedTables.value.forEach(item => {
      const [database, table] = item.split('.')
      if (!databaseMap.has(database)) {
        databaseMap.set(database, [])
      }
      databaseMap.get(database)!.push(table)
    })
    
    submitData.mysqlConfig = {
      databases: Array.from(databaseMap.entries()).map(([database, tables]) => ({
        database,
        tables
      }))
    }
    
    console.log('handleSubmit - MySQL 配置:', submitData.mysqlConfig)
    
    if (taskType.value === 'mysql-mysql' && dbNameTransform.value.enabled) {
      submitData.syncConfig = {
        ...submitData.syncConfig!,
        dbNameTransform: dbNameTransform.value
      }
    }
  } else if (taskType.value === 'elasticsearch-mysql') {
    submitData.esConfig = {
      indices: [{
        pattern: indexPattern.value,
        matchedIndices: matchedIndices.value
      }]
    }
    console.log('handleSubmit - ES→MySQL 配置:', submitData.esConfig)
  }
  
  console.log('handleSubmit - 最终提交数据:', submitData)
  console.log('handleSubmit - syncConfig:', submitData.syncConfig)
  
  emit('submit', submitData)
  handleCancel()
}
</script>
