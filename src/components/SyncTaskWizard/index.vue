<template>
  <n-modal
    v-model:show="visible"
    :title="isEdit ? '编辑任务' : '创建任务'"
    preset="card"
    :style="modalStyle"
    :segmented="{ content: 'soft', footer: 'soft' }"
    :closable="false"
    :mask-closable="false"
  >
    <!-- 步骤条 -->
    <n-steps :current="currentStep" style="margin-bottom: 24px" @update:current="handleStepClick">
      <n-step title="选择数据源" />
      <n-step title="选择数据库和表" />
      <n-step title="同步配置" />
      <n-step title="确认配置" />
    </n-steps>

    <!-- 步骤内容 -->
    <div :style="contentStyle">
      <Step1DataSource
        v-if="currentStep === 1"
        ref="step1Ref"
        :form-data="formData"
        v-model:source-type="sourceType"
        v-model:target-type="targetType"
      />

      <Step2SelectTables
        v-if="currentStep === 2"
        ref="step2Ref"
        :source-id="formData.sourceId!"
        :source-type="sourceType"
        v-model="selectedTables"
        v-model:index-pattern="indexPattern"
        v-model:matched-indices="matchedIndices"
        :db-name-transform="formData.syncConfig?.dbNameTransform"
        @update:db-name-transform="formData.syncConfig!.dbNameTransform = $event"
      />

      <Step3SyncConfig
        v-if="currentStep === 3"
        :form-data="formData"
        :target-type="targetType"
      />

      <Step4Confirm
        v-if="currentStep === 4"
        :preview="configPreview"
        :source-type="sourceType"
        :selected-tables="selectedTables"
      />
    </div>

    <template #footer>
      <n-space justify="space-between">
        <n-button @click="handleCancel">取消</n-button>
        <n-space>
          <n-button v-if="currentStep > 1" @click="prevStep">
            上一步
          </n-button>
          <n-button 
            v-if="currentStep < 4" 
            type="primary" 
            @click="nextStep"
            :disabled="!canGoNext"
          >
            下一步
          </n-button>
          <n-space v-if="currentStep === 4">
            <n-button type="primary" @click="handleSubmit">
              {{ isEdit ? '保存' : '保存任务' }}
            </n-button>
          </n-space>
        </n-space>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { NModal, NSteps, NStep, NSpace, NButton } from 'naive-ui'
import { nextTick, watch } from 'vue'
import Step1DataSource from './Step1DataSource.vue'
import Step2SelectTables from './Step2SelectTables.vue'
import Step3SyncConfig from './Step3SyncConfig.vue'
import Step4Confirm from './Step4Confirm.vue'
import { useDataSourceStore } from '../../stores/dataSource'
import type { SyncTask } from '../../types'

const props = defineProps<{
  modelValue: boolean
  isEdit: boolean
  formData: Partial<SyncTask> & { targetDatabase?: string }
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'submit': [data: Partial<SyncTask>]
}>()

const dataSourceStore = useDataSourceStore()
const step1Ref = ref()
const step2Ref = ref()
const currentStep = ref(1)
const sourceType = ref<'mysql' | 'elasticsearch' | ''>('')
const targetType = ref<'mysql' | 'elasticsearch' | ''>('')
const selectedTables = ref<string[]>([])
const indexPattern = ref('')
const matchedIndices = ref<string[]>([])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

// 监听弹框打开，初始化 sourceType 和 targetType
watch(() => props.modelValue, (newValue) => {
  if (newValue && props.isEdit) {
    // 编辑模式：从 formData 中恢复类型
    sourceType.value = props.formData.sourceType || ''
    targetType.value = props.formData.targetType || ''
    
    // 恢复已选表
    if (props.formData.mysqlConfig?.databases) {
      const tables: string[] = []
      props.formData.mysqlConfig.databases.forEach(db => {
        db.tables.forEach(table => {
          tables.push(`${db.database}.${table}`)
        })
      })
      selectedTables.value = tables
    }
    
    // 恢复索引配置
    if (props.formData.esConfig?.indices?.[0]) {
      indexPattern.value = props.formData.esConfig.indices[0].pattern
      matchedIndices.value = props.formData.esConfig.indices[0].matchedIndices || []
    }
  } else if (newValue && !props.isEdit) {
    // 新建模式：重置所有状态
    currentStep.value = 1
    sourceType.value = ''
    targetType.value = ''
    selectedTables.value = []
    indexPattern.value = ''
    matchedIndices.value = []
  }
})

// 弹框样式 - 占主窗口80%
const modalStyle = computed(() => ({
  width: '80vw',
  maxWidth: '1400px',
  minWidth: '800px'
}))

// 内容区域样式 - 固定高度保持一致
const contentStyle = computed(() => ({
  height: '55vh',
  minHeight: '400px',
  overflowY: 'auto'
}))

const canGoNext = computed(() => {
  switch (currentStep.value) {
    case 1:
      return props.formData.sourceId && props.formData.targetId
    case 2:
      if (sourceType.value === 'mysql') {
        return selectedTables.value.length > 0
      } else if (sourceType.value === 'elasticsearch') {
        return matchedIndices.value.length > 0
      }
      return false
    case 3:
      return props.formData.name && props.formData.syncConfig
    default:
      return true
  }
})

