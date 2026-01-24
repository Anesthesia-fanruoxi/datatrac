import { createDiscreteApi } from 'naive-ui'

const { message, notification, dialog } = createDiscreteApi(['message', 'notification', 'dialog'])

export { message, notification, dialog }

/**
 * 显示成功消息
 */
export function showSuccess(content: string) {
  message.success(content)
}

/**
 * 显示错误消息
 */
export function showError(content: string) {
  message.error(content)
}

/**
 * 显示警告消息
 */
export function showWarning(content: string) {
  message.warning(content)
}

/**
 * 显示信息消息
 */
export function showInfo(content: string) {
  message.info(content)
}

/**
 * 显示加载消息
 */
export function showLoading(content: string) {
  return message.loading(content, { duration: 0 })
}

/**
 * 显示成功通知
 */
export function notifySuccess(title: string, content?: string) {
  notification.success({
    title,
    content,
    duration: 3000
  })
}

/**
 * 显示错误通知
 */
export function notifyError(title: string, content?: string) {
  notification.error({
    title,
    content,
    duration: 5000
  })
}

/**
 * 显示警告通知
 */
export function notifyWarning(title: string, content?: string) {
  notification.warning({
    title,
    content,
    duration: 4000
  })
}

/**
 * 显示信息通知
 */
export function notifyInfo(title: string, content?: string) {
  notification.info({
    title,
    content,
    duration: 3000
  })
}

/**
 * 显示确认对话框
 */
export function showConfirm(title: string, content: string): Promise<boolean> {
  return new Promise((resolve) => {
    dialog.warning({
      title,
      content,
      positiveText: '确定',
      negativeText: '取消',
      onPositiveClick: () => {
        resolve(true)
      },
      onNegativeClick: () => {
        resolve(false)
      }
    })
  })
}

/**
 * 统一处理 API 错误
 */
export function handleApiError(error: any, defaultMessage = '操作失败') {
  console.error('API Error:', error)
  
  let errorMessage = defaultMessage
  
  if (typeof error === 'string') {
    errorMessage = error
  } else if (error?.message) {
    errorMessage = error.message
  }
  
  showError(errorMessage)
}
