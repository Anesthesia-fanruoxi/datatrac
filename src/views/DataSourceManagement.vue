<template>
  <div class="data-source-management">
    <n-space vertical :size="16">
      <!-- 操作栏 -->
      <n-space justify="space-between">
        <n-space>
          <n-button type="primary" @click="handleAdd">
            <template #icon>
              <n-icon><AddIcon /></n-icon>
            </template>
            添加数据源
          </n-button>
          <n-button type="info" @click="handleBatchTest" :disabled="dataSourceStore.dataSources.length === 0">
            <template #icon>
              <n-icon><TestIcon /></n-icon>
            </template>
            批量测试
          </n-button>
        </n-space>
        <n-button @click="loadDataSources">
          <template #icon>
            <n-icon><RefreshIcon /></n-icon>
          </template>
          刷新
        </n-button>
      </n-space>

      <!-- 数据源列表 -->
      <n-data-table
        :columns="columns"
        :data="dataSourceStore.dataSources"
        :loading="dataSourceStore.loading"
        :pagination="pagination"
        :bordered="false"
        :scroll-x="1400"
        :max-height="600"
      />
    </n-space>

    <!-- 添加/编辑数据源对话框 -->
    <n-modal
      v-model:show="showModal"
      :title="isEdit ? '编辑数据源' : '添加数据源'"
      preset="dialog"
      :positive-text="isEdit ? '保存' : '添加'"
      negative-text="取消"
      @positive-click="handleSubmit"
      style="width: 600px"
    >
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-placement="left"
        label-width="100"
        require-mark-placement="right-hanging"
      >
        <n-form-item label="名称" path="name">
          <n-input v-model:value="formData.name" placeholder="请输入数据源名称" />
        </n-form-item>

        <n-form-item label="类型" path="type">
          <n-select
            v-model:value="formData.type"
            :options="typeOptions"
            placeholder="请选择数据源类型"
          />
        </n-form-item>

        <n-form-item label="主机" path="host">
          <n-input v-model:value="formData.host" placeholder="请输入主机地址" />
        </n-form-item>

        <n-form-item label="端口" path="port">
          <n-input-number
            v-model:value="formData.port"
            :min="1"
            :max="65535"
            placeholder="请输入端口号"
            style="width: 100%"
          />
        </n-form-item>

        <n-form-item label="用户名" path="username">
          <n-input v-model:value="formData.username" placeholder="请输入用户名" />
        </n-form-item>

        <n-form-item label="密码" path="password">
          <n-input
            v-model:value="formData.password"
            :type="showPassword ? 'text' : 'password'"
            placeholder="请输入密码"
          >
            <template #suffix>
              <n-icon
                :component="showPassword ? EyeOffIcon : EyeIcon"
                @click="showPassword = !showPassword"
                style="cursor: pointer"
              />
            </template>
          </n-input>
        </n-form-item>

        <n-form-item
          v-if="formData.type === 'mysql'"
          label="数据库名"
          path="database"
        >
          <n-input
            v-model:value="formData.database"
            placeholder="请输入数据库名（可选）"
          />
        </n-form-item>

        <!-- 测试连接按钮 -->
        <n-form-item v-if="isEdit" label=" ">
          <n-button
            type="info"
            :loading="testingConnection"
            @click="handleTestConnection"
          >
            <template #icon>
              <n-icon><TestIcon /></n-icon>
            </template>
            测试连接
          </n-button>
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 删除确认对话框 -->
    <n-modal
      v-model:show="showDeleteModal"
      preset="dialog"
      title="确认删除"
      content="确定要删除这个数据源吗？此操作不可恢复。"
      positive-text="删除"
      negative-text="取消"
      @positive-click="confirmDelete"
    />

    <!-- 测试连接进度弹框 -->
    <n-modal
      v-model:show="showTestModal"
      title="测试连接"
      preset="card"
      style="width: 500px"
      :closable="!testingConnection"
      :mask-closable="false"
    >
      <n-space vertical :size="16">
        <n-steps :current="currentTestStep" :status="getTestStatus()">
          <n-step
            v-for="step in testSteps"
            :key="step.step"
            :title="step.name"
            :description="step.message || '等待执行'"
          >
            <template #icon>
              <n-icon v-if="step.status === 'finish'" color="#18a058">
                <TestIcon />
              </n-icon>
              <n-icon v-else-if="step.status === 'error'" color="#d03050">
                <CloseCircle />
              </n-icon>
              <n-spin v-else-if="step.status === 'process'" :size="20" />
            </template>
          </n-step>
        </n-steps>

        <!-- 步骤详情 -->
        <n-card v-if="testSteps.length > 0" size="small">
          <n-space vertical :size="8">
            <div v-for="step in testSteps" :key="step.step">
              <n-text v-if="step.status !== 'wait'">
                <n-icon
                  :color="step.status === 'finish' ? '#18a058' : step.status === 'error' ? '#d03050' : '#2080f0'"
                  style="vertical-align: middle; margin-right: 8px"
                >
                  <TestIcon v-if="step.status === 'finish'" />
                  <CloseCircle v-else-if="step.status === 'error'" />
                  <TimeOutline v-else />
                </n-icon>
                <n-text strong>步骤{{ step.step }}: {{ step.name }}</n-text>
                <n-text v-if="step.duration" depth="3" style="margin-left: 8px">
                  ({{ step.duration }}ms)
                </n-text>
                <br />
                <n-text depth="2" style="margin-left: 32px">
                  {{ step.message }}
                </n-text>
              </n-text>
            </div>
          </n-space>
        </n-card>
      </n-space>

      <template #footer>
        <n-space justify="end">
          <n-button :disabled="testingConnection" @click="showTestModal = false">
            关闭
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 批量测试对话框 -->
    <BatchTestDialog
      ref="batchTestDialogRef"
      v-model="showBatchTestDialog"
      :data-sources="dataSourceStore.dataSources"
      @test="handleBatchTestExecute"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
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
  NIcon,
  NTag,
  NSteps,
  NStep,
  NSpin,
  NCard,
  NText,
  type DataTableColumns,
  type FormInst,
  type FormRules
} from 'naive-ui'
import {
  Add as AddIcon,
  Refresh as RefreshIcon,
  Eye as EyeIcon,
  EyeOff as EyeOffIcon,
  Create as EditIcon,
  Trash as DeleteIcon,
  CheckmarkCircle as TestIcon,
  CloseCircle,
  TimeOutline
} from '@vicons/ionicons5'
import { useDataSourceStore } from '../stores'
import { showSuccess, showError, handleApiError, notifySuccess, notifyError } from '../utils/message'
import type { DataSource, BatchTestDataSourceResult } from '../types'
import BatchTestDialog from '../components/BatchTestDialog.vue'

