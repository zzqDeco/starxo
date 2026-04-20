import { onBeforeUnmount, watch, type Ref } from 'vue'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'textarea:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

export function useFocusTrap(
  containerRef: Ref<HTMLElement | null>,
  active: Ref<boolean>,
) {
  let previouslyFocused: HTMLElement | null = null

  function focusables(): HTMLElement[] {
    const el = containerRef.value
    if (!el) return []
    return Array.from(el.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
      .filter(n => n.offsetParent !== null || n === document.activeElement)
  }

  function onKeydown(e: KeyboardEvent) {
    if (!active.value || e.key !== 'Tab') return
    const nodes = focusables()
    if (nodes.length === 0) return
    const first = nodes[0]
    const last = nodes[nodes.length - 1]
    const current = document.activeElement as HTMLElement | null
    if (e.shiftKey) {
      if (current === first || !containerRef.value?.contains(current)) {
        e.preventDefault()
        last.focus()
      }
    } else {
      if (current === last) {
        e.preventDefault()
        first.focus()
      }
    }
  }

  function activate() {
    previouslyFocused = document.activeElement as HTMLElement | null
    document.addEventListener('keydown', onKeydown, true)
    requestAnimationFrame(() => {
      const nodes = focusables()
      if (nodes.length > 0) nodes[0].focus()
    })
  }

  function deactivate() {
    document.removeEventListener('keydown', onKeydown, true)
    if (previouslyFocused && document.contains(previouslyFocused)) {
      previouslyFocused.focus()
    }
    previouslyFocused = null
  }

  watch(active, (v) => {
    if (v) activate()
    else deactivate()
  }, { immediate: true })

  onBeforeUnmount(() => deactivate())
}
