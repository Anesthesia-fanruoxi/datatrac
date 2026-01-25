<template>
  <n-modal
    v-model:show="show"
    title="批量测试连接"
    preset="card"
    style="width: 700px"
    :closable="!testing"
    :mask-closable="false"
  >
    <n-space vertical :size="16">
      <!-- 整体进度 -->
      <n-card size="small">
        <n-space vertical :size="8">
          <n-text>总计: {{ results.length }} 个数据源</n-text>
          <n-progress
            type="line"
            :percentage="overallProgress"
            :status="overallStatus"
          />
          <n-space>
            <n-tag type="success">成功: {{ successCount }}</n-tag>
            <n-tag type="error">失败: {{ failedCount }}</n-tag>
          </n-space>
        </n-space>
      </n-card>

      <!-- 使用标签页分开展示测试结果 -->
      <n-tabs v-model:value="currentTab" type="line" animated>
        <n-tab-pane name="step1" tab="步骤1: 测试端口连通性">
          <n-scrollbar style="max-height: 400px">
            <n-space vertical :size="8" style="padding: 8px 0">
              <div v-for="item in allDataSources" :key="item.id">
                <n-space align="center">
                  <!-- 加载中 -->
                  <n-spin v-if="getStep1Status(item.id) === 'loading'" :size="16" />
                  <!-- 成功 -->
                  <n-icon
                    v-else-if="getStep1Status(item.id) === 'success'"
                    :size="16"
                    color="#18a058"
                  >
                    <CheckmarkCircle />
                  </n-icon>
                  <!-- 失败 -->
                  <n-icon
                    v-else-if="getStep1Status(item.id) === 'error'"
                    :size="16"
                    color="#d03050"
                  >
                    <CloseCircle />
                  </n-icon>
                  <!-- 等待 -->
                  <n-icon
                    v-else
                    :size="16"
                    color="#808080"
                  >
                    <TimeOutline />
                  </n-icon>
                  
                  <n-text>数据源 {{ item.name }}:</n-text>
                  <n-text 
                    v-if="getStep1Result(item.id)"
                    :type="getStep1Result(item.id)!.step.success ? 'success' : 'error'"
                  >
                    {{ getStep1Result(item.id)!.step.message }}
                  </n-text>
                  <n-text v-else-if="getStep1Status(item.id) === 'loading'" depth="3">
                    测试中...
                  </n-text>
                  <n-text v-else depth="3">
                    等待测试
                  </n-text>
                  <n-text 
                    v-if="getStep1Result(item.id)?.step.duration" 
                    depth="3" 
                    style="font-size: 12px"
                  >
                    ({{ getStep1Result(item.id)!.step.duration }}ms)
                  </n-text>
                </n-space>
              </div>
            </n-space>
          </n-scrollbar>
        </n-tab-pane>

        <n-tab-pane name="step2" tab="步骤2: 验证账号密码">
          <n-scrollbar style="max-height: 400px">
            <n-space vertical :size="8" style="padding: 8px 0">
              <div v-for="item in allDataSources" :key="item.id">
                <n-space align="center">
                  <!-- 加载中 -->
                  <n-spin v-if="getStep2Status(item.id) === 'loading'" :size="16" />
                  <!-- 成功 -->
                  <n-icon
                    v-else-if="getStep2Status(item.id) === 'success'"
                    :size="16"
                    color="#18a058"
                  >
                    <CheckmarkCircle />
                  </n-icon>
                  <!-- 失败 -->
                  <n-icon
                    v-else-if="getStep2Status(item.id) === 'error'"
                    :size="16"
                    color="#d03050"
                  >
                    <CloseCircle />
                  </n-icon>
                  <!-- 等待 -->
                  <n-icon
                    v-else
                    :size="16"
                    color="#808080"
                  >
                    <TimeOutline />
                  </n-icon>
                  
                  <n-text>数据源 {{ item.name }}:</n-text>
                  <n-text 
                    v-if="getStep2Result(item.id)"
                    :type="getStep2Result(item.id)!.step.success ? 'success' : 'error'"
                  >
                    {{ getStep2Result(item.id)!.step.message }}
                  </n-text>
                  <n-text v-else-if="getStep2Status(item.id) === 'loading'" depth="3">
                    验证中...
                  </n-text>
                  <n-text v-else depth="3">
                    等待验证
                  </n-text>
                  <n-text 
                    v-if="getStep2Result(item.id)?.step.duration" 
                    depth="3" 
                    style="font-size: 12px"
                  >
                    ({{ getStep2Result(item.id)!.step.duration }}ms)
                  </n-text>
                </n-space>
              </div>
            </n-space>
          </n-scrollbar>
        </n-tab-pane>
      </n-tabs>
    </n-space>

    <template #footer>
      <n-space justify="end">
        <n-button :disabled="testing" @click="handleClose">
          关闭
        </n-button>
      </n-space>
    </template>
  </n-modal>

  <!-- 步骤1失败确认对话框 -->
  <n-modal
    v-model:show="showConfirmDialog"
    preset="dialog"
    title="测试失败"
    :content="`步骤1(端口连通性测试)有 ${step1FailedCount} 个数据源失败，是否忽略失败继续测试步骤2(账号密码验证)?`"
    positive-text="忽略失败，继续测试"
    negative-text="停止测试"
    @positive-click="handleContinue"
    @negative-click="handleStop"
  />
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  NModal,
  NCard,
  NSpace,
  NText,
  NProgress,
  NTag,
  NIcon,
  NButton,
  NScrollbar,
  NTabs,
  NTabPane,
  NSpin
} from 'naive-ui'
import {
  CheckmarkCircle,
  CloseCircle,
  TimeOutline
} from '@vicons/ionicons5'
import type { BatchTestDataSourceResult, DataSource } from '../types'

