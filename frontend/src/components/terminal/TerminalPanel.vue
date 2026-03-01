<script lang="ts" setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { NButton, NIcon } from 'naive-ui'
import { TrashOutline } from '@vicons/ionicons5'
import { useWailsEvent } from '@/composables/useWailsEvent'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const terminalEl = ref<HTMLElement | null>(null)
const lines = ref<Array<{ text: string; type: 'stdout' | 'stderr' | 'info' }>>([])
const autoScroll = ref(true)

let termInstance: any = null
let fitAddon: any = null
let xtermLoaded = false

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

    termInstance.writeln('\x1b[36m--- Starxo Terminal ---\x1b[0m')
    termInstance.writeln('\x1b[90mWaiting for sandbox connection...\x1b[0m')
    termInstance.writeln('')
  } catch (e) {
    console.warn('xterm not available, falling back to simple terminal:', e)
    xtermLoaded = false
  }
}

function writeToTerminal(data: string, isError = false) {
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

useWailsEvent('sandbox:ready', () => {
  if (termInstance && xtermLoaded) {
    termInstance.writeln('\x1b[32m✓ Sandbox connected and ready.\x1b[0m')
    termInstance.writeln('')
  }
})

useWailsEvent<{ step: string; percent: number }>('sandbox:progress', (data) => {
  if (termInstance && xtermLoaded) {
    termInstance.writeln(`\x1b[36m[${data.percent}%] ${data.step}\x1b[0m`)
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
      <NButton
        quaternary
        circle
        size="tiny"
        class="clear-btn"
        @click="clearTerminal"
      >
        <template #icon>
          <NIcon size="14"><TrashOutline /></NIcon>
        </template>
      </NButton>
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
}

.term-info {
  color: var(--accent-cyan);
}

.term-placeholder {
  color: var(--text-faint);
  font-style: italic;
  padding: 12px 4px;
}
</style>
