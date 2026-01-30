import { createRouter, createWebHistory } from 'vue-router';
import DataSourceManagement from '../views/DataSourceManagement.vue';
import SyncTaskConfig from '../views/SyncTaskConfig.vue';
import TaskMonitor from '../views/TaskMonitor.vue';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/datasource'
    },
    {
      path: '/datasource',
      name: 'DataSourceManagement',
      component: DataSourceManagement,
      meta: { title: '数据源管理' }
    },
    {
      path: '/tasks',
      name: 'SyncTaskConfig',
      component: SyncTaskConfig,
      meta: { title: '同步任务' }
    },
    {
      path: '/monitor',
      name: 'TaskMonitor',
      component: TaskMonitor,
      meta: { title: '任务监控' }
    }
  ]
});

export default router;
