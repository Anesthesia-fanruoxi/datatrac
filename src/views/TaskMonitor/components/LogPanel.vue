<template>
  <div class="log-panel-wrapper">
    <n-card size="small" class="log-card">
      <n-tabs type="line" animated class="log-tabs">
        <!-- 实时日志 -->
        <n-tab-pane name="logs" class="log-pane">
          <template #tab>
            实时日志
            <n-badge v-if="allLogs.length > 0" :value="allLogs.length" :max="9999" style="margin-left: 8px" />
          </template>
          <div class="log-container">
            <div :ref="el => allLogRef = el" class="log-content">
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
            明细日志
            <n-badge v-if="detailLogs.length > 0" :value="detailLogs.length" :max="9999" style="margin-left: 8px" />
          </template>
          <div class="log-container">
            <div :ref="el => detailLogRef = el" class="log-content">
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
            校验日志
            <n-badge v-if="verifyLogs.length > 0" :value="verifyLogs.length" :max="9999" style="margin-left: 8px" />
          </template>
          <div class="log-container">
            <div :ref="el => verifyLogRef = el" class="log-content">
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
            错误日志
            <n-badge v-if="errors.length > 0" :value="errors.length" :max="99" type="error" style="margin-left: 8px" />
          </template>
          <div class="log-container">
            <div :ref="el => errorLogRef = el" class="log-content">
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
import { NCard, NTabs, NTabPane, NBadge, NTag, NCollapse, NCollapseItem } from 'naive-ui'

interface Log {
  timestamp: string
  level: string
  message: string
  category?: string
}

interface Error {
  timestamp: number
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

interface Emits {
  (e: 'update:allLogRef', ref: HTMLElement | null): void
  (e: 'update:detailLogRef', ref: HTMLElement | null): void
  (e: 'update:verifyLogRef', ref: HTMLElement | null): void
  (e: 'update:errorLogRef', ref: HTMLElement | null): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const allLogRef = ref<HTMLElement | null>(null)
const detailLogRef = ref<HTMLElement | null>(null)
const verifyLogRef = ref<HTMLElement | null>(null)
const errorLogRef = ref<HTMLElement | null>(null)

// 自动滚动到底部
function scrollToBottom(element: HTMLElement | null) {
  if (element) {
    nextTick(() => {
      element.scrollTop = element.scrollHeight
    })
  }
}

// 监听 ref 变化,赋值后立即滚动
watch(allLogRef, (val) => {
  emit('update:allLogRef', val)
  scrollToBottom(val)
})

watch(detailLogRef, (val) => {
  emit('update:detailLogRef', val)
  scrollToBottom(val)
})

watch(verifyLogRef, (val) => {
  emit('update:verifyLogRef', val)
  scrollToBottom(val)
})

watch(errorLogRef, (val) => {
  emit('update:errorLogRef', val)
  scrollToBottom(val)
})

// 监听日志数据变化,自动滚动到底部
watch(() => props.allLogs, () => scrollToBottom(allLogRef.value), { deep: true })
watch(() => props.detailLogs, () => scrollToBottom(detailLogRef.value), { deep: true })
watch(() => props.verifyLogs, () => scrollToBottom(verifyLogRef.value), { deep: true })
watch(() => props.errors, () => scrollToBottom(errorLogRef.value), { deep: true })

// 组件挂载后滚动到底部
onMounted(() => {
  nextTick(() => {
    scrollToBottom(allLogRef.value)
    scrollToBottom(detailLogRef.value)
    scrollToBottom(verifyLogRef.value)
    scrollToBottom(errorLogRef.value)
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
