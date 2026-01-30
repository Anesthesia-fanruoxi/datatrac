<template>
  <div class="log-panel-wrapper">
    <n-card size="small" class="log-card">
      <n-tabs 
        v-model:value="currentTab" 
        type="line" 
        class="log-tabs"
        @update:value="handleTabChange"
      >
        <!-- 实时日志 -->
        <n-tab-pane name="logs" class="log-pane">
          <template #tab>
            <div class="tab-label">
              <n-icon :size="16"><ListIcon /></n-icon>
              <span>实时日志</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="allLogRef" class="log-content">
              <div v-for="(log, index) in displayAllLogs" :key="index" :class="['log-line', `log-${log.level}`]">
                <span class="log-time">{{ log.timestamp }}</span>
                <span class="log-level">{{ getLevelText(log.level) }}</span>
                <span class="log-text">{{ log.message }}</span>
              </div>
              <div v-if="displayAllLogs.length === 0" class="log-empty">暂无日志</div>
            </div>
          </div>
        </n-tab-pane>

        <!-- 明细日志 -->
        <n-tab-pane name="detail" class="log-pane">
          <template #tab>
            <div class="tab-label">
              <n-icon :size="16"><DocumentTextIcon /></n-icon>
              <span>明细日志</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="detailLogRef" class="log-content">
              <div v-for="(log, index) in displayDetailLogs" :key="index" :class="['log-line', `log-${log.level}`]">
                <span class="log-time">{{ log.timestamp }}</span>
                <span class="log-level">{{ getLevelText(log.level) }}</span>
                <span class="log-text">{{ log.message }}</span>
              </div>
              <div v-if="displayDetailLogs.length === 0" class="log-empty">暂无明细日志</div>
            </div>
          </div>
        </n-tab-pane>

        <!-- 校验日志 -->
        <n-tab-pane name="verify" class="log-pane">
          <template #tab>
            <div class="tab-label">
              <n-icon :size="16"><CheckmarkCircleIcon /></n-icon>
              <span>校验日志</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="verifyLogRef" class="log-content">
              <div v-for="(log, index) in displayVerifyLogs" :key="index" :class="['log-line', `log-${log.level}`]">
                <span class="log-time">{{ log.timestamp }}</span>
                <span class="log-level">{{ getLevelText(log.level) }}</span>
                <span class="log-text">{{ log.message }}</span>
              </div>
              <div v-if="displayVerifyLogs.length === 0" class="log-empty">暂无校验日志</div>
            </div>
          </div>
        </n-tab-pane>

        <!-- 错误日志 -->
        <n-tab-pane name="errors" class="log-pane">
          <template #tab>
            <div :class="['tab-label', { 'error-pulse': errors.length > 0 }]">
              <n-icon :size="16"><AlertCircleIcon /></n-icon>
              <span>错误日志</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="errorLogRef" class="log-content">
              <div v-for="(error, index) in displayErrors" :key="index" class="error-item">
                <div class="error-item-header">
                  <span class="error-time">{{ new Date(error.timestamp).toLocaleString() }}</span>
                  <n-tag type="error" size="small">{{ error.errorType }}</n-tag>
                </div>
                <div class="error-message">{{ error.message }}</div>
                <div v-if="error.data" class="error-details">
                  <n-collapse>
                    <n-collapse-item title="详细信息">
                      <pre>{{ JSON.stringify(error.data, null, 2) }}</pre>
                    </n-collapse-item>
                  </n-collapse>
                </div>
              </div>
              <div v-if="displayErrors.length === 0" class="log-empty">暂无错误</div>
            </div>
          </div>
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, computed } from 'vue'
import { NCard, NTabs, NTabPane, NTag, NCollapse, NCollapseItem, NIcon } from 'naive-ui'
import {
  List as ListIcon,
  DocumentText as DocumentTextIcon,
  CheckmarkCircle as CheckmarkCircleIcon,
  AlertCircle as AlertCircleIcon
} from '@vicons/ionicons5'

interface Log {
  timestamp: string
  level: string
  message: string
  category?: string
}

interface Error {
  timestamp: number | string
  errorType: string
  message: string
  data?: any
}

interface Props {
  allLogs: Log[]
  detailLogs: Log[]
  verifyLogs: Log[]
  errors: Error[]
}

const props = defineProps<Props>()

const allLogRef = ref<HTMLElement | null>(null)
const detailLogRef = ref<HTMLElement | null>(null)
const verifyLogRef = ref<HTMLElement | null>(null)
const errorLogRef = ref<HTMLElement | null>(null)
const currentTab = ref<string>('logs')

// 限制显示的日志数量，避免渲染过多 DOM 导致卡顿
const MAX_DISPLAY_LOGS = 1000
const MAX_DISPLAY_ERRORS = 100

// 只显示最近的 N 条日志
const displayAllLogs = computed(() => {
  return props.allLogs.slice(-MAX_DISPLAY_LOGS)
})

const displayDetailLogs = computed(() => {
  return props.detailLogs.slice(-MAX_DISPLAY_LOGS)
})

const displayVerifyLogs = computed(() => {
  return props.verifyLogs.slice(-MAX_DISPLAY_LOGS)
})

const displayErrors = computed(() => {
  return props.errors.slice(-MAX_DISPLAY_ERRORS)
})

// 自动滚动到底部
function scrollToBottom(element: HTMLElement | null) {
  if (element) {
    nextTick(() => {
      element.scrollTop = element.scrollHeight
    })
  }
}

