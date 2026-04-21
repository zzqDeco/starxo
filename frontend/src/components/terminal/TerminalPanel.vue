<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { NButton, NIcon, NTooltip } from 'naive-ui'
import { TrashOutline, Cube } from '@vicons/ionicons5'
import { useWailsEvent } from '@/composables/useWailsEvent'
import { useConnectionStore } from '@/stores/connectionStore'
import { useContainerStore } from '@/stores/containerStore'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const connectionStore = useConnectionStore()
const containerStore = useContainerStore()

const terminalEl = ref<HTMLElement | null>(null)
const lines = ref<Array<{ text: string; type: 'stdout' | 'stderr' | 'info' }>>([])
const autoScroll = ref(true)
const lineCount = ref(0)

let termInstance: any = null
let fitAddon: any = null
let xtermLoaded = false

const sshConnected = computed(() => connectionStore.sshConnected)
const activeContainer = computed(() => containerStore.activeContainerID || '')

function formatTime(): string {
  const now = new Date()
  return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`
}

async function initXterm() {
  if (!terminalEl.value || xtermLoaded) return
  try {
    const { Terminal } = await import('@xterm/xterm')
    const { FitAddon } = await import('@xterm/addon-fit')
    await import('@xterm/xterm/css/xterm.css')

    fitAddon = new FitAddon()
    termInstance = new Terminal({
      theme: {
        background: '#080a14',
        foreground: '#c8c9d6',
        cursor: '#22d3ee',
        cursorAccent: '#080a14',
        selectionBackground: 'rgba(34, 211, 238, 0.2)',
        black: '#0c0e1a',
        red: '#f43f5e',
        green: '#10b981',
        yellow: '#f59e0b',
        blue: '#3b82f6',
        magenta: '#c792ea',
        cyan: '#22d3ee',
        white: '#c8c9d6',
        brightBlack: '#5a5c72',
        brightRed: '#fb7185',
        brightGreen: '#34d399',
        brightYellow: '#fbbf24',
        brightBlue: '#60a5fa',
        brightMagenta: '#ddb6f2',
        brightCyan: '#67e8f9',
        brightWhite: '#f0f0f5'
      },
      fontFamily: '"JetBrains Mono", "Cascadia Code", "Fira Code", Consolas, monospace',
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
    xtermLoaded = true

    termInstance.writeln('\x1b[36m\x1b[1m  Starxo Terminal  \x1b[0m')
    termInstance.writeln('\x1b[90m  AI Coding Agent v0.1.0\x1b[0m')
    termInstance.writeln('\x1b[90m  Waiting for connection...\x1b[0m')
    termInstance.writeln('')
    lineCount.value = 4
  } catch (e) {
    console.warn('xterm not available, falling back to simple terminal:', e)
    xtermLoaded = false
  }
}

function writeToTerminal(data: string, isError = false) {
  lineCount.value++
  if (termInstance && xtermLoaded) {
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

function clearTerminal() {
  lineCount.value = 0
  if (termInstance && xtermLoaded) {
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
  if (termInstance && xtermLoaded) {
    termInstance.writeln(`\x1b[32m[${formatTime()}] Container connected and ready.\x1b[0m`)
    termInstance.writeln('')
    lineCount.value += 2
  }
})

useWailsEvent<{ step: string; percent: number }>('container:progress', (data) => {
  if (termInstance && xtermLoaded) {
    termInstance.writeln(`\x1b[36m[${formatTime()}] [${data.percent}%] ${data.step}\x1b[0m`)
    lineCount.value++
  }
})

onMounted(() => {
  nextTick(() => initXterm())
})

onUnmounted(() => {
  if (termInstance) {
    termInstance.dispose()
    termInstance = null
    xtermLoaded = false
  }
})

// Handle resize
const resizeObserver = ref<ResizeObserver | null>(null)
onMounted(() => {
  if (terminalEl.value) {
    resizeObserver.value = new ResizeObserver(() => {
      if (fitAddon && xtermLoaded) {
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
        <div
          v-for="(line, i) in lines"
          :key="i"
          :class="['term-line', `term-${line.type}`]"
        >{{ line.text }}</div>
        <div v-if="lines.length === 0" class="term-placeholder">
          {{ t('terminal.waitingForOutput') }}
        </div>
      </template>
    </div>
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
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.terminal-title {
  font-size: 11px;
  font-weight: 700;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.8px;
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
  background: #080a14;
  padding: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.4;
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
  background: var(--bg-elevated);
  border-top: 1px solid var(--border-subtle);
  font-size: var(--fs-2xs);
  font-family: var(--font-mono);
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
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.4);
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
