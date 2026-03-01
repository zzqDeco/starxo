<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NIcon, NCollapse, NCollapseItem, NButton } from 'naive-ui'
import {
  Build, CheckmarkCircle, Reload, InformationCircle, AlertCircle,
  DocumentText, Terminal, CodeSlash, People, FolderOpen
} from '@vicons/ionicons5'
import { useMarkdown } from '@/composables/useHelpers'
import type { TurnEvent } from '@/types/message'
import TodoBoard from './TodoBoard.vue'
import type { TodoItem } from './TodoBoard.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  event: TurnEvent
  showAgentBadge?: boolean
}>()

const { renderMarkdown } = useMarkdown()
const renderedContent = computed(() => renderMarkdown(props.event.content))

// Tool call: auto-expand if no result yet (active), collapse if done
const isExpanded = ref<string[]>(
  (!props.event.toolResult && props.event.type === 'tool_call') ? [props.event.toolId || props.event.id] : []
)

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
  label: string
  detail?: string
}

function tryParseArgs(args?: string): any {
  if (!args) return null
  try { return JSON.parse(args) } catch { return null }
}

function truncStr(s: string, max: number): string {
  return s.length <= max ? s : s.substring(0, max) + '...'
}

const toolInfo = computed<ToolDisplayInfo>(() => {
  const name = props.event.toolName || ''
  const args = tryParseArgs(props.event.toolArgs)

  if (name === 'read_file')
    return { category: 'file', color: '#34d399', label: 'read_file', detail: args?.path }
  if (name === 'write_file')
    return { category: 'file', color: '#34d399', label: 'write_file', detail: args?.path }
  if (name === 'list_files')
    return { category: 'file', color: '#34d399', label: 'list_files', detail: args?.path || '/workspace' }
  if (name === 'str_replace_editor') {
    const cmd = args?.command || 'edit'
    return { category: 'edit', color: '#38bdf8', label: cmd, detail: args?.path }
  }
  if (name === 'shell_execute')
    return { category: 'shell', color: '#a78bfa', label: 'shell', detail: truncStr(args?.command || '', 80) }
  if (name === 'python_execute')
    return { category: 'shell', color: '#a78bfa', label: 'python', detail: truncStr(args?.code?.split('\n')[0] || '', 80) }
  if (name === 'task')
    return { category: 'agent', color: '#22d3ee', label: args?.subagent_type || 'sub-agent', detail: truncStr(args?.description || '', 120) }
  if (name === 'notify_user') {
    const msg = args?.message || ''
    return { category: 'notify', color: '#22d3ee', label: 'notify_user', detail: truncStr(msg, 120) }
  }
  if (name === 'update_todo') {
    const detail = args ? `${args.id} → ${args.status}` : ''
    return { category: 'todo', color: '#f59e0b', label: 'update_todo', detail }
  }
  if (name === 'write_todos')
    return { category: 'todo', color: '#f59e0b', label: 'write_todos' }
  return { category: 'other', color: '#f59e0b', label: name }
})

