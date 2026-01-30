<template>
  <n-space vertical :size="24">
    <n-form-item label="任务名称" required>
      <n-input
        :value="taskName"
        @update:value="$emit('update:taskName', $event)"
        placeholder="请输入任务名称"
        clearable
      />
    </n-form-item>

    <div class="type-selection-container">
      <div class="type-box">
        <div class="box-label">源数据源类型</div>
        <n-button
          block
          size="large"
          :type="sourceType ? 'primary' : 'default'"
          :disabled="isEdit"
          @click="showSourceSelector = true"
        >
          <template #icon>
            <n-icon v-if="sourceType">
              <component :is="getIcon(sourceType)" />
            </n-icon>
          </template>
          {{ sourceType ? getLabel(sourceType) : '点击选择源类型' }}
        </n-button>
      </div>

      <div class="arrow-box">
        <n-icon size="32" color="#999"><ArrowForward /></n-icon>
      </div>

      <div class="type-box">
        <div class="box-label">目标数据源类型</div>
        <n-button
          block
          size="large"
          :type="targetType ? 'primary' : 'default'"
          :disabled="isEdit"
          @click="showTargetSelector = true"
        >
          <template #icon>
            <n-icon v-if="targetType">
              <component :is="getIcon(targetType)" />
            </n-icon>
          </template>
          {{ targetType ? getLabel(targetType) : '点击选择目标类型' }}
        </n-button>
      </div>
    </div>

    <n-alert v-if="sourceType && targetType" type="success" title="任务类型已确定">
      当前模式：{{ getLabel(sourceType) }} → {{ getLabel(targetType) }}
      <span v-if="isEdit" style="margin-left: 8px; opacity: 0.6">（编辑模式不可修改类型）</span>
    </n-alert>

    <DataSourceTypeSelector
      v-model="showSourceSelector"
      title="选择源类型"
      :selected-type="sourceType"
      @select="$emit('update:sourceType', $event)"
    />

    <DataSourceTypeSelector
      v-model="showTargetSelector"
      title="选择目标类型"
      :selected-type="targetType"
      @select="$emit('update:targetType', $event)"
    />
  </n-space>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { NSpace, NFormItem, NInput, NButton, NIcon, NAlert } from 'naive-ui';
import { Server, Cloud, ArrowForward } from '@vicons/ionicons5';
import DataSourceTypeSelector from './DataSourceTypeSelector.vue';
import type { DataSourceType } from '../../types';

defineProps<{
  taskName: string;
  sourceType?: DataSourceType;
  targetType?: DataSourceType;
  isEdit?: boolean;
}>();

defineEmits(['update:taskName', 'update:sourceType', 'update:targetType']);

const showSourceSelector = ref(false);
const showTargetSelector = ref(false);

function getIcon(type: DataSourceType) {
  return type === 'mysql' ? Server : Cloud;
}

function getLabel(type: DataSourceType) {
  return type === 'mysql' ? 'MySQL' : 'Elasticsearch';
}
</script>

<style scoped>
.type-selection-container {
  display: flex;
  align-items: center;
  gap: 24px;
}
.type-box {
  flex: 1;
}
.box-label {
  margin-bottom: 8px;
  font-size: 13px;
  color: #666;
}
.arrow-box {
  padding-top: 24px;
}
</style>
