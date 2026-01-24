<template>
  <div class="data-source-management">
    <n-space vertical :size="16">
      <!-- 操作栏 -->
      <n-space justify="space-between">
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <n-icon><AddIcon /></n-icon>
          </template>
          添加数据源
        </n-button>
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
  CheckmarkCircle as TestIcon
} from '@vicons/ionicons5'
import { useDataSourceStore } from '../stores/dataSource'
import { showSuccess, showError, handleApiError } from '../utils/message'
import type { DataSource } from '../types'

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
    width: 150
  },
  {
    title: '类型',
    key: 'type',
    width: 120,
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
    width: 150
  },
  {
    title: '端口',
    key: 'port',
    width: 80
  },
  {
    title: '用户名',
    key: 'username',
    width: 120
  },
  {
    title: '数据库',
    key: 'database',
    width: 120,
    render: (row) => row.database || '-'
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
    width: 200,
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
      await dataSourceStore.addDataSource(formData.value as DataSource)
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
  try {
    const result = await dataSourceStore.testConnection(row.id)
    if (result.success) {
      showSuccess('连接测试成功')
    } else {
      showError(`连接测试失败: ${result.message}`)
    }
  } catch (error) {
    handleApiError(error, '连接测试失败')
  }
}

async function loadDataSources() {
  try {
    await dataSourceStore.fetchDataSources()
  } catch (error) {
    handleApiError(error, '加载数据源列表失败')
  }
}

onMounted(() => {
  loadDataSources()
})
</script>

<style scoped>
.data-source-management {
  height: 100%;
}
</style>