// Tab 切换时滚动到底部
function handleTabChange(tabName: string) {
  currentTab.value = tabName
  // 延迟滚动，等待 DOM 更新
  setTimeout(() => {
    switch (tabName) {
      case 'logs':
        scrollToBottom(allLogRef.value)
        break
      case 'detail':
        scrollToBottom(detailLogRef.value)
        break
      case 'verify':
        scrollToBottom(verifyLogRef.value)
        break
      case 'errors':
        scrollToBottom(errorLogRef.value)
        break
    }
  }, 100)
}

// 监听日志数据变化,自动滚动到底部
watch(() => props.allLogs.length, () => {
  if (currentTab.value === 'logs' && allLogRef.value) {
    scrollToBottom(allLogRef.value)
  }
})
watch(() => props.detailLogs.length, () => {
  if (currentTab.value === 'detail' && detailLogRef.value) {
    scrollToBottom(detailLogRef.value)
  }
})
watch(() => props.verifyLogs.length, () => {
  if (currentTab.value === 'verify' && verifyLogRef.value) {
    scrollToBottom(verifyLogRef.value)
  }
})
watch(() => props.errors.length, () => {
  if (currentTab.value === 'errors' && errorLogRef.value) {
    scrollToBottom(errorLogRef.value)
  }
})

// 组件挂载后滚动到底部
onMounted(() => {
  nextTick(() => {
    scrollToBottom(allLogRef.value)
  })
})

function getLevelText(level: string): string {
  const levelMap: Record<string, string> = {
    info: 'INFO',
    warn: 'WARN',
    error: 'ERROR',
  }
  return levelMap[level] || level.toUpperCase()
}

function formatCount(count: number): string {
  if (count >= 1000) {
    return (count / 1000).toFixed(1) + 'K'
  }
  return count.toString()
}
</script>

<style scoped>
.log-panel-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.log-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  border-radius: 12px;
}

.log-card :deep(.n-card__content) {
  height: 100%;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.log-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.log-tabs :deep(.n-tabs-nav) {
  flex-shrink: 0;
  padding-left: 20px;
}

.log-tabs :deep(.n-tabs-content) {
  flex: 1;
  height: 0;
}

.log-tabs :deep(.n-tab-pane) {
  height: 100%;
  padding: 16px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.log-container {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.log-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto !important;
  overflow-x: hidden;
  padding: 12px;
  background-color: #1e1e1e;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  border: 1px solid #e0e0e6;
  border-radius: 4px;
  box-sizing: border-box;
}

.log-content::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

.log-content::-webkit-scrollbar-track {
  background: #2d2d2d;
  border-radius: 5px;
}

.log-content::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 5px;
}

.log-content::-webkit-scrollbar-thumb:hover {
  background: #777;
}

.log-line {
  margin-bottom: 4px;
  white-space: pre-wrap;
  word-break: break-all;
  padding-left: 8px;
  border-left: 3px solid transparent;
  transition: background-color 0.2s;
}

.log-line:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.log-info {
  border-left-color: #4ec9b0;
}

.log-warn {
  border-left-color: #dcdcaa;
}

.log-error {
  border-left-color: #f48771;
}

.log-time {
  color: #858585;
  margin-right: 8px;
}

.log-level {
  display: inline-block;
  width: 50px;
  margin-right: 8px;
  font-weight: bold;
}

.log-text {
  color: #d4d4d4;
}

.log-info .log-level {
  color: #4ec9b0;
}

.log-warn .log-level {
  color: #dcdcaa;
}

.log-error .log-level {
  color: #f48771;
}

.error-item {
  margin-bottom: 12px;
  padding: 12px;
  background-color: #2d2d2d;
  border-radius: 4px;
  border-left: 3px solid #f48771;
}

.error-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.error-time {
  color: #858585;
  font-size: 12px;
}

.error-message {
  color: #f48771;
  margin-bottom: 8px;
}

.error-details pre {
  color: #d4d4d4;
  font-size: 12px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-empty {
  color: #858585;
  text-align: center;
  padding: 20px;
}

.log-content::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

.log-content::-webkit-scrollbar-track {
  background: #2d2d2d;
  border-radius: 5px;
}

.log-content::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 5px;
}

.log-content::-webkit-scrollbar-thumb:hover {
  background: #777;
}
</style>

<!-- 非 scoped 样式，用于 Tab 插槽内容 -->
<style>
.log-tabs .tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 8px;
  transition: all 0.3s;
}

/* 实时日志 tab - 紫色 */
.log-tabs .n-tabs-tab[data-name="logs"] .tab-label {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
}

/* 明细日志 tab - 蓝色 */
.log-tabs .n-tabs-tab[data-name="detail"] .tab-label {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: #fff;
}

/* 校验日志 tab - 绿色 */
.log-tabs .n-tabs-tab[data-name="verify"] .tab-label {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  color: #fff;
}

/* 错误日志 tab - 红粉色 */
.log-tabs .n-tabs-tab[data-name="errors"] .tab-label {
  background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
  color: #fff;
}

/* hover 效果 */
.log-tabs .n-tabs-tab:hover .tab-label {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* 激活状态 */
.log-tabs .n-tabs-tab--active .tab-label {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.log-tabs .error-pulse {
  animation: error-pulse-animation 1.5s ease-in-out infinite;
}

.log-tabs .error-pulse .n-icon {
  color: #fff;
}

@keyframes error-pulse-animation {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}
</style>