const props = defineProps<{
  modelValue: boolean
  dataSources: DataSource[] // 接收所有数据源列表
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'test': [skipFailed: boolean]
}>()

const show = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const testing = ref(false)
const results = ref<BatchTestDataSourceResult[]>([])
const showConfirmDialog = ref(false)
const step1FailedCount = ref(0)
const currentTab = ref<'step1' | 'step2'>('step1')
const loadingStep1Ids = ref<Set<string>>(new Set()) // 步骤1加载中的数据源ID
const loadingStep2Ids = ref<Set<string>>(new Set()) // 步骤2加载中的数据源ID

// 所有数据源列表
const allDataSources = computed(() => props.dataSources)

// 获取步骤1的结果
function getStep1Result(dataSourceId: string) {
  const result = results.value.find(r => r.dataSourceId === dataSourceId)
  if (result && result.steps.length >= 1) {
    return {
      dataSourceId: result.dataSourceId,
      dataSourceName: result.dataSourceName,
      step: result.steps[0]
    }
  }
  return null
}

// 获取步骤2的结果
function getStep2Result(dataSourceId: string) {
  const result = results.value.find(r => r.dataSourceId === dataSourceId)
  if (result && result.steps.length >= 2) {
    return {
      dataSourceId: result.dataSourceId,
      dataSourceName: result.dataSourceName,
      step: result.steps[1]
    }
  }
  return null
}

// 获取步骤1的状态
function getStep1Status(dataSourceId: string): 'wait' | 'loading' | 'success' | 'error' {
  if (loadingStep1Ids.value.has(dataSourceId)) {
    return 'loading'
  }
  const result = getStep1Result(dataSourceId)
  if (result) {
    return result.step.success ? 'success' : 'error'
  }
  return 'wait'
}

// 获取步骤2的状态
function getStep2Status(dataSourceId: string): 'wait' | 'loading' | 'success' | 'error' {
  if (loadingStep2Ids.value.has(dataSourceId)) {
    return 'loading'
  }
  const result = getStep2Result(dataSourceId)
  if (result) {
    return result.step.success ? 'success' : 'error'
  }
  return 'wait'
}

// 按步骤分组的结果（保留用于进度计算）
const step1Results = computed(() => {
  return results.value
    .filter(r => r.steps.length >= 1)
    .map(r => ({
      dataSourceId: r.dataSourceId,
      dataSourceName: r.dataSourceName,
      step: r.steps[0]
    }))
})

