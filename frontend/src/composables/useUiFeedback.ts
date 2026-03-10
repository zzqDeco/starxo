import { useDialog, useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'

function toReadableError(error: unknown): string {
  if (typeof error === 'string') {
    return error.trim() || ''
  }
  if (error && typeof error === 'object' && 'message' in error) {
    const msg = String((error as { message?: unknown }).message || '').trim()
    if (msg) return msg
  }
  return ''
}

function stripTechnicalDetails(message: string): string {
  return message
    .replace(/^error:\s*/i, '')
    .replace(/\s*\(.*stack.*\)$/i, '')
    .replace(/\s+at\s+.+$/i, '')
    .trim()
}

export function useUiFeedback() {
  const message = useMessage()
  const dialog = useDialog()
  const { t } = useI18n()

  function success(content: string) {
    message.success(content)
  }

  function info(content: string) {
    message.info(content)
  }

  function error(actionLabel: string, rawError?: unknown) {
    const detail = stripTechnicalDetails(toReadableError(rawError))
    const content = detail
      ? t('feedback.errorWithReason', { action: actionLabel, reason: detail })
      : t('feedback.errorGeneric', { action: actionLabel })
    message.error(content)
  }

  async function confirmDanger(content: string): Promise<boolean> {
    return await new Promise((resolve) => {
      dialog.warning({
        title: t('feedback.confirmTitle'),
        content,
        positiveText: t('common.confirm'),
        negativeText: t('common.cancel'),
        onPositiveClick: () => resolve(true),
        onNegativeClick: () => resolve(false),
      })
    })
  }

  return {
    success,
    info,
    error,
    confirmDanger,
  }
}
