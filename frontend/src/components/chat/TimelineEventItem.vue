<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NIcon, NButton } from 'naive-ui'
import {
  Build, CheckmarkCircle, Reload, InformationCircle, AlertCircle,
  DocumentText, Terminal, CodeSlash, People, ChevronForward, CloseCircle, FolderOpen
} from '@vicons/ionicons5'
import { useMarkdown } from '@/composables/useHelpers'
import type { TurnEvent } from '@/types/message'
import { useI18n } from 'vue-i18n'
import { openWorkspacePath } from '@/composables/useWorkspaceBridge'

const { t } = useI18n()

const props = defineProps<{
  event: TurnEvent
  showAgentBadge?: boolean
}>()

const { renderMarkdown } = useMarkdown()
const renderedContent = computed(() => renderMarkdown(props.event.content))

const expanded = ref(false)
function toggleExpanded() {
  if (!hasDetails.value) return
  expanded.value = !expanded.value
}

// Result truncation
const resultTruncateLimit = 500
const showFullResult = ref(false)
const truncatedResult = computed(() => {
  if (!props.event.toolResult) return ''
  if (showFullResult.value || props.event.toolResult.length <= resultTruncateLimit) {
    return props.event.toolResult
  }
  return props.event.toolResult.substring(0, resultTruncateLimit) + '...'
})
const isResultTruncated = computed(() =>
  !!props.event.toolResult && props.event.toolResult.length > resultTruncateLimit && !showFullResult.value
)

// ---------- Tool categorization ----------
type ToolCategory = 'file' | 'shell' | 'edit' | 'agent' | 'todo' | 'notify' | 'other'

interface ToolDisplayInfo {
  category: ToolCategory
  color: string
  action: string
  primary: string
  secondary?: string
}

function tryParseArgs(args?: string): any {
  if (!args) return null
  try { return JSON.parse(args) } catch { return null }
}

function truncStr(s: string, max: number): string {
  return s.length <= max ? s : s.substring(0, max) + '...'
}

function firstLine(s: string): string {
  return s.split('\n')[0] || ''
}

function parseExitCode(result: string): number | null {
  const m = result.match(/(?:exit(?:\s+code)?|code)\s*[:=]\s*(-?\d+)/i)
  if (!m) return null
  const n = Number.parseInt(m[1], 10)
  return Number.isNaN(n) ? null : n
}

function countLines(s: string): number {
  if (!s) return 0
  return s.split('\n').length
}

function jsonInline(v: unknown): string {
  try {
    return JSON.stringify(v)
  } catch {
    return ''
  }
}

function todoStats(todos: TodoItem[]): string {
  const total = todos.length
  if (total === 0) return ''
  let done = 0
  let doing = 0
  let todo = 0
  for (const item of todos) {
    if (item.status === 'done') done++
    else if (item.status === 'in_progress') doing++
    else todo++
  }
  return `${done}/${doing}/${todo}`
}

interface TodoItem {
  id: string
  title: string
  status: 'pending' | 'in_progress' | 'done' | 'failed' | 'blocked'
  depends_on?: string[]
}

// ---------- Parsed todos for write_todos / update_todo tools ----------
const parsedTodos = computed<TodoItem[]>(() => {
  if (props.event.toolName === 'write_todos') {
    const args = tryParseArgs(props.event.toolArgs)
    if (args?.todos && Array.isArray(args.todos)) {
      return args.todos as TodoItem[]
    }
    return []
  }
  if (props.event.toolName === 'update_todo' && props.event.toolResult) {
    const parts = props.event.toolResult.split('---\n')
    if (parts.length >= 2) {
      try {
        const todos = JSON.parse(parts[parts.length - 1])
        if (Array.isArray(todos)) return todos as TodoItem[]
      } catch {
        // ignore parse failure
      }
    }
  }
  return []
})

