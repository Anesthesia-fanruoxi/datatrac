<template>
  <n-modal
    v-model:show="visible"
    :title="title"
    preset="card"
    style="width: 500px"
  >
    <n-space vertical :size="16">
      <n-card
        v-for="type in dataSourceTypes"
        :key="type.value"
        :class="['type-card', { selected: modelValue === type.value }]"
        hoverable
        @click="selectType(type.value)"
      >
        <div class="card-content">
          <n-icon :size="48" :color="type.color">
            <component :is="type.icon" />
          </n-icon>
          <div style="margin-left: 16px">
            <h4 style="margin: 0 0 4px">{{ type.label }}</h4>
            <n-text depth="3" style="font-size: 13px">{{ type.description }}</n-text>
          </div>
        </div>
      </n-card>
    </n-space>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NModal, NSpace, NCard, NIcon, NText } from 'naive-ui'
import { Server as ServerIcon, Cloud as CloudIcon } from '@vicons/ionicons5'
import type { DataSourceType } from '../../types'

const props = defineProps<{
  modelValue: boolean
  title: string
  selectedType?: DataSourceType
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'select': [type: DataSourceType]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const dataSourceTypes = [
  {
    value: 'mysql' as DataSourceType,
    label: 'MySQL',
    description: 'MySQL 数据库',
    icon: ServerIcon,
    color: '#18a058'
  },
  {
    value: 'elasticsearch' as DataSourceType,
    label: 'Elasticsearch',
    description: 'Elasticsearch 搜索引擎',
    icon: CloudIcon,
    color: '#2080f0'
  }
]

function selectType(type: DataSourceType) {
  emit('select', type)
  visible.value = false
}
</script>

<style scoped>
.type-card {
  cursor: pointer;
  transition: all 0.3s;
}

.type-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.type-card.selected {
  border: 2px solid #18a058;
}

.card-content {
  display: flex;
  align-items: center;
  padding: 8px;
}
</style>
