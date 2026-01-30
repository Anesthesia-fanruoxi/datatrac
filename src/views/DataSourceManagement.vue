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
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <n-icon><AddIcon /></n-icon>
          </template>
          新增数据源
        </n-button>
      </n-space>
    </div>

    <!-- 卡片视图 -->
    <n-grid v-if="viewMode === 'card'" :cols="3" :x-gap="16" :y-gap="16">
      <n-grid-item v-for="item in dataSources" :key="item.id">
        <n-card 
          :title="item.name" 
          hoverable 
          class="datasource-card"
          @click="handleEdit(item)"
        >
          <template #header-extra>
            <n-tag :type="item.type === 'mysql' ? 'info' : 'warning'" size="small">
              {{ item.type.toUpperCase() }}
            </n-tag>
          </template>
          
          <div class="card-content">
            <div class="info-item">
              <n-icon><LocationIcon /></n-icon>
              <span>{{ item.host }}:{{ item.port }}</span>
            </div>
            <div class="info-item" v-if="item.database">
              <n-icon><DatabaseIcon /></n-icon>
              <span>{{ item.database }}</span>
            </div>
            <div class="info-item">
              <n-icon><PersonIcon /></n-icon>
              <span>{{ item.username }}</span>
            </div>
          </div>

          <template #action>
            <div class="card-actions" @click.stop>
              <n-button size="small" quaternary @click="handleTest(item)">测试</n-button>
              <n-button size="small" quaternary @click="handleEdit(item)">编辑</n-button>
              <n-popconfirm @positive-click="handleDelete(item.id)">
                <template #trigger>
                  <n-button size="small" quaternary type="error">删除</n-button>
                </template>
                确定要删除该数据源吗？
              </n-popconfirm>
            </div>
          </template>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- 列表视图 -->
    <n-data-table
      v-else
      :columns="listColumns"
      :data="dataSources"
      :bordered="false"
      :single-line="false"
      @row-click="handleEdit"
      class="datasource-table"
    />

    <!-- 编辑模态框 -->
    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 500px">
      <n-form :model="formModel" :rules="rules" ref="formRef" label-placement="left" label-width="80">
        <n-form-item label="名称" path="name">
          <n-input v-model:value="formModel.name" placeholder="请输入名称" />
        </n-form-item>
        <n-form-item label="类型" path="type">
          <n-select v-model:value="formModel.type" :options="typeOptions" />
        </n-form-item>
        <n-form-item label="地址" path="host">
          <n-input v-model:value="formModel.host" placeholder="localhost" />
        </n-form-item>
        <n-form-item label="端口" path="port">
          <n-input-number v-model:value="formModel.port" :min="1" :max="65535" />
        </n-form-item>
        <n-form-item label="用户名" path="username">
          <n-input v-model:value="formModel.username" />
        </n-form-item>
        <n-form-item label="密码" path="password">
          <n-input v-model:value="formModel.password" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item label="数据库" path="database" v-if="formModel.type === 'mysql'">
          <n-input v-model:value="formModel.database" placeholder="可选" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 12px">
          <n-button @click="showModal = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSave">保存</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, h } from 'vue';
import { 
  NButton, NGrid, NGridItem, NCard, NTag, NText, NPopconfirm, 
  NModal, NForm, NFormItem, NInput, NInputNumber, NSelect, NIcon,
  NSpace, NRadioButton, NRadioGroup, NDataTable
} from 'naive-ui';
import { 
  Add as AddIcon,
  Grid as GridIcon,
  List as ListIcon,
  Location as LocationIcon,
  Server as DatabaseIcon,
  Person as PersonIcon,
  Trash as TrashIcon,
  Create as EditIcon,
  CheckmarkCircle as TestIcon
} from '@vicons/ionicons5';
import { datasourceApi } from '../api/datasource';
import type { DataSource, CreateDataSourceRequest } from '../types';
import { showSuccess, showError, showInfo } from '../utils/message';

