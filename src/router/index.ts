import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/datasources'
  },
  {
    path: '/datasources',
    name: 'DataSources',
    component: () => import('../views/DataSourceManagement.vue'),
    meta: { title: '数据源管理' }
  },
  {
    path: '/tasks',
    name: 'Tasks',
    component: () => import('../views/SyncTaskConfig.vue'),
    meta: { title: '同步任务配置' }
  },
  {
    path: '/monitor',
    name: 'Monitor',
    component: () => import('../views/TaskMonitor.vue'),
    meta: { title: '任务执行监控' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
