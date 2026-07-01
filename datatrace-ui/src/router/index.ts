import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from '../views/AppLayout.vue'
import HomeView from '../views/HomeView.vue'
import DataSourceView from '../views/DataSourceView.vue'
import CredentialView from '../views/CredentialView.vue'
import TaskConfigView from '../views/TaskConfigView.vue'
import TaskMonitorPage from '../views/task-monitor/TaskMonitorPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: '', name: 'home', component: HomeView },
        { path: 'datasources', name: 'datasources', component: DataSourceView },
        { path: 'credentials', name: 'credentials', component: CredentialView },
        { path: 'task-config', name: 'task-config', component: TaskConfigView },
        { path: 'task-monitor/:id?', name: 'task-monitor', component: TaskMonitorPage }
      ]
    }
  ]
})

export default router
