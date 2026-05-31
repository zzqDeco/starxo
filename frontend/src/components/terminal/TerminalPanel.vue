<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { NButton, NIcon, NInput, NTooltip } from 'naive-ui'
import { TrashOutline, Cube, PaperPlaneOutline } from '@vicons/ionicons5'
import SxTerminalRow from '@/components/ui/SxTerminalRow.vue'
import { useWailsEvent } from '@/composables/useWailsEvent'
import { useConnectionStore } from '@/stores/connectionStore'
import { useContainerStore } from '@/stores/containerStore'
import { useI18n } from 'vue-i18n'
import { RunTerminalCommand } from '../../../wailsjs/go/service/SandboxService'

const { t } = useI18n()
const connectionStore = useConnectionStore()
const containerStore = useContainerStore()

const terminalEl = ref<HTMLElement | null>(null)
const lines = ref<Array<{ text: string; type: 'stdout' | 'stderr' | 'info' }>>([])
const autoScroll = ref(true)
const lineCount = ref(0)
const commandInput = ref('')
const commandRunning = ref(false)

let termInstance: any = null
let fitAddon: any = null
let terminalSchemeQuery: MediaQueryList | null = null
let terminalSchemeListener: ((event: MediaQueryListEvent) => void) | null = null
const xtermLoaded = ref(false)

const sshConnected = computed(() => connectionStore.sshConnected)
const activeContainer = computed(() => containerStore.activeContainerID || '')
const canRunCommand = computed(() => sshConnected.value && !!activeContainer.value && !commandRunning.value)
const commandPlaceholder = computed(() => {
  if (!sshConnected.value) return t('terminal.connectFirst')
  if (!activeContainer.value) return t('terminal.activateSandboxFirst')
  return t('terminal.commandPlaceholder')
})

