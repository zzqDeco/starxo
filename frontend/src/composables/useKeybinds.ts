import { onBeforeUnmount, onMounted } from 'vue'

export type KeyCombo = {
  key: string
  meta?: boolean
  ctrl?: boolean
  shift?: boolean
  alt?: boolean
}

export type KeybindHandler = (event: KeyboardEvent) => void

export type Keybind = {
  combo: KeyCombo
  handler: KeybindHandler
  allowInInput?: boolean
}

function isInputTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  if (target.isContentEditable) return true
  const tag = target.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT'
}

function matches(e: KeyboardEvent, combo: KeyCombo): boolean {
  const wantCmd = !!combo.meta || !!combo.ctrl
  const hasCmd = e.metaKey || e.ctrlKey
  if (wantCmd !== hasCmd) return false
  if (!!combo.shift !== e.shiftKey) return false
  if (!!combo.alt !== e.altKey) return false
  return e.key.toLowerCase() === combo.key.toLowerCase()
}

export function useKeybinds(bindings: Keybind[]) {
  function onKeydown(e: KeyboardEvent) {
    for (const b of bindings) {
      if (!matches(e, b.combo)) continue
      if (!b.allowInInput && isInputTarget(e.target)) continue
      e.preventDefault()
      e.stopPropagation()
      b.handler(e)
      return
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', onKeydown, true)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('keydown', onKeydown, true)
  })
}

export function isMac(): boolean {
  if (typeof navigator === 'undefined') return false
  const uaData = (navigator as any).userAgentData
  if (uaData?.platform === 'macOS') return true
  return /Mac|iPhone|iPad/.test(navigator.platform ?? '')
}

export function comboLabel(combo: KeyCombo): string {
  const mod = combo.meta || combo.ctrl ? (isMac() ? '⌘' : 'Ctrl') : ''
  const shift = combo.shift ? (isMac() ? '⇧' : 'Shift') : ''
  const alt = combo.alt ? (isMac() ? '⌥' : 'Alt') : ''
  const key = combo.key.length === 1 ? combo.key.toUpperCase() : combo.key
  return [mod, shift, alt, key].filter(Boolean).join(isMac() ? '' : '+')
}
