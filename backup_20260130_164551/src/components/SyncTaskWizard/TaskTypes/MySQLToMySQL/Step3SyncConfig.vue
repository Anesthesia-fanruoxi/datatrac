<template>
  <SyncConfigForm
    :name="name"
    :sync-config="syncConfig"
    @update:name="$emit('update:name', $event)"
    @update:sync-config="$emit('update:syncConfig', $event)"
  />
</template>

<script setup lang="ts">
import SyncConfigForm from '../../shared/SyncConfigForm.vue'
import type { SyncConfig } from '../../../../types'

defineProps<{
  name: string
  syncConfig: SyncConfig
}>()

defineEmits<{
  'update:name': [value: string]
  'update:syncConfig': [value: SyncConfig]
}>()
</script>
