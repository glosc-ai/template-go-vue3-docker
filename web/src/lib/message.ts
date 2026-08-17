// Single entry point for user-facing feedback, backed by element-plus-message
// (the trimmed ElMessage / ElMessageBox / ElNotification bundle).
//
// Views and stores import from here rather than from the library directly, so
// the popup implementation can be swapped without touching feature code.
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus-message'

import 'element-plus-message/dist/index.css'
import 'element-plus-message/theme-chalk/dark/css-vars.css'

const DEFAULT_DURATION = 3000

export function notifySuccess(message: string) {
  ElMessage.success({ message, duration: DEFAULT_DURATION, showClose: true })
}

export function notifyError(message: string) {
  // Errors stay on screen a little longer than confirmations.
  ElMessage.error({ message, duration: 5000, showClose: true })
}

export function notifyWarning(message: string) {
  ElMessage.warning({ message, duration: DEFAULT_DURATION, showClose: true })
}

export function notifyInfo(message: string) {
  ElMessage.info({ message, duration: DEFAULT_DURATION, showClose: true })
}

/**
 * Asks the user to confirm a destructive action.
 * Resolves to true when confirmed and false when dismissed, so callers can
 * `if (!await confirmAction(...)) return` instead of catching a rejection.
 */
export async function confirmAction(
  message: string,
  title = '请确认',
  options: { confirmText?: string, cancelText?: string, danger?: boolean } = {},
): Promise<boolean> {
  try {
    await ElMessageBox.confirm(message, title, {
      confirmButtonText: options.confirmText ?? '确定',
      cancelButtonText: options.cancelText ?? '取消',
      type: options.danger ? 'warning' : 'info',
      confirmButtonClass: options.danger ? 'el-button--danger' : undefined,
    })
    return true
  }
  catch {
    // ElMessageBox rejects on cancel and on close; both mean "do not proceed".
    return false
  }
}

/** Corner notification for events worth more room than a toast. */
export function notify(title: string, message: string, type: 'success' | 'warning' | 'info' | 'error' = 'info') {
  ElNotification({ title, message, type, duration: 4500 })
}
