<template>
  <div class="target-config">
    <n-form-item label="写入模式" required>
      <n-radio-group v-model:value="model.writeMode">
        <n-radio value="index">索引 (Index)</n-radio>
        <n-radio value="create">创建 (Create)</n-radio>
      </n-radio-group>
    </n-form-item>

    <n-form-item label="分片配置 (Shards)">
      <n-grid :cols="2" :x-gap="12">
        <n-form-item-gi label="主分片数">
          <n-input-number v-model:value="model.shards" :min="1" />
        </n-form-item-gi>
        <n-form-item-gi label="副本数">
          <n-input-number v-model:value="model.replicas" :min="0" />
        </n-form-item-gi>
      </n-grid>
    </n-form-item>

    <n-alert title="自动映射" type="warning">
      默认情况下，系统将根据源数据的字段类型自动创建映射。如果需要复杂映射，请在任务创建后通过 API 调整索引模板。
    </n-alert>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { NFormItem, NRadioGroup, NRadio, NGrid, NFormItemGi, NInputNumber, NAlert } from 'naive-ui';

const props = defineProps<{
  dataSourceId: string;
  modelValue: any;
}>();

const emit = defineEmits(['update:modelValue']);

const model = ref(props.modelValue);

watch(model, (newVal) => {
  emit('update:modelValue', newVal);
}, { deep: true });
</script>