const dataSources = ref<DataSource[]>([]);
const viewMode = ref<'card' | 'list'>('card');
const showModal = ref(false);
const modalTitle = ref('新增数据源');
const submitting = ref(false);
const formRef = ref<any>(null);

const formModel = reactive<CreateDataSourceRequest & { id?: string }>({
  name: '',
  type: 'mysql',
  host: 'localhost',
  port: 3306,
  username: 'root',
  password: '',
  database: ''
});

const rules = {
  name: { required: true, message: '请输入名称', trigger: 'blur' },
  host: { required: true, message: '请输入地址', trigger: 'blur' },
  username: { required: true, message: '请输入用户名', trigger: 'blur' }
};

const typeOptions = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'Elasticsearch', value: 'elasticsearch' }
];

const listColumns = [
  { title: '名称', key: 'name' },
  { 
    title: '类型', 
    key: 'type',
    render: (row: DataSource) => h(NTag, { 
      type: row.type === 'mysql' ? 'info' : 'warning',
      size: 'small'
    }, { default: () => row.type.toUpperCase() })
  },
  { title: '主机', key: 'host' },
  { title: '端口', key: 'port' },
  { title: '用户名', key: 'username' },
  { 
    title: '操作', 
    key: 'actions',
    render: (row: DataSource) => h(NSpace, null, {
      default: () => [
        h(NButton, { size: 'small', quaternary: true, onClick: (e) => { e.stopPropagation(); handleTest(row); } }, { icon: () => h(TestIcon) }),
        h(NButton, { size: 'small', quaternary: true, onClick: (e) => { e.stopPropagation(); handleEdit(row); } }, { icon: () => h(EditIcon) }),
        h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error', onClick: (e) => e.stopPropagation() }, { icon: () => h(TrashIcon) }),
          default: () => '确定要删除该数据源吗？'
        })
      ]
    })
  }
];

async function loadData() {
  try {
    dataSources.value = await datasourceApi.list();
  } catch (e: any) {
    showError('获取数据源失败: ' + e);
  }
}

function handleAdd() {
  modalTitle.value = '新增数据源';
  Object.assign(formModel, {
    id: undefined,
    name: '',
    type: 'mysql',
    host: 'localhost',
    port: 3306,
    username: 'root',
    password: '',
    database: ''
  });
  showModal.value = true;
}

function handleEdit(item: DataSource) {
  modalTitle.value = '编辑数据源';
  Object.assign(formModel, { ...item });
  showModal.value = true;
}

async function handleSave() {
  formRef.value?.validate(async (errors: any) => {
    if (!errors) {
      submitting.value = true;
      try {
        if (formModel.id) {
          await datasourceApi.update(formModel.id, formModel as any);
          showSuccess('更新成功');
        } else {
          await datasourceApi.create(formModel);
          showSuccess('创建成功');
        }
        showModal.value = false;
        loadData();
      } catch (e: any) {
        showError('操作失败: ' + e);
      } finally {
        submitting.value = false;
      }
    }
  });
}

async function handleDelete(id: string) {
  try {
    await datasourceApi.delete(id);
    showSuccess('删除成功');
    loadData();
  } catch (e: any) {
    showError('删除失败: ' + e);
  }
}

async function handleTest(item: DataSource) {
  showInfo('正在测试连接...');
  try {
    const result = await datasourceApi.testConnection(item.id);
    if (result.success) {
      showSuccess(`连接成功: ${result.message} (${result.durationMs}ms)`);
    } else {
      showError(`连接失败: ${result.message}`);
    }
  } catch (e: any) {
    showError('连接测试异常: ' + e);
  }
}

onMounted(loadData);
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

.datasource-card {
  cursor: pointer;
  transition: all 0.3s ease;
}

.datasource-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.card-content {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #666;
}

.info-item .n-icon {
  font-size: 18px;
  color: #999;
}

.card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.datasource-table :deep(.n-data-table-tr) {
  cursor: pointer;
}
</style>