const toolInfo = computed<ToolDisplayInfo>(() => {
  const name = props.event.toolName || ''
  const args = tryParseArgs(props.event.toolArgs)
  const result = props.event.toolResult || ''
  const exitCode = parseExitCode(result)

  if (name === 'read_file') {
    return {
      category: 'file',
      color: 'var(--agent-file-manager)',
      action: t('message.tool.read'),
      primary: args?.path || '-',
      secondary: result ? `${result.length} ${t('message.tool.chars')}` : undefined,
    }
  }

  if (name === 'write_file') {
    return {
      category: 'file',
      color: 'var(--agent-file-manager)',
      action: t('message.tool.write'),
      primary: args?.path || '-',
      secondary: result ? t('message.tool.saved') : undefined,
    }
  }

  if (name === 'list_files') {
    return {
      category: 'file',
      color: 'var(--agent-file-manager)',
      action: t('message.tool.list'),
      primary: args?.path || '/workspace',
      secondary: result ? `${countLines(result)} ${t('message.tool.lines')}` : undefined,
    }
  }

  if (name === 'str_replace_editor') {
    const cmd = args?.command || 'edit'
    return {
      category: 'edit',
      color: 'var(--agent-code-writer)',
      action: t('message.tool.edit'),
      primary: args?.path || '-',
      secondary: cmd,
    }
  }

  if (name === 'shell_execute') {
    return {
      category: 'shell',
      color: 'var(--agent-code-executor)',
      action: t('message.tool.shell'),
      primary: truncStr(firstLine(args?.command || '') || '-', 80),
      secondary: exitCode !== null ? `exit ${exitCode}` : result ? `${countLines(result)} ${t('message.tool.lines')}` : undefined,
    }
  }

  if (name === 'python_execute') {
    return {
      category: 'shell',
      color: 'var(--agent-code-executor)',
      action: t('message.tool.python'),
      primary: truncStr(firstLine(args?.code || '') || '-', 80),
      secondary: exitCode !== null ? `exit ${exitCode}` : result ? `${countLines(result)} ${t('message.tool.lines')}` : undefined,
    }
  }

  if (name === 'task') {
    return {
      category: 'agent',
      color: 'var(--agent-orchestrator)',
      action: t('message.tool.delegate'),
      primary: args?.subagent_type || 'sub-agent',
      secondary: truncStr(args?.description || '', 60) || undefined,
    }
  }

  if (name === 'notify_user') {
    const msgFromResult = result.replace(/^\[Status\]\s*/, '')
    const msg = msgFromResult || args?.message || ''
    return {
      category: 'notify',
      color: 'var(--agent-orchestrator)',
      action: t('message.tool.notify'),
      primary: truncStr(msg || '-', 80),
      secondary: result ? t('status.done') : undefined,
    }
  }

  if (name === 'update_todo') {
    const detail = args ? `${args.id} -> ${args.status}` : '-'
    return {
      category: 'todo',
      color: 'var(--agent-default)',
      action: t('message.tool.todoUpdate'),
      primary: detail,
      secondary: parsedTodos.value.length > 0 ? todoStats(parsedTodos.value) : undefined,
    }
  }

  if (name === 'write_todos') {
    return {
      category: 'todo',
      color: 'var(--agent-default)',
      action: t('message.tool.todos'),
      primary: parsedTodos.value.length > 0
        ? `${parsedTodos.value.length} ${t('message.tool.items')}`
        : '-',
      secondary: parsedTodos.value.length > 0 ? todoStats(parsedTodos.value) : undefined,
    }
  }

  return {
    category: 'other',
    color: 'var(--agent-default)',
    action: name || t('message.tool.tool'),
    primary: truncStr(jsonInline(args) || '-', 80),
  }
})

function agentColor(name: string): string {
  if (!name) return 'var(--text-muted)'
  if (name.includes('orchestrator')) return 'var(--agent-orchestrator)'
  if (name.includes('writer') || name.includes('code_w')) return 'var(--agent-code-writer)'
  if (name.includes('executor') || name.includes('code_e')) return 'var(--agent-code-executor)'
  if (name.includes('file')) return 'var(--agent-file-manager)'
  return 'var(--agent-default)'
}

function agentLabel(name: string): string {
  const labels: Record<string, string> = {
    'orchestrator': 'Orchestrator',
    'code_writer': 'Code Writer',
    'code_executor': 'Code Executor',
    'file_manager': 'File Manager',
    'coding_agent': 'Coding Agent'
  }
  return labels[name] || name
}

function formatArgs(args: string): string {
  if (!args) return ''
  try { return JSON.stringify(JSON.parse(args), null, 2) } catch { return args }
}

const hasResult = computed(() => !!props.event.toolResult)
const hasDetails = computed(() =>
  toolInfo.value.category !== 'todo' && (!!props.event.toolArgs || !!props.event.toolResult)
)

type ToolStatus = 'running' | 'done' | 'error'
const toolStatus = computed<ToolStatus>(() => {
  const result = props.event.toolResult
  if (!result) return 'running'
  const exit = parseExitCode(result)
  if (exit !== null && exit !== 0) return 'error'
  return 'done'
})

