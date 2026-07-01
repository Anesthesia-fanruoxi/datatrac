<template>
  <div class="layout-shell">
    <aside class="sidebar">
      <div class="brand">DataTrace</div>
      <RouterLink v-for="item in menus" :key="item.path" class="menu-link" :to="item.path">
        {{ item.label }}
      </RouterLink>
    </aside>
    <main class="page-main">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
const menus = [
  { path: '/', label: '概览' },
  { path: '/datasources', label: '数据源配置' },
  { path: '/credentials', label: '凭据管理' },
  { path: '/task-config', label: '任务配置' },
  { path: '/task-monitor', label: '任务监控' }
]
</script>