const configPreview = computed(() => {
  const source = dataSourceStore.dataSources.find(ds => ds.id === props.formData.sourceId)
  const target = dataSourceStore.dataSources.find(ds => ds.id === props.formData.targetId)
  
  let databaseInfo = ''
  if (sourceType.value === 'mysql') {
    const tableCount = selectedTables.value.length
    const dbSet = new Set(selectedTables.value.map(t => t.split('.')[0]))
    databaseInfo = `${dbSet.size} 个数据库，${tableCount} 个表`
  } else if (sourceType.value === 'elasticsearch') {
    databaseInfo = `${matchedIndices.value.length} 个索引`
  }
  
  return {
    taskName: props.formData.name || '未命名',
    sourceName: source?.name || '',
    sourceType: source?.type || '',
    targetName: target?.name || '',
    targetType: target?.type || '',
    databaseInfo,
    threadCount: props.formData.syncConfig?.threadCount || 4,
    batchSize: props.formData.syncConfig?.batchSize || 1000,
    errorStrategy: props.formData.syncConfig?.errorStrategy === 'skip' ? '跳过错误' : '遇错暂停'
  }
})

function nextStep() {
  if (currentStep.value < 4) {
    currentStep.value++
    
    // 进入步骤2时，如果是MySQL源，自动加载数据库和表
    if (currentStep.value === 2 && sourceType.value === 'mysql' && props.formData.sourceId) {
      // 使用 nextTick 并添加延迟确保组件已完全渲染
      nextTick(() => {
        setTimeout(() => {
          if (step2Ref.value && typeof step2Ref.value.loadAllDatabaseTables === 'function') {
            step2Ref.value.loadAllDatabaseTables()
          }
        }, 100)
      })
    }
  }
}

function handleStepClick(step: number) {
  // 只允许跳转到已完成的步骤或当前步骤
  // 步骤1：始终可以跳转
  if (step === 1) {
    currentStep.value = step
    return
  }
  
  // 步骤2：需要步骤1完成
  if (step === 2) {
    if (props.formData.sourceId && props.formData.targetId) {
      currentStep.value = step
      
      // 如果是MySQL源，自动加载数据库和表
      if (sourceType.value === 'mysql' && props.formData.sourceId) {
        nextTick(() => {
          setTimeout(() => {
            if (step2Ref.value && typeof step2Ref.value.loadAllDatabaseTables === 'function') {
              step2Ref.value.loadAllDatabaseTables()
            }
          }, 100)
        })
      }
    }
    return
  }
  
  // 步骤3：需要步骤1和步骤2完成
  if (step === 3) {
    const step1Complete = props.formData.sourceId && props.formData.targetId
    const step2Complete = sourceType.value === 'mysql' 
      ? selectedTables.value.length > 0
      : sourceType.value === 'elasticsearch'
        ? matchedIndices.value.length > 0
        : false
    
    if (step1Complete && step2Complete) {
      currentStep.value = step
    }
    return
  }
  
  // 步骤4：需要所有前置步骤完成
  if (step === 4) {
    const step1Complete = props.formData.sourceId && props.formData.targetId
    const step2Complete = sourceType.value === 'mysql' 
      ? selectedTables.value.length > 0
      : sourceType.value === 'elasticsearch'
        ? matchedIndices.value.length > 0
        : false
    const step3Complete = props.formData.name && props.formData.syncConfig
    
    if (step1Complete && step2Complete && step3Complete) {
      currentStep.value = step
    }
    return
  }
}

function prevStep() {
  if (currentStep.value > 1) {
    currentStep.value--
    
    // 返回到步骤2时，如果是MySQL源，自动加载数据库和表
    if (currentStep.value === 2 && sourceType.value === 'mysql' && props.formData.sourceId) {
      nextTick(() => {
        setTimeout(() => {
          if (step2Ref.value && typeof step2Ref.value.loadAllDatabaseTables === 'function') {
            step2Ref.value.loadAllDatabaseTables()
          }
        }, 100)
      })
    }
  }
}

function handleCancel() {
  visible.value = false
  currentStep.value = 1
  sourceType.value = ''
  targetType.value = ''
  selectedTables.value = []
  indexPattern.value = ''
  matchedIndices.value = []
}

function handleSubmit() {
  // 构建提交数据
  const submitData = { ...props.formData }
  
  if (sourceType.value === 'mysql') {
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
  } else if (sourceType.value === 'elasticsearch') {
    submitData.esConfig = {
      indices: [{
        pattern: indexPattern.value,
        matchedIndices: matchedIndices.value
      }]
    }
  }
  
  emit('submit', submitData)
  
  // 重置状态
  currentStep.value = 1
  sourceType.value = ''
  targetType.value = ''
  selectedTables.value = []
  indexPattern.value = ''
  matchedIndices.value = []
}

defineExpose({
  reset: () => {
    currentStep.value = 1
    sourceType.value = ''
    targetType.value = ''
    selectedTables.value = []
    indexPattern.value = ''
    matchedIndices.value = []
  }
})
</script>
