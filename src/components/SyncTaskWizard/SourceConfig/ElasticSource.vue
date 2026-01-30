<template>
  <div class="source-config">
    <n-form-item label="选择同步索引" required>
      <n-transfer
        v-model:value="model.selectedIndices"
        :options="indexOptions"
        source-filterable
        target-filterable
        title="选择索引"
      />
    </n-form-item>

    <n-alert title="提示" type="info" style="margin-top: 16px">
      Elasticsearch 同步目前支持 Scroll API 模式。建议选择单个或相关联的一组索引进行同步。
    </n-alert>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { NFormItem, NTransfer, NAlert } from 'naive-ui';
import { taskApi } from '../../../api/task';

const props = defineProps<{
  dataSourceId: string;
  modelValue: any;
}>();

const emit = defineEmits(['update:modelValue']);

const model = ref(props.modelValue);
const indexOptions = ref<{ label: string, value: string }[]>([]);

async function loadIndices() {
  if (!props.dataSourceId) return;
  try {
    const indices = await taskApi.getIndices(props.dataSourceId);
    indexOptions.value = indices.map(idx => ({ label: idx, value: idx }));
  } catch (e) {
    console.error('加载索引失败', e);
  }
}

watch(model, (newVal) => {
  emit('update:modelValue', newVal);
}, { deep: true });

onMounted(loadIndices);
</script>