const statusLabel = computed(() => {
  if (toolStatus.value === 'running') return t('taskRail.running')
  if (toolStatus.value === 'error') return t('taskRail.failed')
  return t('taskRail.done')
})

const canOpenWorkspacePath = computed(() => {
  const category = toolInfo.value.category
  const path = toolInfo.value.primary || ''
  return (category === 'file' || category === 'edit') && path.startsWith('/')
})
</script>

<template>
  <div :class="['timeline-event', `event-${event.type}`]">
    <!-- Message event -->
    <template v-if="event.type === 'message'">
      <div class="event-message">
        <div v-if="showAgentBadge !== false" class="event-agent-badge" :style="{ '--agent-color': agentColor(event.agent) }">
          {{ agentLabel(event.agent) }}
        </div>
        <div class="event-message-content markdown-body">
          <div v-html="renderedContent"></div>
          <span v-if="event.isStreaming" class="streaming-cursor">&#9608;</span>
        </div>
      </div>
    </template>

    <!-- Tool call event -->
    <template v-else-if="event.type === 'tool_call'">
      <div class="event-tool-call">
        <div
          class="tool-strip"
          :class="[`tool-strip-${toolInfo.category}`, { expandable: hasDetails }]"
          :role="hasDetails ? 'button' : undefined"
          :tabindex="hasDetails ? 0 : -1"
          :aria-expanded="expanded"
          @click="toggleExpanded"
          @keydown.enter.prevent="toggleExpanded"
          @keydown.space.prevent="toggleExpanded"
        >
          <NIcon size="13" :style="{ color: toolInfo.color }">
            <DocumentText v-if="toolInfo.category === 'file'" />
            <CodeSlash v-else-if="toolInfo.category === 'edit'" />
            <Terminal v-else-if="toolInfo.category === 'shell'" />
            <People v-else-if="toolInfo.category === 'agent'" />
            <InformationCircle v-else-if="toolInfo.category === 'notify'" />
            <Build v-else />
          </NIcon>
          <span class="tool-strip-action" :style="{ color: toolInfo.color }">{{ toolInfo.action }}</span>
          <span class="tool-strip-primary" :title="toolInfo.primary">{{ toolInfo.primary }}</span>
          <span v-if="toolInfo.secondary" class="tool-strip-secondary">{{ toolInfo.secondary }}</span>
          <button
            v-if="canOpenWorkspacePath"
            type="button"
            class="tool-open-path"
            :aria-label="t('workspace.openFile')"
            @click.stop="openWorkspacePath(toolInfo.primary)"
          >
            <NIcon size="12"><FolderOpen /></NIcon>
          </button>
          <span class="tool-status-pill" :class="`status-${toolStatus}`">
            <NIcon size="11" class="status-pill-icon">
              <CheckmarkCircle v-if="toolStatus === 'done'" />
              <CloseCircle v-else-if="toolStatus === 'error'" />
              <Reload v-else />
            </NIcon>
            <span class="status-pill-label">{{ statusLabel }}</span>
          </span>
          <span v-if="hasDetails" class="tool-strip-chevron" :class="{ expanded }">
            <NIcon size="11"><ChevronForward /></NIcon>
          </span>
        </div>

        <transition name="expand">
          <div v-if="expanded && hasDetails" class="tool-call-body">
            <div v-if="event.toolArgs" class="tool-section">
              <div class="tool-section-label">{{ t('message.arguments') }}</div>
              <pre class="tool-code" :class="{ 'tool-code-shell': toolInfo.category === 'shell' }">{{ formatArgs(event.toolArgs) }}</pre>
            </div>
            <div v-if="event.toolResult" class="tool-section">
              <div class="tool-section-label">{{ t('message.result') }}</div>
              <pre class="tool-code tool-result-code">{{ truncatedResult }}</pre>
              <NButton
                v-if="isResultTruncated"
                quaternary
                size="tiny"
                class="expand-btn"
                @click.stop="showFullResult = true"
              >
                {{ t('message.showFull') }}
              </NButton>
            </div>
            <div v-if="!event.toolArgs && !event.toolResult" class="tool-section">
              <span class="tool-executing">{{ t('message.executing') }}</span>
            </div>
          </div>
        </transition>
      </div>
    </template>

    <!-- Transfer event (standalone fallback) -->
    <template v-else-if="event.type === 'transfer'">
      <div class="event-transfer-inline">
        <span class="transfer-text">
          <span :style="{ color: agentColor(event.agent) }">{{ agentLabel(event.agent) }}</span>
          <NIcon size="12" class="transfer-arrow"><ChevronForward /></NIcon>
          <span :style="{ color: agentColor(event.content) }">{{ agentLabel(event.content) }}</span>
        </span>
        <span v-if="event.toolArgs" class="transfer-desc">{{ event.toolArgs }}</span>
      </div>
    </template>

    <!-- Interrupt event -->
    <template v-else-if="event.type === 'interrupt'">
      <div class="event-interrupt">
        <NIcon size="14"><AlertCircle /></NIcon>
        <span>{{ t('interrupt.agentNeedsInfo') }}</span>
      </div>
    </template>

    <!-- Info event -->
    <template v-else-if="event.type === 'info'">
      <div class="event-info">
        <NIcon size="12"><InformationCircle /></NIcon>
        <span>{{ event.content }}</span>
      </div>
    </template>

    <!-- Reasoning: agent's intent explanation before tool calls -->
    <template v-else-if="event.type === 'reasoning'">
      <div class="event-reasoning">
        <span class="reasoning-agent" :style="{ color: agentColor(event.agent) }">
          {{ agentLabel(event.agent) }}
        </span>
        <span class="reasoning-text">{{ event.content }}</span>
      </div>
    </template>

    <!-- Thinking: fallback "thinking..." animation -->
    <template v-else-if="event.type === 'thinking'">
      <div class="event-thinking">
        <span class="thinking-dots">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </span>
        <span class="thinking-agent" :style="{ color: agentColor(event.agent) }">
          {{ agentLabel(event.agent) }}
        </span>
        <span class="thinking-label">{{ t('message.thinking') }}</span>
      </div>
    </template>
  </div>