function formatTime(): string {
  const now = new Date()
  return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`
}

function isDarkTerminalTheme(): boolean {
  return typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-color-scheme: dark)').matches
}

function terminalTheme(dark: boolean) {
  return {
    background: dark ? '#1c1c1e' : '#ffffff',
    foreground: dark ? '#d1d1d6' : '#1d1d1f',
    cursor: dark ? '#0a84ff' : '#007aff',
    cursorAccent: dark ? '#1c1c1e' : '#ffffff',
    selectionBackground: dark ? 'rgba(10, 132, 255, 0.28)' : 'rgba(0, 122, 255, 0.18)',
    black: dark ? '#48484a' : '#1d1d1f',
    red: dark ? '#ff453a' : '#ff3b30',
    green: dark ? '#30d158' : '#34c759',
    yellow: dark ? '#ffd60a' : '#ff9f0a',
    blue: dark ? '#0a84ff' : '#007aff',
    magenta: dark ? '#bf5af2' : '#af52de',
    cyan: dark ? '#64d2ff' : '#32ade6',
    white: dark ? '#d1d1d6' : '#3a3a3c',
    brightBlack: dark ? '#8e8e93' : '#6e6e73',
    brightRed: dark ? '#ff6961' : '#d70015',
    brightGreen: dark ? '#63e6be' : '#248a3d',
    brightYellow: dark ? '#ffe066' : '#c66a00',
    brightBlue: dark ? '#409cff' : '#006bd6',
    brightMagenta: dark ? '#da8fff' : '#8944ab',
    brightCyan: dark ? '#86e1ff' : '#0071a4',
    brightWhite: dark ? '#f5f5f7' : '#1d1d1f'
  }
}

function applyTerminalTheme(dark = isDarkTerminalTheme()) {
  if (termInstance && xtermLoaded.value) {
    termInstance.options.theme = terminalTheme(dark)
  }
}

function startTerminalThemeWatcher() {
  if (typeof window === 'undefined' || !window.matchMedia || terminalSchemeQuery) return
  terminalSchemeQuery = window.matchMedia('(prefers-color-scheme: dark)')
  terminalSchemeListener = (event: MediaQueryListEvent) => applyTerminalTheme(event.matches)
  terminalSchemeQuery.addEventListener?.('change', terminalSchemeListener)
  if (!terminalSchemeQuery.addEventListener) {
    terminalSchemeQuery.addListener(terminalSchemeListener)
  }
}

function stopTerminalThemeWatcher() {
  if (!terminalSchemeQuery || !terminalSchemeListener) return
  terminalSchemeQuery.removeEventListener?.('change', terminalSchemeListener)
  if (!terminalSchemeQuery.removeEventListener) {
    terminalSchemeQuery.removeListener(terminalSchemeListener)
  }
  terminalSchemeQuery = null
  terminalSchemeListener = null
}

async function initXterm() {
  if (!terminalEl.value || xtermLoaded.value) return
  try {
    const { Terminal } = await import('@xterm/xterm')
    const { FitAddon } = await import('@xterm/addon-fit')
    await import('@xterm/xterm/css/xterm.css')

    fitAddon = new FitAddon()
    termInstance = new Terminal({
      theme: terminalTheme(isDarkTerminalTheme()),
      fontFamily: '"SF Mono", "JetBrains Mono", "Cascadia Code", ui-monospace, monospace',
      fontSize: 12,
      lineHeight: 1.4,
      cursorBlink: true,
      cursorStyle: 'bar',
      scrollback: 5000,
      allowTransparency: true,
      convertEol: true,
    })

    termInstance.loadAddon(fitAddon)
    termInstance.open(terminalEl.value)
    fitAddon.fit()
    xtermLoaded.value = true

    termInstance.writeln('\x1b[90mStarxo terminal\x1b[0m')
    termInstance.writeln('\x1b[90mRun commands in the active sandbox workspace.\x1b[0m')
    termInstance.writeln('')
    lineCount.value = 3
  } catch (e) {
    console.warn('xterm not available, falling back to simple terminal:', e)
    xtermLoaded.value = false
  }
}

function writeToTerminal(data: string, isError = false) {
  lineCount.value++
  if (termInstance && xtermLoaded.value) {
    if (isError) {
      termInstance.writeln(`\x1b[31m${data}\x1b[0m`)
    } else {
      termInstance.writeln(data)
    }
  } else {
    lines.value.push({
      text: data,
      type: isError ? 'stderr' : 'stdout'
    })
    if (autoScroll.value) {
      nextTick(() => {
        const el = terminalEl.value
        if (el) el.scrollTop = el.scrollHeight
      })
    }
  }
}

function writeCommandEcho(command: string) {
  lineCount.value++
  if (termInstance && xtermLoaded.value) {
    termInstance.writeln(`\x1b[90m$\x1b[0m ${command}`)
  } else {
    lines.value.push({ text: `$ ${command}`, type: 'info' })
    if (autoScroll.value) {
      nextTick(() => {
        const el = terminalEl.value
        if (el) el.scrollTop = el.scrollHeight
      })
    }
  }
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return String(error || t('terminal.commandFailed'))
}

async function submitCommand() {
  const command = commandInput.value.trim()
  if (!command || !canRunCommand.value) return
  commandInput.value = ''
  writeCommandEcho(command)
  commandRunning.value = true
  try {
    await RunTerminalCommand(command)
  } catch (e) {
    writeToTerminal(`${t('terminal.commandFailed')}: ${errorMessage(e)}`, true)
  } finally {
    commandRunning.value = false
  }
}

function clearTerminal() {
  lineCount.value = 0
  if (termInstance && xtermLoaded.value) {
    termInstance.clear()
  } else {
    lines.value = []
  }
}

useWailsEvent<{ stdout?: string; stderr?: string; exitCode?: number }>('terminal:output', (data) => {
  if (data.stdout) writeToTerminal(data.stdout)
  if (data.stderr) writeToTerminal(data.stderr, true)
  if (data.exitCode !== undefined && data.exitCode !== 0) {
    writeToTerminal(`Process exited with code ${data.exitCode}`, true)
  }
})

useWailsEvent('container:ready', () => {
  if (termInstance && xtermLoaded.value) {
    termInstance.writeln(`\x1b[32m[${formatTime()}] Sandbox connected and ready.\x1b[0m`)
    termInstance.writeln('')
    lineCount.value += 2
  }
})

useWailsEvent<{ step: string; percent: number }>('container:progress', (data) => {
  if (termInstance && xtermLoaded.value) {
    termInstance.writeln(`\x1b[36m[${formatTime()}] [${data.percent}%] ${data.step}\x1b[0m`)
    lineCount.value++
  }
})

onMounted(() => {
  startTerminalThemeWatcher()
  nextTick(() => initXterm())
})

onUnmounted(() => {
  if (termInstance) {
    termInstance.dispose()
    termInstance = null
    xtermLoaded.value = false
  }
  stopTerminalThemeWatcher()
})

// Handle resize
const resizeObserver = ref<ResizeObserver | null>(null)
onMounted(() => {
  if (terminalEl.value) {
    resizeObserver.value = new ResizeObserver(() => {
      if (fitAddon && xtermLoaded.value) {
        try { fitAddon.fit() } catch (_) { /* ignore */ }
      }
    })
    resizeObserver.value.observe(terminalEl.value)
  }
})

onUnmounted(() => {
  resizeObserver.value?.disconnect()
})
</script>

<template>
  <div class="terminal-panel">
    <div class="terminal-header">
      <span class="terminal-title">{{ t('terminal.output') }}</span>
      <NTooltip trigger="hover" placement="left">
        <template #trigger>
          <NButton
            quaternary
            circle
            size="tiny"
            class="clear-btn"
            :aria-label="t('terminal.clear')"
            @click="clearTerminal"
          >
            <template #icon>
              <NIcon size="14"><TrashOutline /></NIcon>
            </template>
          </NButton>
        </template>
        {{ t('terminal.clear') }}
      </NTooltip>
    </div>
    <div ref="terminalEl" class="terminal-container">
      <!-- Fallback if xterm doesn't load -->
      <template v-if="!xtermLoaded">
        <SxTerminalRow
          v-for="(line, i) in lines"
          :key="i"
          :text="line.text"
          :type="line.type"
        />
        <div v-if="lines.length === 0" class="term-placeholder">
          {{ t('terminal.waitingForOutput') }}
        </div>
      </template>
    </div>
    <form class="terminal-command-bar" @submit.prevent="submitCommand">
      <span class="terminal-prompt">$</span>
      <NInput
        v-model:value="commandInput"
        size="small"
        class="terminal-command-input"
        :placeholder="commandPlaceholder"
        :disabled="!sshConnected || !activeContainer"
        :loading="commandRunning"
        clearable
      />
      <NButton
        size="small"
        secondary
        attr-type="submit"
        :disabled="!canRunCommand || !commandInput.trim()"
        :loading="commandRunning"
      >
        <template #icon>
          <NIcon size="14"><PaperPlaneOutline /></NIcon>
        </template>
        {{ t('terminal.run') }}
      </NButton>
    </form>
    <!-- Status Bar -->
    <div class="terminal-status-bar">
      <div class="status-left">
        <span class="status-dot" :class="sshConnected ? 'connected' : 'disconnected'" />
        <span class="status-label">{{ sshConnected ? 'SSH' : 'Disconnected' }}</span>
        <template v-if="activeContainer">
          <span class="status-sep">|</span>
          <NIcon size="11"><Cube /></NIcon>
          <NTooltip trigger="hover" placement="top">
            <template #trigger>
              <span class="status-label status-container">{{ activeContainer }}</span>
            </template>
            {{ activeContainer }}
          </NTooltip>
        </template>
      </div>
      <div class="status-right">
        <span class="line-count">{{ lineCount }} lines</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.terminal-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  flex: 1;
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px 6px 12px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.terminal-title {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: none;
  letter-spacing: 0;
}

.clear-btn {
  color: var(--text-faint) !important;
}

.clear-btn:hover {
  color: var(--text-secondary) !important;
}

.terminal-container {
  flex: 1;
  overflow-y: auto;
  background: var(--platform-bg-content);
  padding: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.4;
}

.terminal-command-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border-top: 1px solid var(--border-subtle);
  background: color-mix(in srgb, var(--platform-bg-toolbar) 86%, transparent);
}

.terminal-prompt {
  flex: 0 0 auto;
  color: var(--text-faint);
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 700;
}

.terminal-command-input {
  flex: 1;
  min-width: 0;
}

.term-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.term-stdout {
  color: var(--text-secondary);
}

.term-stderr {
  color: var(--accent-rose);
  background: rgba(244, 63, 94, 0.06);
  border-left: 2px solid var(--accent-rose);
  padding-left: 8px;
  margin-left: -8px;
}

.term-info {
  color: var(--accent-cyan);
}

.term-placeholder {
  color: var(--text-faint);
  font-style: italic;
  padding: 12px 4px;
}

/* Status Bar */
.terminal-status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px;
  background: color-mix(in srgb, var(--platform-bg-toolbar) 86%, transparent);
  border-top: 1px solid var(--border-subtle);
  font-size: var(--fs-2xs);
  font-family: var(--font-sans);
  color: var(--text-faint);
  flex-shrink: 0;
  gap: 8px;
}

.status-container {
  max-width: 220px;
  cursor: help;
}

.status-left {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.status-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.connected {
  background: var(--accent-emerald);
}

.status-dot.disconnected {
  background: var(--text-faint);
}

.status-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-sep {
  color: var(--border-subtle);
}

.line-count {
  color: var(--text-faint);
}
</style>
