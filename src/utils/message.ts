import { useMessage } from 'naive-ui';

let message: ReturnType<typeof useMessage> | null = null;

export function setupMessage(msg: ReturnType<typeof useMessage>) {
  message = msg;
}

export function showSuccess(text: string) {
  message?.success(text);
}

export function showError(text: string) {
  message?.error(text);
}

export function showWarning(text: string) {
  message?.warning(text);
}

export function showInfo(text: string) {
  message?.info(text);
}
