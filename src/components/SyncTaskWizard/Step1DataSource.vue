<template>
  <n-form
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-placement="left"
    label-width="120"
  >
    <n-form-item label="任务名称" path="name">
      <n-input v-model:value="formData.name" placeholder="请输入任务名称" />
    </n-form-item>

    <n-divider />

    <n-form-item label="同步方向" path="syncDirection">
      <div style="display: flex; align-items: center; gap: 16px;">
        <!-- 源选择按钮 -->
        <n-button @click="showSourceTypeModal = true" style="min-width: 120px;">
          <template #icon>
            <n-icon v-if="props.sourceType === 'mysql'" color="#0ea5e9">
              <DatabaseIcon />
            </n-icon>
            <n-icon v-else-if="props.sourceType === 'elasticsearch'" color="#10b981">
              <SearchIcon />
            </n-icon>
          </template>
          {{ props.sourceType === 'mysql' ? 'MySQL' : props.sourceType === 'elasticsearch' ? 'Elasticsearch' : '选择源类型' }}
        </n-button>

        <!-- 箭头 -->
        <n-icon size="24">
          <ArrowForwardIcon />
        </n-icon>

        <!-- 目标选择按钮 -->
        <n-button @click="showTargetTypeModal = true" style="min-width: 120px;">
          <template #icon>
            <n-icon v-if="props.targetType === 'mysql'" color="#0ea5e9">
              <DatabaseIcon />
            </n-icon>
            <n-icon v-else-if="props.targetType === 'elasticsearch'" color="#10b981">
              <SearchIcon />
            </n-icon>
          </template>
          {{ props.targetType === 'mysql' ? 'MySQL' : props.targetType === 'elasticsearch' ? 'Elasticsearch' : '选择目标类型' }}
        </n-button>
      </div>
    </n-form-item>

    <n-divider />

    <n-form-item label="源数据源" path="sourceId">
      <n-select
        v-model:value="formData.sourceId"
        :options="sourceDataSourceOptions"
        placeholder="请先选择源类型"
        :disabled="!props.sourceType"
      />
    </n-form-item>

    <n-form-item label="目标数据源" path="targetId">
      <n-select
        v-model:value="formData.targetId"
        :options="targetDataSourceOptions"
        placeholder="请先选择目标类型"
        :disabled="!props.targetType"
      />
    </n-form-item>
  </n-form>

  <!-- 源类型选择弹框 -->
  <n-modal
    v-model:show="showSourceTypeModal"
    preset="card"
    title="选择源类型"
    style="width: 400px;"
  >
    <n-space justify="center">
      <n-card
        :class="['type-card', { 'type-card-selected': props.sourceType === 'mysql' }]"
        @click="handleSourceTypeSelect('mysql')"
        hoverable
      >
        <div class="type-card-content">
          <n-icon size="48" color="#0ea5e9">
            <DatabaseIcon />
          </n-icon>
          <div class="type-card-title">MySQL</div>
        </div>
      </n-card>
      
      <n-card
        :class="['type-card', { 'type-card-selected': props.sourceType === 'elasticsearch' }]"
        @click="handleSourceTypeSelect('elasticsearch')"
        hoverable
      >
        <div class="type-card-content">
          <n-icon size="48" color="#10b981">
            <SearchIcon />
          </n-icon>
          <div class="type-card-title">Elasticsearch</div>
        </div>
      </n-card>
    </n-space>
  </n-modal>

  <!-- 目标类型选择弹框 -->
  <n-modal
    v-model:show="showTargetTypeModal"
    preset="card"
    title="选择目标类型"
    style="width: 400px;"
  >
    <n-space justify="center">
      <n-card
        :class="['type-card', { 'type-card-selected': props.targetType === 'mysql' }]"
        @click="handleTargetTypeSelect('mysql')"
        hoverable
      >
        <div class="type-card-content">
          <n-icon size="48" color="#0ea5e9">
            <DatabaseIcon />
          </n-icon>
          <div class="type-card-title">MySQL</div>
        </div>
      </n-card>
      
      <n-card
        :class="['type-card', { 'type-card-selected': props.targetType === 'elasticsearch' }]"
        @click="handleTargetTypeSelect('elasticsearch')"
        hoverable
      >
        <div class="type-card-content">
          <n-icon size="48" color="#10b981">
            <SearchIcon />
          </n-icon>
          <div class="type-card-title">Elasticsearch</div>
        </div>
      </n-card>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { NForm, NFormItem, NInput, NSelect, NDivider, NSpace, NCard, NIcon, NButton, NModal, type FormInst, type FormRules } from 'naive-ui'
import { Server as DatabaseIcon, Search as SearchIcon, ArrowForward as ArrowForwardIcon } from '@vicons/ionicons5'
import { useDataSourceStore } from '../../stores/dataSource'
import type { SyncTask } from '../../types'

const props = defineProps<{
  formData: Partial<SyncTask>
  sourceType: 'mysql' | 'elasticsearch' | ''
  targetType: 'mysql' | 'elasticsearch' | ''
}>()

const emit = defineEmits<{
  'update:sourceType': [type: 'mysql' | 'elasticsearch']
  'update:targetType': [type: 'mysql' | 'elasticsearch']
}>()

const dataSourceStore = useDataSourceStore()
const formRef = ref<FormInst | null>(null)
const showSourceTypeModal = ref(false)
const showTargetTypeModal = ref(false)

const rules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  sourceId: [{ required: true, message: '请选择源数据源', trigger: 'change' }],
  targetId: [{ required: true, message: '请选择目标数据源', trigger: 'change' }]
}

// 源数据源选项（根据类型过滤，排除已选为目标的）
const sourceDataSourceOptions = computed(() => {
  if (!props.sourceType) return []
  
  return dataSourceStore.dataSources
    .filter(ds => ds.type === props.sourceType && ds.id !== props.formData.targetId)
    .map(ds => ({
      label: ds.name,
      value: ds.id
    }))
})

// 目标数据源选项（根据类型过滤，排除已选为源的）
const targetDataSourceOptions = computed(() => {
  if (!props.targetType) return []
  
  return dataSourceStore.dataSources
    .filter(ds => ds.type === props.targetType && ds.id !== props.formData.sourceId)
    .map(ds => ({
      label: ds.name,
      value: ds.id
    }))
})

function handleSourceTypeSelect(type: 'mysql' | 'elasticsearch') {
  // 清空已选择的源数据源
  props.formData.sourceId = ''
  emit('update:sourceType', type)
  showSourceTypeModal.value = false
}

function handleTargetTypeSelect(type: 'mysql' | 'elasticsearch') {
  // 清空已选择的目标数据源
  props.formData.targetId = ''
  emit('update:targetType', type)
  showTargetTypeModal.value = false
}

async function validate() {
  return formRef.value?.validate()
}

defineExpose({ validate })
</script>

<style scoped>
.type-card {
  width: 140px;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 2px solid transparent;
}

.type-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.type-card-selected {
  border-color: #18a058;
  background-color: rgba(24, 160, 88, 0.05);
}

.type-card-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 16px 8px;
}

.type-card-title {
  font-size: 16px;
  font-weight: 500;
  color: #333;
}
</style>
