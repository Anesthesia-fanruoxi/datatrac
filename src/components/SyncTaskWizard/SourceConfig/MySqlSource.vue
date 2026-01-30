<template>
  <div class="source-config">
    <n-form-item label="源数据库" required>
      <n-select
        v-model:value="model.database"
        :options="dbOptions"
        placeholder="请选择数据库"
        @update:value="handleDbChange"
      />
    </n-form-item>

    <n-form-item label="选择同步表" required>
      <n-transfer
        v-model:value="model.selectedTables"
        :options="tableOptions"
        source-filterable
        target-filterable
        title="选择表"
      />
    </n-form-item>

    <n-collapse>
      <n-collapse-item title="高级选项 (库名转换)" name="advanced">
        <n-space vertical>
          <n-checkbox v-model:checked="model.dbNameTransform.enabled">
            启用库名转换
          </n-checkbox>
          <div v-if="model.dbNameTransform.enabled" class="transform-form">
            <n-radio-group v-model:value="model.dbNameTransform.mode">
              <n-radio value="prefix">添加前缀</n-radio>
              <n-radio value="suffix">添加后缀</n-radio>
              <n-radio value="replace">正则替换</n-radio>
            </n-radio-group>
            <div class="transform-inputs">
              <n-input v-model:value="model.dbNameTransform.sourcePattern" placeholder="原模式" />
              <n-input v-model:value="model.dbNameTransform.targetPattern" placeholder="目标模式" />
            </div>
          </div>
        </n-space>
      </n-collapse-item>
    </n-collapse>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { 
  NFormItem, NSelect, NTransfer, NCollapse, NCollapseItem, 
  NSpace, NCheckbox, NRadioGroup, NRadio, NInput 
} from 'naive-ui';
import { taskApi } from '../../../api/task';

const props = defineProps<{
  dataSourceId: string;
  modelValue: any;
}>();

const emit = defineEmits(['update:modelValue']);

const model = ref(props.modelValue);

const dbOptions = ref<{ label: string, value: string }[]>([]);
const tableOptions = ref<{ label: string, value: string }[]>([]);

async function loadDatabases() {
  if (!props.dataSourceId) return;
  try {
    const dbs = await taskApi.getDatabases(props.dataSourceId);
    dbOptions.value = dbs.map(db => ({ label: db, value: db }));
  } catch (e) {
    console.error('加载数据库失败', e);
  }
}

async function handleDbChange(db: string) {
  model.value.selectedTables = [];
  try {
    const tables = await taskApi.getTables(props.dataSourceId, db);
    tableOptions.value = tables.map(t => ({ label: t, value: `${db}.${t}` }));
  } catch (e) {
    console.error('加载表失败', e);
  }
}

watch(model, (newVal) => {
  emit('update:modelValue', newVal);
}, { deep: true });

onMounted(loadDatabases);
</script>

<style scoped>
.transform-form {
  margin-top: 12px;
  padding: 12px;
  background: #f9f9f9;
  border-radius: 4px;
}
.transform-inputs {
  margin-top: 8px;
  display: flex;
  gap: 8px;
}
</style>
