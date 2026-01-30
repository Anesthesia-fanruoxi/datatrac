<template>
  <div class="target-config">
    <n-form-item label="写入策略" required>
      <n-radio-group v-model:value="model.tableExistsStrategy">
        <n-space vertical>
          <n-radio value="truncate">
            清空数据 (Truncate)
            <n-text depth="3" style="margin-left: 8px">保留表结构，清空表中所有数据</n-text>
          </n-radio>
          <n-radio value="drop">
            重建表 (Drop & Create)
            <n-text depth="3" style="margin-left: 8px">删除旧表并根据源结构重新创建</n-text>
          </n-radio>
          <n-radio value="append">
            追加写入 (Append)
            <n-text depth="3" style="margin-left: 8px">在现有数据后追加，不处理冲突</n-text>
          </n-radio>
        </n-space>
      </n-radio-group>
    </n-form-item>

    <n-form-item label="批量插入大小">
      <n-input-number v-model:value="model.bulkInsertSize" :min="100" :max="5000" :step="100" />
      <template #feedback>MySQL 批量写入的 Batch 长度</template>
    </n-form-item>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { NFormItem, NRadioGroup, NRadio, NSpace, NText, NInputNumber } from 'naive-ui';

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
