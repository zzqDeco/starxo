import { nextTick, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

type NativeTextControl = HTMLInputElement | HTMLTextAreaElement

function isElement(value: unknown): value is Element {
  return typeof Element !== 'undefined' && value instanceof Element
}

function queryTextControl(root: unknown): NativeTextControl | null {
  if (!root) return null
  if (isElement(root)) {
    if (root instanceof HTMLInputElement || root instanceof HTMLTextAreaElement) {
      return root
    }
    return root.querySelector<NativeTextControl>('textarea:not([disabled]), input:not([disabled])')
  }
  return null
}

export function useNativeTextInputSync(value: Ref<string>) {
  const inputRef = ref<any>(null)
  const rootRef = ref<HTMLElement | null>(null)
  let pollTimer: ReturnType<typeof window.setInterval> | null = null

  function nativeInput(): NativeTextControl | null {
    return (
      queryTextControl(rootRef.value) ||
      queryTextControl(inputRef.value?.$el) ||
      queryTextControl(inputRef.value)
    )
  }

  function syncFromDom() {
    const el = nativeInput()
    if (el && el.value !== value.value) {
      value.value = el.value
    }
  }

  function stopPolling() {
    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function startPolling() {
    if (pollTimer || typeof window === 'undefined') return
    pollTimer = window.setInterval(() => {
      syncFromDom()
    }, 80)
  }

  function focusInput() {
    nextTick(() => {
      const el = nativeInput()
      if (!el) return
      el.focus()
      const end = el.value.length
      try {
        el.setSelectionRange(end, end)
      } catch {
        // Some input types do not support selection ranges.
      }
      syncFromDom()
      startPolling()
    })
  }

  function onFocusIn() {
    syncFromDom()
    startPolling()
  }

  function onFocusOut() {
    syncFromDom()
    stopPolling()
  }

  function onInputLike() {
    syncFromDom()
  }

  onMounted(startPolling)
  onBeforeUnmount(stopPolling)

  return {
    inputRef,
    rootRef,
    focusInput,
    syncFromDom,
    onFocusIn,
    onFocusOut,
    onInputLike,
  }
}
