<template>
  <div class="log-panel-wrapper">
    <n-card size="small" class="log-card">
      <n-tabs type="line" animated class="log-tabs">
        <!-- 实时日志 -->
        <n-tab-pane name="logs" class="log-pane">
          <template #tab>
            <div class="tab-label">
              <n-icon :size="16"><ListIcon /></n-icon>
              <span>实时日志</span>
              <span v-if="allLogs.length > 0" class="tab-badge">{{ formatCount(allLogs.length) }}</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="allLogRef" class="log-content">
              <div v-for="(log, index) in allLogs" :key="index" :class="['log-line', `log-${log.level}`]">
                <span class="log-time">{{ log.timestamp }}</span>
                <span class="log-level">{{ getLevelText(log.level) }}</span>
                <span class="log-text">{{ log.message }}</span>
              </div>
              <div v-if="allLogs.length === 0" class="log-empty">暂无日志</div>
            </div>
          </div>
        </n-tab-pane>

        <!-- 明细日志 -->
        <n-tab-pane name="detail" class="log-pane">
          <template #tab>
            <div class="tab-label">
              <n-icon :size="16"><DocumentTextIcon /></n-icon>
              <span>明细日志</span>
              <span v-if="detailLogs.length > 0" class="tab-badge detail">{{ formatCount(detailLogs.length) }}</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="detailLogRef" class="log-content">
              <div v-for="(log, index) in detailLogs" :key="index" :class="['log-line', `log-${log.level}`]">
                <span class="log-time">{{ log.timestamp }}</span>
                <span class="log-level">{{ getLevelText(log.level) }}</span>
                <span class="log-text">{{ log.message }}</span>
              </div>
              <div v-if="detailLogs.length === 0" class="log-empty">暂无明细日志</div>
            </div>
          </div>
        </n-tab-pane>

        <!-- 校验日志 -->
        <n-tab-pane name="verify" class="log-pane">
          <template #tab>
            <div class="tab-label">
              <n-icon :size="16"><CheckmarkCircleIcon /></n-icon>
              <span>校验日志</span>
              <span v-if="verifyLogs.length > 0" class="tab-badge verify">{{ formatCount(verifyLogs.length) }}</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="verifyLogRef" class="log-content">
              <div v-for="(log, index) in verifyLogs" :key="index" :class="['log-line', `log-${log.level}`]">
                <span class="log-time">{{ log.timestamp }}</span>
                <span class="log-level">{{ getLevelText(log.level) }}</span>
                <span class="log-text">{{ log.message }}</span>
              </div>
              <div v-if="verifyLogs.length === 0" class="log-empty">暂无校验日志</div>
            </div>
          </div>
        </n-tab-pane>

        <!-- 错误日志 -->
        <n-tab-pane name="errors" class="log-pane">
          <template #tab>
            <div :class="['tab-label', { 'error-pulse': errors.length > 0 }]">
              <n-icon :size="16"><AlertCircleIcon /></n-icon>
              <span>错误日志</span>
              <span v-if="errors.length > 0" class="tab-badge error">{{ errors.length }}</span>
            </div>
          </template>
          <div class="log-container">
            <div ref="errorLogRef" class="log-content">
              <div v-for="(error, index) in errors" :key="index" class="error-item">
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
              <div v-if="errors.length === 0" class="log-empty">暂无错误</div>
            </div>
          </div>
        </n-tab-pane>
      </n-tabs>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue'
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

// 自动滚动到底部
function scrollToBottom(element: HTMLElement | null) {
  if (element && element.parentNode) {
    nextTick(() => {
      if (element.parentNode) {
        element.scrollTop = element.scrollHeight
      }
    })
  }
}

// 监听日志数据变化,自动滚动到底部
watch(() => props.allLogs.length, () => {
  if (allLogRef.value) {
    scrollToBottom(allLogRef.value)
  }
})
watch(() => props.detailLogs.length, () => {
  if (detailLogRef.value) {
    scrollToBottom(detailLogRef.value)
  }
})
watch(() => props.verifyLogs.length, () => {
  if (verifyLogRef.value) {
    scrollToBottom(verifyLogRef.value)
  }
})
watch(() => props.errors.length, () => {
  if (errorLogRef.value) {
    scrollToBottom(errorLogRef.value)
  }
})

// 组件挂载后滚动到底部
onMounted(() => {
  nextTick(() => {
    if (allLogRef.value) scrollToBottom(allLogRef.value)
    if (detailLogRef.value) scrollToBottom(detailLogRef.value)
    if (verifyLogRef.value) scrollToBottom(verifyLogRef.value)
    if (errorLogRef.value) scrollToBottom(errorLogRef.value)
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
}

.log-tabs :deep(.n-tabs-content) {
  flex: 1;
  height: 0;
}

.log-tabs :deep(.n-tab-pane) {
  height: 100%;
  padding: 16px;
  box-sizing: border-box;
}

.log-container {
  height: 100%;
  max-height: 100%;
  overflow: hidden;
}

.log-content {
  height: 100%;
  max-height: 100%;
  overflow-y: scroll !important;
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
}

.log-tabs .tab-badge {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  margin-left: 4px;
}

.log-tabs .tab-badge.detail {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.log-tabs .tab-badge.verify {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
}

.log-tabs .tab-badge.error {
  background: linear-gradient(135deg, #f5576c 0%, #f093fb 100%);
}

.log-tabs .error-pulse {
  animation: error-pulse-animation 1.5s ease-in-out infinite;
}

.log-tabs .error-pulse .n-icon {
  color: #f5576c;
}

@keyframes error-pulse-animation {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}
</style>
