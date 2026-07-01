<template>
  <div class="tm-tabs">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      class="tm-tab"
      :class="{ active: modelValue === tab.key }"
      @click="$emit('update:modelValue', tab.key)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{ modelValue: 'full' | 'incremental' | 'structure' }>()
defineEmits<{ 'update:modelValue': [value: 'full' | 'incremental' | 'structure'] }>()

const tabs = [
  { key: 'full', label: '全量同步' },
  { key: 'incremental', label: '增量同步' },
  { key: 'structure', label: '结构同步' }
] as const
</script>
