<template>
  <div class="pipeline-config">
    <n-grid :cols="2" :x-gap="24">
      <n-form-item-gi label="批次大小 (Batch Size)">
        <n-input-number v-model:value="model.batchSize" :min="100" :max="10000" :step="100" style="width: 100%" />
        <template #feedback>单次读取并传输的数据条数</template>
      </n-form-item-gi>
      <n-form-item-gi label="并行线程数 (Concurrency)">
        <n-input-number v-model:value="model.threadCount" :min="1" :max="16" style="width: 100%" />
        <template #feedback>并发执行的任务单元数量</template>
      </n-form-item-gi>
    </n-grid>

    <n-divider title-placement="left">异常处理</n-divider>

    <n-form-item label="错误策略">
      <n-radio-group v-model:value="model.errorStrategy">
        <n-space>
          <n-radio value="skip">跳过错误记录并继续</n-radio>
          <n-radio value="pause">立即暂停任务</n-radio>
        </n-space>
      </n-radio-group>
    </n-form-item>

    <n-form-item label="最大重试次数">
      <n-input-number v-model:value="model.maxRetries" :min="0" :max="5" />
    </n-form-item>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { 
  NFormItem, NFormItemGi, NGrid, NInputNumber, 
  NDivider, NRadioGroup, NRadio, NSpace 
} from 'naive-ui';

const props = defineProps<{
  modelValue: any;
}>();

const emit = defineEmits(['update:modelValue']);

const model = ref(props.modelValue);

watch(model, (newVal) => {
  emit('update:modelValue', newVal);
}, { deep: true });
</script>
