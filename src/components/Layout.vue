<template>
  <n-layout has-sider style="height: 100vh;">
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
      <n-menu
        :collapsed="collapsed"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
        :value="currentRoute"
        @update:value="handleMenuSelect"
      />
    </n-layout-sider>
    
    <n-layout>
      <n-layout-header bordered style="height: 64px; padding: 0 24px; display: flex; align-items: center">
        <h2 style="margin: 0">{{ currentTitle }}</h2>
      </n-layout-header>
      
      <n-layout-content style="height: calc(100vh - 64px);">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NLayout, NLayoutSider, NLayoutHeader, NLayoutContent, NMenu, NIcon } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import {
  Server as DatabaseIcon,
  Sync as SyncIcon,
  BarChart as MonitorIcon
} from '@vicons/ionicons5'

const router = useRouter()
const route = useRoute()

const collapsed = ref(false)

const currentRoute = computed(() => route.name as string)
const currentTitle = computed(() => route.meta.title as string || '数据同步工具')

function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  {
    label: '数据源管理',
    key: 'DataSources',
    icon: renderIcon(DatabaseIcon)
  },
  {
    label: '同步任务配置',
    key: 'Tasks',
    icon: renderIcon(SyncIcon)
  },
  {
    label: '任务执行监控',
    key: 'Monitor',
    icon: renderIcon(MonitorIcon)
  }
]

function handleMenuSelect(key: string) {
  router.push({ name: key })
}
</script>

<style scoped>
</style>
