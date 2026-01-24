<template>
  <n-config-provider :theme="theme">
    <n-message-provider>
      <n-notification-provider>
        <Layout />
      </n-notification-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { NConfigProvider, NMessageProvider, NNotificationProvider } from 'naive-ui'
import Layout from './components/Layout.vue'

const theme = ref(null)
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>
