<template>
  <n-layout has-sider position="absolute">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="240"
      :collapsed="collapsed"
      show-trigger
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="logo">
        <n-icon size="32" color="#18a058">
          <SyncIcon />
        </n-icon>
        <span v-if="!collapsed" class="logo-text">DataTrac</span>
      </div>
      <n-menu
        v-model:value="activeKey"
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
      />
    </n-layout-sider>
    <n-layout>
      <n-layout-header bordered class="header">
        <div class="header-content">
          <n-breadcrumb>
            <n-breadcrumb-item>{{ currentTitle }}</n-breadcrumb-item>
          </n-breadcrumb>
        </div>
      </n-layout-header>
      <n-layout-content content-style="padding: 24px;">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, h, computed, type Component } from 'vue';
import { useRoute, RouterLink } from 'vue-router';
import { 
  NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, 
  NMenu, NIcon, NBreadcrumb, NBreadcrumbItem 
} from 'naive-ui';
import { 
  Server as DataSourceIcon, 
  Settings as TaskIcon, 
  StatsChart as MonitorIcon,
  Sync as SyncIcon
} from '@vicons/ionicons5';

const collapsed = ref(false);
const route = useRoute();

const activeKey = computed({
  get: () => route.path,
  set: () => {}
});

const currentTitle = computed(() => {
  return route.meta.title || 'DataTrac';
});

function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) });
}

const menuOptions = [
  {
    label: () => h(RouterLink, { to: '/datasource' }, { default: () => '数据源管理' }),
    key: '/datasource',
    icon: renderIcon(DataSourceIcon)
  },
  {
    label: () => h(RouterLink, { to: '/tasks' }, { default: () => '同步任务' }),
    key: '/tasks',
    icon: renderIcon(TaskIcon)
  },
  {
    label: () => h(RouterLink, { to: '/monitor' }, { default: () => '任务监控' }),
    key: '/monitor',
    icon: renderIcon(MonitorIcon)
  }
];
</script>

<style scoped>
.logo {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 12px;
  overflow: hidden;
  white-space: nowrap;
}

.logo-text {
  font-size: 20px;
  font-weight: bold;
  color: var(--color-primary);
}

.header {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 24px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