</template>

<style scoped>
.timeline-event {
  animation: fadeIn 200ms ease both;
}

/* Message */
.event-message {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.event-agent-badge {
  display: inline-flex;
  align-items: center;
  font-size: 11px;
  font-weight: 700;
  font-family: var(--font-mono);
  color: var(--agent-color);
  padding: 2px 8px;
  background: color-mix(in srgb, var(--agent-color) 10%, transparent);
  border-radius: 4px;
  width: fit-content;
  letter-spacing: 0.3px;
}

.event-message-content {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 4px 12px 12px 12px;
  padding: 12px 16px;
  font-size: 13.5px;
  line-height: 1.7;
}

.streaming-cursor {
  display: inline-block;
  color: var(--accent-cyan);
  font-size: 14px;
  line-height: 1;
  vertical-align: text-bottom;
  animation: blink 800ms steps(2) infinite;
}

@keyframes blink {
  0% { opacity: 1; }
  50% { opacity: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .streaming-cursor {
    animation: none;
    opacity: 0.3;
  }
}

/* Tool call compact strip */
.event-tool-call {
  margin: 2px 0;
}

.tool-strip {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  padding: 4px 8px;
  border-radius: 8px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-surface);
  color: var(--text-secondary);
  appearance: none;
  text-align: left;
}

.tool-strip.expandable {
  cursor: pointer;
}

.tool-strip.expandable:hover {
  border-color: color-mix(in srgb, var(--accent-cyan) 30%, var(--border-subtle));
  background: var(--bg-hover);
}

.tool-strip-file {
  border-left: 3px solid var(--agent-file-manager);
}

.tool-strip-edit {
  border-left: 3px solid var(--agent-code-writer);
}

.tool-strip-shell {
  border-left: 3px solid var(--agent-code-executor);
}

.tool-strip-agent {
  border-left: 3px solid var(--agent-orchestrator);
}

.tool-strip-todo {
  border-left: 3px solid var(--agent-default);
}

.tool-strip-notify {
  border-left: 3px solid var(--agent-orchestrator);
}

.tool-strip-other {
  border-left: 3px solid var(--text-muted);
}

.tool-strip-action {
  font-size: 11px;
  font-weight: 700;
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.tool-strip-primary {
  font-size: 11.5px;
  font-family: var(--font-mono);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.tool-strip-secondary {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  white-space: nowrap;
  flex-shrink: 0;
}

.tool-open-path {
  width: 24px;
  height: 22px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-deepest);
  color: var(--text-muted);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: color var(--transition-ui), background var(--transition-ui), border-color var(--transition-ui);
}

.tool-open-path:hover,
.tool-open-path:focus-visible {
  color: var(--accent-cyan);
  background: var(--bg-hover);
  border-color: var(--accent-cyan-dim);
}

.tool-strip-chevron {
  color: var(--text-faint);
  display: inline-flex;
  align-items: center;
  transition: transform 180ms ease;
  margin-left: 2px;
}

.tool-strip-chevron.expanded {
  transform: rotate(90deg);
}

.tool-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  margin-left: auto;
  flex-shrink: 0;
  border-radius: 999px;
  font-size: 10px;
  font-family: var(--font-mono);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.3px;
  border: 1px solid transparent;
  line-height: 1.2;
}

.status-pill-icon {
  display: inline-flex;
  align-items: center;
}

.status-pill-label {
  white-space: nowrap;
}

.tool-status-pill.status-done {
  color: var(--accent-emerald);
  background: color-mix(in srgb, var(--accent-emerald) 10%, transparent);
  border-color: color-mix(in srgb, var(--accent-emerald) 22%, transparent);
}

.tool-status-pill.status-error {
  color: var(--accent-rose, #f43f5e);
  background: color-mix(in srgb, var(--accent-rose, #f43f5e) 10%, transparent);
  border-color: color-mix(in srgb, var(--accent-rose, #f43f5e) 22%, transparent);
}

.tool-status-pill.status-running {
  color: var(--text-muted);
  background: var(--bg-deepest);
  border-color: var(--border-subtle);
}

.tool-status-pill.status-running .status-pill-icon {
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .tool-status-pill.status-running .status-pill-icon {
    animation: none;
  }
}

.tool-call-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 4px 2px 4px;
}

.tool-section-label {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  color: var(--text-faint);
  margin-bottom: 4px;
}

.tool-code {
  background: var(--bg-deepest);
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  padding: 8px 12px;
  font-family: var(--font-mono);
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-secondary);
  overflow-x: auto;
  white-space: pre-wrap;
  overflow-wrap: break-word;
  margin: 0;
  max-height: 300px;
  overflow-y: auto;
}

.tool-code-shell {
  border-left: 3px solid var(--agent-code-executor);
  background: rgba(167, 139, 250, 0.05);
}

.tool-result-code {
  border-left: 3px solid var(--accent-emerald-dim);
}

.tool-executing {
  font-size: 11px;
  color: var(--text-faint);
  font-style: italic;
}

.expand-btn {
  margin-top: 4px;
  font-size: 11px !important;
  color: var(--accent-cyan) !important;
}

/* Transfer inline (fallback) */
.event-transfer-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.transfer-text {
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-family: var(--font-mono);
}

.transfer-arrow {
  color: var(--text-faint);
  display: inline-flex;
  align-items: center;
}

/* Interrupt */
.event-interrupt {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--accent-amber);
  padding: 4px 10px;
  background: rgba(245, 158, 11, 0.08);
  border-radius: var(--radius-sm);
  border: 1px solid rgba(245, 158, 11, 0.2);
}

/* Info */
.event-info {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--text-faint);
  font-style: italic;
  padding: 2px 0;
}

/* Reasoning: agent intent explanation */
.event-reasoning {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 12px;
  padding: 6px 12px;
  background: rgba(139, 141, 163, 0.06);
  border-radius: var(--radius-sm);
  border-left: 3px solid rgba(139, 141, 163, 0.3);
  margin: 2px 0;
}

.reasoning-agent {
  font-size: 11px;
  font-weight: 700;
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.reasoning-text {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

/* Thinking: animated dots indicator */
.event-thinking {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  padding: 6px 12px;
  margin: 2px 0;
}

.thinking-dots {
  display: flex;
  gap: 3px;
  align-items: center;
}

.thinking-dots .dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--text-faint);
  animation: thinking-bounce 1.4s ease-in-out infinite;
}

.thinking-dots .dot:nth-child(2) {
  animation-delay: 0.2s;
}

.thinking-dots .dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes thinking-bounce {
  0%, 80%, 100% {
    opacity: 0.3;
    transform: scale(0.8);
  }
  40% {
    opacity: 1;
    transform: scale(1);
  }
}

.thinking-agent {
  font-size: 11px;
  font-weight: 700;
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.thinking-label {
  color: var(--text-faint);
  font-size: 11px;
  font-style: italic;
}

/* Transfer description */
.transfer-desc {
  font-size: 11px;
  color: var(--text-muted);
  margin-left: 4px;
}

.expand-enter-active,
.expand-leave-active {
  transition: all 180ms ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
}
</style>