// ---------- Agent helpers ----------
function agentColor(name: string): string {
  if (!name) return '#8b8da3'
  if (name.includes('orchestrator')) return '#22d3ee'
  if (name.includes('writer') || name.includes('code_w')) return '#38bdf8'
  if (name.includes('executor') || name.includes('code_e')) return '#a78bfa'
  if (name.includes('file')) return '#34d399'
  return '#f59e0b'
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

// ---------- Parsed todos for write_todos / update_todo tools ----------
const parsedTodos = computed<TodoItem[]>(() => {
  // write_todos: todos are in args
  if (props.event.toolName === 'write_todos') {
    const args = tryParseArgs(props.event.toolArgs)
    if (args?.todos && Array.isArray(args.todos)) {
      return args.todos as TodoItem[]
    }
    return []
  }
  // update_todo: updated full list is in the result after "---\n"
  if (props.event.toolName === 'update_todo' && props.event.toolResult) {
    const parts = props.event.toolResult.split('---\n')
    if (parts.length >= 2) {
      try {
        const todos = JSON.parse(parts[parts.length - 1])
        if (Array.isArray(todos)) return todos as TodoItem[]
      } catch { /* ignore */ }
    }
  }
  return []
})

// ---------- Notify message extraction ----------
const notifyMessage = computed(() => {
  if (props.event.toolName !== 'notify_user') return ''
  // Try result first (contains "[Status] message")
  if (props.event.toolResult) {
    return props.event.toolResult.replace(/^\[Status\]\s*/, '')
  }
  // Fallback to args
  const args = tryParseArgs(props.event.toolArgs)
  return args?.message || ''
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
      <!-- Special: notify_user renders as inline status banner -->
      <div v-if="toolInfo.category === 'notify'" class="event-notify">
        <NIcon size="13"><InformationCircle /></NIcon>
        <span v-if="showAgentBadge !== false" class="notify-agent" :style="{ color: agentColor(event.agent) }">{{ agentLabel(event.agent) }}</span>
        <span class="notify-text">{{ notifyMessage || toolInfo.detail }}</span>
      </div>

      <!-- Special: sub-agent delegation card -->
      <div v-else-if="toolInfo.category === 'agent'" class="event-task-card">
        <div class="task-card-header">
          <NIcon size="14" class="task-card-icon"><People /></NIcon>
          <span class="task-card-agent">{{ toolInfo.label }}</span>
          <NIcon v-if="hasResult" size="12" class="tool-status-icon done"><CheckmarkCircle /></NIcon>
          <NIcon v-else size="12" class="tool-status-icon running"><Reload /></NIcon>
        </div>
        <div v-if="toolInfo.detail" class="task-card-desc">{{ toolInfo.detail }}</div>
      </div>

      <!-- Special: write_todos renders as TodoBoard -->
      <div v-else-if="toolInfo.category === 'todo' && parsedTodos.length > 0" class="event-todo">
        <TodoBoard :todos="parsedTodos" />
      </div>

      <!-- Standard tool call with category styling -->
      <div v-else class="event-tool-call">
        <NCollapse v-model:expanded-names="isExpanded" arrow-placement="left">
          <NCollapseItem :name="event.toolId || event.id">
            <template #header>
              <div class="tool-call-header">
                <!-- Category icon -->
                <NIcon size="13" :style="{ color: toolInfo.color }">
                  <DocumentText v-if="toolInfo.category === 'file'" />
                  <CodeSlash v-else-if="toolInfo.category === 'edit'" />
                  <Terminal v-else-if="toolInfo.category === 'shell'" />
                  <Build v-else />
                </NIcon>
                <!-- Tool name -->
                <span class="tool-name" :style="{ color: toolInfo.color }">{{ toolInfo.label }}</span>
                <!-- Path or command detail -->
                <span v-if="toolInfo.detail" class="tool-detail" :class="`tool-detail-${toolInfo.category}`">{{ toolInfo.detail }}</span>
                <!-- Status -->
                <NIcon v-if="hasResult" size="12" class="tool-status-icon done"><CheckmarkCircle /></NIcon>
                <NIcon v-else size="12" class="tool-status-icon running"><Reload /></NIcon>
              </div>
            </template>
            <div class="tool-call-body">
              <!-- Shell: show command in terminal style -->
              <div v-if="toolInfo.category === 'shell' && event.toolArgs" class="tool-section">
                <div class="tool-section-label">{{ t('message.arguments') }}</div>
                <pre class="tool-code tool-code-shell">{{ formatArgs(event.toolArgs) }}</pre>
              </div>
              <!-- File/Edit: show args -->
              <div v-else-if="event.toolArgs" class="tool-section">
                <div class="tool-section-label">{{ t('message.arguments') }}</div>
                <pre class="tool-code">{{ formatArgs(event.toolArgs) }}</pre>
              </div>
              <!-- Result -->
              <div v-if="event.toolResult" class="tool-section">
                <div class="tool-section-label">{{ t('message.result') }}</div>
                <pre class="tool-code tool-result-code">{{ truncatedResult }}</pre>
                <NButton
                  v-if="isResultTruncated"
                  quaternary
                  size="tiny"
                  class="expand-btn"
                  @click="showFullResult = true"
                >
                  {{ t('message.showFull') }}
                </NButton>
              </div>
              <div v-if="!event.toolArgs && !event.toolResult" class="tool-section">
                <span class="tool-executing">{{ t('message.executing') }}</span>
              </div>
            </div>
          </NCollapseItem>
        </NCollapse>
      </div>
    </template>

    <!-- Transfer event (standalone fallback) -->
    <template v-else-if="event.type === 'transfer'">
      <div class="event-transfer-inline">
        <span class="transfer-text">
          <span :style="{ color: agentColor(event.agent) }">{{ agentLabel(event.agent) }}</span>
          <span class="transfer-arrow">&rarr;</span>
          <span :style="{ color: agentColor(event.content) }">{{ agentLabel(event.content) }}</span>
        </span>
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

/* Task delegation card */
.event-task-card {
  background: rgba(34, 211, 238, 0.06);
  border: 1px solid rgba(34, 211, 238, 0.2);
  border-radius: var(--radius-md);
  padding: 10px 14px;
  margin: 4px 0;
}

.task-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-card-icon {
  color: var(--accent-cyan);
}

.task-card-agent {
  font-size: 12px;
  font-weight: 700;
  font-family: var(--font-mono);
  color: var(--accent-cyan);
  letter-spacing: 0.3px;
}

.task-card-desc {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-secondary);
  overflow-wrap: break-word;
}

/* Tool Call */
.event-tool-call {
  margin: 2px 0;
}

.tool-call-header {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.tool-name {
  font-size: 12px;
  font-weight: 700;
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.tool-detail {
  font-size: 11px;
  color: var(--text-muted);
  font-family: var(--font-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.tool-detail-file,
.tool-detail-edit {
  color: var(--text-secondary);
  background: rgba(255,255,255,0.04);
  padding: 1px 6px;
  border-radius: 3px;
}

.tool-detail-shell {
  color: #c4b5fd;
  opacity: 0.8;
}

.tool-status-icon.done {
  color: var(--accent-emerald);
  margin-left: auto;
  flex-shrink: 0;
}

.tool-status-icon.running {
  color: var(--text-faint);
  animation: spin 1.5s linear infinite;
  margin-left: auto;
  flex-shrink: 0;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.tool-call-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
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
  border-left: 3px solid #a78bfa;
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
  font-size: 14px;
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

/* Todo board wrapper */
.event-todo {
  margin: 4px 0;
}

/* Notify banner */
.event-notify {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--accent-cyan);
  padding: 6px 12px;
  background: rgba(34, 211, 238, 0.06);
  border-radius: var(--radius-sm);
  border: 1px solid rgba(34, 211, 238, 0.15);
  margin: 2px 0;
}

.notify-agent {
  font-size: 11px;
  font-weight: 700;
  font-family: var(--font-mono);
  flex-shrink: 0;
}

.notify-text {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.4;
}

/* Collapse overrides */
.event-tool-call :deep(.n-collapse-item__header) {
  padding: 6px 0 !important;
}

.event-tool-call :deep(.n-collapse-item__content-inner) {
  padding-top: 4px !important;
}
</style>
