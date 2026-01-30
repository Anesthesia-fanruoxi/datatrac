<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="title"
    style="width: 400px"
    @after-leave="$emit('update:modelValue', false)"
  >
    <n-grid :cols="2" :x-gap="12" :y-gap="12">
      <n-grid-item v-for="option in options" :key="option.value">
        <n-card 
          hoverable 
          class="type-card" 
          :class="{ active: selectedType === option.value }"
          @click="handleSelect(option.value)"
        >
          <div class="type-content">
            <n-icon size="32" :color="option.color">
              <component :is="option.icon" />
            </n-icon>
            <span class="type-label">{{ option.label }}</span>
          </div>
        </n-card>
      </n-grid-item>
    </n-grid>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { NModal, NGrid, NGridItem, NCard, NIcon } from 'naive-ui';
import { Server, Cloud } from '@vicons/ionicons5';
import type { DataSourceType } from '../../types';

const props = defineProps<{
  modelValue: boolean;
  title: string;
  selectedType?: DataSourceType;
}>();

const emit = defineEmits(['update:modelValue', 'select']);

const show = ref(props.modelValue);

watch(() => props.modelValue, (val) => {
  show.value = val;
});

const options = [
  { label: 'MySQL', value: 'mysql' as DataSourceType, icon: Server, color: '#2080f0' },
  { label: 'Elasticsearch', value: 'elasticsearch' as DataSourceType, icon: Cloud, color: '#18a058' }
];

function handleSelect(type: DataSourceType) {
  emit('select', type);
  emit('update:modelValue', false);
}
</script>

<style scoped>
.type-card {
  cursor: pointer;
  transition: all 0.3s ease;
  border: 2px solid transparent;
}
.type-card.active {
  border-color: var(--color-primary);
  background-color: rgba(24, 160, 88, 0.05);
}
.type-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
}
.type-label {
  font-weight: bold;
}
</style>
