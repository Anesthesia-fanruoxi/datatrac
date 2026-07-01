<template>
  <section class="tm-card detail-panel">
    <header class="tm-card-header gradient soft">
      <div>
        <h2>{{ task?.name || '请选择任务' }}</h2>
        <p>{{ statusText }}</p>
      </div>
    </header>

    <div v-if="task && progress" class="detail-body">
      <TaskProgressSteps :sync-mode="progress.sync_mode" :current-step="progress.current_step" :task-status="task.status" :is-running="isRunning" />

      <div v-if="showOverall" class="overview-grid">
        <div class="metric"><span>总体进度</span><strong>{{ percentage(progress.overall_progress) }}</strong></div>
        <div class="metric"><span>表初始化完成</span><strong>{{ progress.init_tables || 0 }} / {{ progress.total_tables || 0 }}</strong></div>
        <div class="metric"><span>表同步完成</span><strong>{{ progress.completed_tables || 0 }} / {{ progress.total_tables || 0 }}</strong></div>
        <div class="metric"><span>已处理记录</span><strong>{{ progress.processed_records || 0 }} / {{ progress.total_records || 0 }}</strong></div>
        <div class="metric"><span>同步速度</span><strong>{{ progress.sync_speed || 0 }} 条/秒</strong></div>
        <div class="metric"><span>已用时间</span><strong>{{ progress.elapsed_time || '00:00:00' }}</strong></div>
        <div class="metric"><span>预计剩余</span><strong>{{ progress.estimated_time || '00:00:00' }}</strong></div>
      </div>

      <TargetTabs :targets="task.target_conns || []" :model-value="currentTargetId" @update:model-value="$emit('switchTarget', $event)" />

      <div v-if="showIncrementalTable" class="stats-block">
        <table class="stats-table">
          <thead>
            <tr>
              <th></th><th>数据库</th><th>表数量</th><th>全量同步进度</th><th>历史增量</th><th>今日增量</th><th>INSERT</th><th>UPDATE</th><th>DELETE</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="db in databaseStats" :key="db.database">
              <tr>
                <td><button class="expand-btn" @click="$emit('switchDatabase', db.database)">{{ expandedDatabase === db.database ? '⌄' : '›' }}</button></td>
                <td>{{ db.database }}</td>
                <td>{{ db.table_count }}</td>
                <td>{{ percentage(db.full_sync_progress) }}</td>
                <td>{{ db.total_count }}</td>
                <td>{{ db.today_count }}</td>
                <td>{{ db.total_insert_count }}</td>
                <td>{{ db.total_update_count }}</td>
                <td>{{ db.total_delete_count }}</td>
              </tr>
              <tr v-if="expandedDatabase === db.database">
                <td colspan="9">
                  <table class="sub-table">
                    <thead>
                      <tr><th>表名</th><th>全量总数</th><th>全量进度</th><th>历史增量</th><th>今日增量</th><th>INSERT</th><th>UPDATE</th><th>DELETE</th><th>复制延迟</th></tr>
                    </thead>
                    <tbody>
                      <tr v-for="table in filteredTables(db.database)" :key="`${table.database}.${table.table}`">
                        <td>{{ table.table }}</td><td>{{ table.full_sync_total_records || 0 }}</td><td>{{ percentage(table.full_sync_progress) }}</td><td>{{ table.total_count || 0 }}</td><td>{{ table.today_count || 0 }}</td><td>{{ table.insert_count || 0 }}</td><td>{{ table.update_count || 0 }}</td><td>{{ table.delete_count || 0 }}</td><td>{{ table.replication_lag_seconds || 0 }} 秒</td>
                      </tr>
                    </tbody>
                  </table>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </div>
    <div v-else class="empty-state">请选择任务查看详情</div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { IncrementalDatabaseStats, IncrementalTableStats, TaskDetail, TaskProgress } from '../../types/task-monitor'
import TaskProgressSteps from './TaskProgressSteps.vue'
import TargetTabs from './TargetTabs.vue'

const props = defineProps<{
  task: TaskDetail | null
  progress: TaskProgress | null
  currentTargetId?: string | null
  databaseStats: IncrementalDatabaseStats[]
  tableStats: IncrementalTableStats[]
  expandedDatabase?: string | null
}>()

defineEmits<{ switchTarget: [targetId: string]; switchDatabase: [database: string] }>()

const isRunning = computed(() => props.task?.is_running || !!props.progress?.current_step)
const showOverall = computed(() => !(props.progress?.sync_mode === 'incremental' && props.progress?.current_step === 'incremental'))
const showIncrementalTable = computed(() => props.progress?.sync_mode === 'incremental' && props.progress?.current_step === 'incremental')
const statusText = computed(() => (props.task?.is_running ? `运行中 / ${props.task.current_step || '-'}` : props.task?.status || '未运行'))

function percentage(value?: number) {
  return `${(value || 0).toFixed(2)}%`
}

function filteredTables(database: string) {
  return props.tableStats.filter((item) => item.database === database)
}
</script>
