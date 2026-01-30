<template>
  <div style="margin-top: 16px; padding: 16px; border: 1px solid #e0e0e0; border-radius: 4px; background-color: #fafafa;">
    <div style="margin-bottom: 12px; font-weight: 500;">数据库名称转换配置</div>
    
    <n-form label-placement="left" label-width="120">
      <n-form-item label="启用转换">
        <n-switch v-model:value="enabled" />
        <span style="margin-left: 12px; color: #999; font-size: 13px;">
          批量修改数据库名称前缀或后缀
        </span>
      </n-form-item>

      <template v-if="enabled">
        <n-form-item label="转换模式">
          <n-radio-group v-model:value="mode">
            <n-space>
              <n-radio value="prefix">前缀替换</n-radio>
              <n-radio value="suffix">后缀替换</n-radio>
            </n-space>
          </n-radio-group>
        </n-form-item>

        <n-form-item :label="mode === 'prefix' ? '源前缀' : '源后缀'">
          <n-input
            v-model:value="sourcePattern"
            :placeholder="mode === 'prefix' ? '如：a_' : '如：_app'"
          />
        </n-form-item>

        <n-form-item :label="mode === 'prefix' ? '目标前缀' : '目标后缀'">
          <n-input
            v-model:value="targetPattern"
            :placeholder="mode === 'prefix' ? '如：b_' : '如：_prod'"
          />
        </n-form-item>
      </template>
    </n-form>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { NForm, NFormItem, NSwitch, NRadioGroup, NRadio, NSpace, NInput } from 'naive-ui'

const props = defineProps<{
  modelValue?: {
    enabled: boolean
    mode: 'prefix' | 'suffix'
    sourcePattern: string
    targetPattern: string
  }
}>()

const emit = defineEmits<{
  'update:modelValue': [value: {
    enabled: boolean
    mode: 'prefix' | 'suffix'
    sourcePattern: string
    targetPattern: string
  }]
}>()

const enabled = ref(props.modelValue?.enabled || false)
const mode = ref<'prefix' | 'suffix'>(props.modelValue?.mode || 'prefix')
const sourcePattern = ref(props.modelValue?.sourcePattern || '')
const targetPattern = ref(props.modelValue?.targetPattern || '')

// 监听变化，同步到父组件
watch([enabled, mode, sourcePattern, targetPattern], () => {
  emit('update:modelValue', {
    enabled: enabled.value,
    mode: mode.value,
    sourcePattern: sourcePattern.value,
    targetPattern: targetPattern.value
  })
})
</script>
