<template>
  <div v-if="targets.length > 1" class="target-tabs">
    <button
      v-for="target in targets"
      :key="target.id"
      class="target-tab"
      :class="{ active: target.id === modelValue }"
      @click="$emit('update:modelValue', target.id)"
    >
      {{ target.name }}
    </button>
  </div>
</template>

<script setup lang="ts">
import type { TargetConn } from '../../types/task-monitor'

defineProps<{ targets: TargetConn[]; modelValue?: string | null }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