const dataSourceStore = useDataSourceStore()

// 表格配置
const pagination = {
  pageSize: 10
}

// 表格列定义
const columns: DataTableColumns<DataSource> = [
  {
    title: '名称',
    key: 'name',
    minWidth: 120,
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: '类型',
    key: 'type',
    width: 100,
    render: (row) => {
      return h(
        NTag,
        {
          type: row.type === 'mysql' ? 'info' : 'success'
        },
        { default: () => row.type === 'mysql' ? 'MySQL' : 'Elasticsearch' }
      )
    }
  },
  {
    title: '主机',
    key: 'host',
    minWidth: 140,
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: '端口',
    key: 'port',
    width: 80
  },
  {
    title: '用户名',
    key: 'username',
    width: 100,
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: '数据库',
    key: 'database',
    width: 100,
    ellipsis: {
      tooltip: true
    },
    render: (row) => row.database || '-'
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 160,
    render: (row) => {
      // 毫秒时间戳转换为本地时间 (UTC+8)
      const date = new Date(row.createdAt)
      return date.toLocaleString('zh-CN', { 
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      })
    }
  },
  {
    title: '更新时间',
    key: 'updatedAt',
    width: 160,
    render: (row) => {
      // 毫秒时间戳转换为本地时间 (UTC+8)
      const date = new Date(row.updatedAt)
      return date.toLocaleString('zh-CN', { 
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      })
    }
  },
  {
    title: '操作',
    key: 'actions',
    width: 320,
    fixed: 'right',
    render: (row) => {
      return h(
        NSpace,
        { size: 8, wrap: false },
        {
          default: () => [
            h(
              NButton,
              {
                size: 'small',
                onClick: () => handleTest(row)
              },
              { default: () => '测试连接', icon: () => h(NIcon, null, { default: () => h(TestIcon) }) }
            ),
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
const showPassword = ref(false)
const isEdit = ref(false)
const testingConnection = ref(false)
const formRef = ref<FormInst | null>(null)
const formData = ref<Partial<DataSource>>({
  name: '',
  type: 'mysql',
  host: '',
  port: 3306,
  username: '',
  password: '',
  database: ''
})

// 测试连接进度弹框
const showTestModal = ref(false)
const testSteps = ref<Array<{
  step: number
  name: string
  status: 'wait' | 'process' | 'finish' | 'error'
  message?: string
  duration?: number
}>>([])
const currentTestStep = ref(0)

const typeOptions = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'Elasticsearch', value: 'elasticsearch' }
]

const rules: FormRules = {
  name: [
    { required: true, message: '请输入数据源名称', trigger: 'blur' }
  ],
  type: [
    { required: true, message: '请选择数据源类型', trigger: 'change' }
  ],
  host: [
    { required: true, message: '请输入主机地址', trigger: 'blur' }
  ],
  port: [
    { required: true, type: 'number', message: '请输入端口号', trigger: 'blur' }
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ]
}

// 删除相关
const showDeleteModal = ref(false)
const deleteTarget = ref<DataSource | null>(null)

// 批量测试相关
const showBatchTestDialog = ref(false)
const batchTestDialogRef = ref<InstanceType<typeof BatchTestDialog> | null>(null)

// 方法
function handleAdd() {
  isEdit.value = false
  formData.value = {
    name: '',
    type: 'mysql',
    host: '',
    port: 3306,
    username: '',
    password: '',
    database: ''
  }
  showPassword.value = false
  showModal.value = true
}

function handleEdit(row: DataSource) {
  isEdit.value = true
  formData.value = { ...row }
  showPassword.value = false
  showModal.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
    
    if (isEdit.value && formData.value.id) {
      await dataSourceStore.updateDataSource(formData.value.id, formData.value as DataSource)
      showSuccess('数据源更新成功')
    } else {
      // 创建新数据源时，直接传递 formData（不包含 id、createdAt、updatedAt）
      await dataSourceStore.addDataSource(formData.value as any)
      showSuccess('数据源添加成功')
    }
    
    showModal.value = false
    await loadDataSources()
  } catch (error) {
    handleApiError(error, isEdit.value ? '更新数据源失败' : '添加数据源失败')
    return false
  }
}

function handleDelete(row: DataSource) {
  deleteTarget.value = row
  showDeleteModal.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  
  try {
    await dataSourceStore.deleteDataSource(deleteTarget.value.id)
    showSuccess('数据源删除成功')
    await loadDataSources()
  } catch (error) {
    handleApiError(error, '删除数据源失败')
  } finally {
    deleteTarget.value = null
  }
}

async function handleTest(row: DataSource) {
  // 初始化测试步骤
  testSteps.value = [
    { step: 1, name: '测试端口连通性', status: 'wait' },
    { step: 2, name: '验证账号密码', status: 'wait' }
  ]
  currentTestStep.value = 0
  showTestModal.value = true
  testingConnection.value = true
  
  // 监听测试步骤事件
  const { listen } = await import('@tauri-apps/api/event')
  const unlisten = await listen('connection-test-step', (event: any) => {
    const step = event.payload
    const index = step.step - 1
    
    if (index >= 0 && index < testSteps.value.length) {
      testSteps.value[index].status = step.success ? 'finish' : 'error'
      testSteps.value[index].message = step.message
      testSteps.value[index].duration = step.duration
      currentTestStep.value = step.step
    }
  })
  
  try {
    const result = await dataSourceStore.testConnection(row.id)
    
    // 显示最终结果通知
    if (result.success) {
      notifySuccess('连接测试成功', '所有测试步骤均通过')
    } else {
      notifyError('连接测试失败', result.message)
    }
  } catch (error) {
    // 标记当前步骤为失败
    if (currentTestStep.value > 0 && currentTestStep.value <= testSteps.value.length) {
      testSteps.value[currentTestStep.value - 1].status = 'error'
      testSteps.value[currentTestStep.value - 1].message = '测试失败'
    }
    handleApiError(error, '连接测试失败')
  } finally {
    testingConnection.value = false
    unlisten()
  }
}

async function handleTestConnection() {
  // 在表单中测试连接（仅在编辑模式下可用）
  if (!isEdit.value || !formData.value.id) {
    showError('只能在编辑模式下测试连接')
    return
  }
  
  // 初始化测试步骤
  testSteps.value = [
    { step: 1, name: '测试端口连通性', status: 'wait' },
    { step: 2, name: '验证账号密码', status: 'wait' }
  ]
  currentTestStep.value = 0
  showTestModal.value = true
  testingConnection.value = true
  
  // 监听测试步骤事件
  const { listen } = await import('@tauri-apps/api/event')
  const unlisten = await listen('connection-test-step', (event: any) => {
    const step = event.payload
    const index = step.step - 1
    
    if (index >= 0 && index < testSteps.value.length) {
      testSteps.value[index].status = step.success ? 'finish' : 'error'
      testSteps.value[index].message = step.message
      testSteps.value[index].duration = step.duration
      currentTestStep.value = step.step
    }
  })
  
  try {
    const result = await dataSourceStore.testConnection(formData.value.id)
    
    // 显示最终结果通知
    if (result.success) {
      notifySuccess('连接测试成功', '所有测试步骤均通过')
    } else {
      notifyError('连接测试失败', result.message)
    }
  } catch (error) {
    // 标记当前步骤为失败
    if (currentTestStep.value > 0 && currentTestStep.value <= testSteps.value.length) {
      testSteps.value[currentTestStep.value - 1].status = 'error'
      testSteps.value[currentTestStep.value - 1].message = '测试失败'
    }
    handleApiError(error, '连接测试失败')
  } finally {
    testingConnection.value = false
    unlisten()
  }
}

function getTestStatus() {
  // 根据测试步骤状态返回整体状态
  const hasError = testSteps.value.some(step => step.status === 'error')
  const allFinished = testSteps.value.every(step => step.status === 'finish')
  
  if (hasError) {
    return 'error'
  } else if (allFinished) {
    return 'finish'
  } else {
    return 'process'
  }
}

async function loadDataSources() {
  try {
    await dataSourceStore.fetchDataSources()
  } catch (error) {
    handleApiError(error, '加载数据源列表失败')
  }
}

async function handleBatchTest() {
  showBatchTestDialog.value = true
  // 等待对话框打开
  await new Promise(resolve => setTimeout(resolve, 100))
  batchTestDialogRef.value?.startTest()
}

async function handleBatchTestExecute(skipFailedStep1: boolean) {
  try {
    // 监听批量测试步骤事件
    const { listen } = await import('@tauri-apps/api/event')
    const unlisten = await listen('batch-test-step', (event: any) => {
      const result: BatchTestDataSourceResult = event.payload
      batchTestDialogRef.value?.updateResult(result)
    })
    
    // 执行批量测试
    const result = await dataSourceStore.batchTestConnections(skipFailedStep1)
    
    // 如果是第一次测试(skipFailedStep1=false)且步骤1有失败
    if (!skipFailedStep1) {
      // 检查是否所有数据源都完成了2个步骤
      const allCompleted = result.results.every(r => r.steps.length === 2)
      
      if (!allCompleted) {
        // 有数据源只完成了步骤1,检查是否有失败
        const step1Only = result.results.filter(r => r.steps.length === 1)
        const step1Failed = step1Only.filter(r => !r.steps[0].success).length
        
        if (step1Failed > 0) {
          // 显示确认对话框
          batchTestDialogRef.value?.checkStep1Failures()
          unlisten()
          return
        }
      }
    }
    
    // 测试完成
    batchTestDialogRef.value?.finishTest()
    unlisten()
  } catch (error) {
    console.error('批量测试错误:', error)
    handleApiError(error, '批量测试失败')
    batchTestDialogRef.value?.finishTest()
  }
}

onMounted(() => {
  loadDataSources()
})
</script>

<style scoped>
.data-source-management {
  width: 100%;
  height: 100%;
  padding: 24px;
  box-sizing: border-box;
}
</style>