const step2Results = computed(() => {
  return results.value
    .filter(r => r.steps.length >= 2)
    .map(r => ({
      dataSourceId: r.dataSourceId,
      dataSourceName: r.dataSourceName,
      step: r.steps[1]
    }))
})

const successCount = computed(() => 
  results.value.filter(r => r.success).length
)

const failedCount = computed(() => 
  results.value.filter(r => r.steps.length > 0 && !r.success).length
)

const overallProgress = computed(() => {
  if (results.value.length === 0) return 0
  const completed = results.value.filter(r => r.steps.length === 2).length
  const progress = (completed / results.value.length) * 100
  return isNaN(progress) ? 0 : Math.round(progress)
})

const overallStatus = computed<'default' | 'success' | 'error' | 'warning'>(() => {
  if (testing.value) return 'default'
  if (failedCount.value > 0) return 'error'
  if (successCount.value > 0) return 'success'
  return 'default'
})

function getResultColor(result: BatchTestDataSourceResult) {
  if (result.success) return '#18a058'
  if (result.steps.length > 0 && !result.success) return '#d03050'
  return '#808080'
}

function handleClose() {
  show.value = false
}

function handleContinue() {
  showConfirmDialog.value = false
  
  // 切换到步骤2标签页
  currentTab.value = 'step2'
  
  // 标记所有步骤1成功的数据源步骤2为加载中
  results.value.forEach(r => {
    if (r.steps[0].success) {
      loadingStep2Ids.value.add(r.dataSourceId)
    }
  })
  
  // 触发步骤2测试
  emit('test', true)
}

function handleStop() {
  showConfirmDialog.value = false
  testing.value = false
}

function startTest() {
  testing.value = true
  results.value = []
  currentTab.value = 'step1' // 重置到步骤1标签页
  loadingStep1Ids.value.clear()
  loadingStep2Ids.value.clear()
  
  // 标记所有数据源步骤1为加载中
  props.dataSources.forEach(ds => {
    loadingStep1Ids.value.add(ds.id)
  })
  
  emit('test', false)
}

function updateResult(result: BatchTestDataSourceResult) {
  const index = results.value.findIndex(r => r.dataSourceId === result.dataSourceId)
  
  // 如果是步骤1的结果
  if (result.steps.length === 1) {
    // 延迟1秒后更新结果（显示加载动画）
    setTimeout(() => {
      loadingStep1Ids.value.delete(result.dataSourceId)
      
      if (index >= 0) {
        results.value[index] = result
      } else {
        results.value.push(result)
      }
      
      // 检查是否所有数据源都完成了步骤1
      if (testing.value) {
        const allStep1Complete = results.value.every(r => r.steps.length >= 1) && 
                                 results.value.length === props.dataSources.length
        if (allStep1Complete) {
          // 检查是否有失败的
          const step1Failed = results.value.filter(r => !r.steps[0].success).length
          
          if (step1Failed > 0) {
            // 有失败的，显示确认对话框
            step1FailedCount.value = step1Failed
            showConfirmDialog.value = true
          } else {
            // 全部成功，自动切换到步骤2标签页
            setTimeout(() => {
              currentTab.value = 'step2'
              // 标记所有数据源步骤2为加载中
              results.value.forEach(r => {
                loadingStep2Ids.value.add(r.dataSourceId)
              })
            }, 500)
          }
        }
      }
    }, 1000)
  } 
  // 如果是步骤2的结果
  else if (result.steps.length === 2) {
    // 延迟1秒后更新结果（显示加载动画）
    setTimeout(() => {
      loadingStep2Ids.value.delete(result.dataSourceId)
      
      if (index >= 0) {
        results.value[index] = result
      } else {
        results.value.push(result)
      }
    }, 1000)
  }
}

function checkStep1Failures() {
  const failures = results.value.filter(r => 
    r.steps.length === 1 && !r.steps[0].success
  ).length
  
  if (failures > 0) {
    step1FailedCount.value = failures
    showConfirmDialog.value = true
  }
}

function finishTest() {
  testing.value = false
}

defineExpose({
  startTest,
  updateResult,
  checkStep1Failures,
  finishTest
})
</script>
