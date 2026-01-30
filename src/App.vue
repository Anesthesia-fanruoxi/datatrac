<template>
  <n-config-provider :theme="null">
    <n-message-provider>
      <MessageContainer />
      <n-dialog-provider>
        <Layout />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { defineComponent, h } from 'vue';
import { NConfigProvider, NMessageProvider, NDialogProvider, useMessage } from 'naive-ui';
import Layout from './components/Layout.vue';
import { setupMessage } from './utils/message';

// 消息容器组件，用于获取 message 实例并设置到工具类中
const MessageContainer = defineComponent({
  setup() {
    const message = useMessage();
    setupMessage(message);
    return () => h('div');
  }
});
</script>

<style>
:root {
  --color-primary: #18a058;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: v-sans, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background-color: #f5f7f9;
}
</style>
