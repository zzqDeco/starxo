<script lang="ts" setup>
import { computed, ref } from 'vue'
import { NIcon } from 'naive-ui'
import {
  HardwareChip, Clipboard, People, CheckmarkCircle, Reload,
  ChevronForward, CodeSlash, Terminal, DocumentText
} from '@vicons/ionicons5'
import { useMarkdown } from '@/composables/useHelpers'
import type { Message, TurnEvent } from '@/types/message'
import TimelineEventItem from './TimelineEventItem.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps<{
  message: Message
}>()

const { renderMarkdown } = useMarkdown()

const renderedContent = computed(() => renderMarkdown(props.message.content))

const isUser = computed(() => props.message.role === 'user')
const isSystem = computed(() => props.message.role === 'system')
const isAssistant = computed(() => !isUser.value && !isSystem.value)

const hasEvents = computed(() => props.message.events && props.message.events.length > 0)

const timeStr = computed(() => {
  const d = new Date(props.message.timestamp)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
})

// ---------- Agent classification ----------
const mainAgents = new Set(['coding_agent', 'orchestrator', ''])

type SegmentType = 'main' | 'subagent' | 'transfer'

interface EventSegment {
  id: string
  type: SegmentType
  agent: string
  events: TurnEvent[]
  fromAgent?: string
  description?: string
  taskDescription?: string
  isRunning: boolean
  toolCalls: number
  messages: number
  lastAction?: string
}

function tryParseArgs(args?: string): any {
  if (!args) return null
  try { return JSON.parse(args) } catch { return null }
}

function truncStr(s: string, max: number): string {
  return s.length <= max ? s : s.substring(0, max) + '...'
}

function summarizeToolCall(evt: TurnEvent): string {
  const name = evt.toolName || 'tool'
  const args = tryParseArgs(evt.toolArgs)

  if (name === 'read_file') return `Read ${args?.path || '-'}`
  if (name === 'write_file') return `Write ${args?.path || '-'}`
  if (name === 'list_files') return `List ${args?.path || '/workspace'}`
  if (name === 'str_replace_editor') return `Edit ${args?.path || '-'}`
  if (name === 'shell_execute') return `Shell ${truncStr((args?.command || '').split('\n')[0] || '-', 70)}`
  if (name === 'python_execute') return `Python ${truncStr((args?.code || '').split('\n')[0] || '-', 70)}`
  if (name === 'task') return `Delegate ${args?.subagent_type || 'sub-agent'}`
  if (name === 'write_todos') return 'Write todos'
  if (name === 'update_todo') return `Todo ${args?.id || '-'} -> ${args?.status || '-'}`
  if (name === 'notify_user') return `Notify ${truncStr(args?.message || '-', 50)}`

  return truncStr(name, 70)
}

function summarizeEvent(evt: TurnEvent): string {
  if (evt.type === 'tool_call') return summarizeToolCall(evt)
  if (evt.type === 'message') return truncStr(evt.content || '', 100)
  if (evt.type === 'reasoning') return truncStr(evt.content || '', 100)
  if (evt.type === 'thinking') return t('message.thinking')
  if (evt.type === 'interrupt') return t('interrupt.agentNeedsInfo')
  if (evt.type === 'info') return truncStr(evt.content || '', 100)
  return ''
}

function finalizeSegment(seg: EventSegment): EventSegment {
  const toolCalls = seg.events.filter(e => e.type === 'tool_call').length
  const messages = seg.events.filter(e => e.type === 'message').length
  const hasPendingTool = seg.events.some(e => e.type === 'tool_call' && !e.toolResult)
  const hasStreaming = seg.events.some(e => e.type === 'message' && e.isStreaming)
  const hasThinking = seg.events.length > 0 && seg.events[seg.events.length - 1].type === 'thinking'
  const isRunning = hasPendingTool || hasStreaming || hasThinking

  let lastAction = ''
  for (let i = seg.events.length - 1; i >= 0; i--) {
    const s = summarizeEvent(seg.events[i])
    if (s) {
      lastAction = s
      break
    }
  }

  return {
    ...seg,
    toolCalls,
    messages,
    isRunning,
    lastAction,
  }
}

function createMainSegment(agent: string, seedId: string): EventSegment {
  return {
    id: `main-${seedId}`,
    type: 'main',
    agent,
    events: [],
    isRunning: false,
    toolCalls: 0,
    messages: 0,
  }
}

function createSubSegment(agent: string, seedId: string, taskDescription?: string): EventSegment {
  return {
    id: `sub-${seedId}`,
    type: 'subagent',
    agent,
    events: [],
    taskDescription,
    isRunning: false,
    toolCalls: 0,
    messages: 0,
  }
}

const segments = computed<EventSegment[]>(() => {
  if (!props.message.events || props.message.events.length === 0) return []

  const segs: EventSegment[] = []
  let current: EventSegment | null = null
  let pendingTaskDescription = ''

  const flushCurrent = () => {
    if (!current) return

    if (current.type === 'subagent' && current.events.length === 0 && !current.taskDescription) {
      current = null
      return
    }

    if (current.type === 'main' || current.type === 'subagent') {
      segs.push(finalizeSegment(current))
    } else {
      segs.push(current)
    }
    current = null
  }

  for (const evt of props.message.events) {
    if (evt.type === 'tool_call' && evt.toolName === 'task') {
      const args = tryParseArgs(evt.toolArgs)
      if (args?.description) {
        pendingTaskDescription = args.description
      }
    }

    if (evt.type === 'transfer') {
      flushCurrent()

      const fromAgent = evt.agent || ''
      const toAgent = evt.content || ''

      segs.push({
        id: `transfer-${evt.id}`,
        type: 'transfer',
        agent: toAgent,
        fromAgent,
        description: evt.toolArgs,
        events: [evt],
        isRunning: false,
        toolCalls: 0,
        messages: 0,
      })

      current = createSubSegment(toAgent, evt.id, pendingTaskDescription || undefined)
      pendingTaskDescription = ''
      continue
    }

    const evtAgent = evt.agent || ''
    const segType: SegmentType = mainAgents.has(evtAgent) ? 'main' : 'subagent'

    if (!current) {
      current = segType === 'main'
        ? createMainSegment(evtAgent || 'coding_agent', evt.id)
        : createSubSegment(evtAgent, evt.id)
    } else if (current.type !== segType || current.agent !== evtAgent) {
      flushCurrent()
      current = segType === 'main'
        ? createMainSegment(evtAgent || 'coding_agent', evt.id)
        : createSubSegment(evtAgent, evt.id)
    }

    current.events.push(evt)
  }

  flushCurrent()
  return segs
})

// ---------- Sub-agent expand/collapse ----------
const subAgentToggled = ref<Record<string, boolean>>({})

function isSubAgentExpanded(seg: EventSegment): boolean {
  return !!subAgentToggled.value[seg.id]
}

function toggleSubAgent(seg: EventSegment) {
  subAgentToggled.value = {
    ...subAgentToggled.value,
    [seg.id]: !isSubAgentExpanded(seg),
  }
}

function segmentStats(seg: EventSegment): string {
  const parts: string[] = []
  if (seg.toolCalls > 0) parts.push(`${seg.toolCalls} ${t('message.subagent.tools')}`)
  if (seg.messages > 0) parts.push(`${seg.messages} ${t('message.subagent.msgs')}`)
  return parts.join(' · ') || t('message.subagent.noActions')
}

function segmentStatusLabel(seg: EventSegment): string {
  return seg.isRunning ? t('message.subagent.running') : t('message.subagent.done')
}

// ---------- Agent helpers ----------
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

function agentIconType(name: string): string {
  if (name.includes('writer') || name.includes('code_w')) return 'code'
  if (name.includes('executor') || name.includes('code_e')) return 'terminal'
  if (name.includes('file')) return 'file'
  return 'people'
}

function copyContent() {
  const textParts: string[] = []
  if (props.message.content) {
    textParts.push(props.message.content)
  }
  if (props.message.events) {
    for (const evt of props.message.events) {
      if (evt.type === 'message' && evt.content) {
        textParts.push(evt.content)
      }
    }
  }
  navigator.clipboard.writeText(textParts.join('\n\n'))
}
</script>

<template>
  <div :class="['message-bubble', {
    'message-user': isUser,
    'message-assistant': isAssistant,
    'message-system': isSystem
  }]">
    <!-- System message -->
    <div v-if="isSystem" class="system-msg">
      <span class="system-text">{{ message.content }}</span>
    </div>

    <!-- User message -->
    <div v-else-if="isUser" class="user-bubble">
      <div class="bubble-content user-content">
        {{ message.content }}
      </div>
      <span class="msg-time">{{ timeStr }}</span>
    </div>

    <!-- Assistant message — segmented timeline view -->
    <div v-else class="assistant-bubble">
      <!-- Header -->
      <div class="assistant-header">
        <div class="assistant-avatar">
          <NIcon size="14"><HardwareChip /></NIcon>
        </div>
        <span class="msg-time">{{ timeStr }}</span>
        <button
          class="copy-btn"
          type="button"
          @click="copyContent"
          :title="t('message.copy')"
          :aria-label="t('message.copy')"
        >
          <NIcon size="12"><Clipboard /></NIcon>
        </button>
      </div>

      <!-- Legacy: show content if no events (e.g. restored from persistence) -->
      <div v-if="!hasEvents && message.content" class="bubble-content markdown-body" v-html="renderedContent"></div>

      <!-- Segmented timeline events -->
      <div v-if="hasEvents" class="timeline-container">
        <template v-for="seg in segments" :key="seg.id">
          <!-- Transfer separator -->
          <div v-if="seg.type === 'transfer'" class="transfer-divider">
            <span class="transfer-line"></span>
            <span class="transfer-label">
              <span :style="{ color: agentColor(seg.fromAgent || '') }">{{ agentLabel(seg.fromAgent || '') }}</span>
              <span class="transfer-arrow">&rarr;</span>
              <span :style="{ color: agentColor(seg.agent) }">{{ agentLabel(seg.agent) }}</span>
            </span>
            <span class="transfer-line"></span>
          </div>

          <!-- Sub-agent collapsible segment -->
          <div v-else-if="seg.type === 'subagent'" class="subagent-segment" :style="{ '--agent-color': agentColor(seg.agent) }">
            <div class="subagent-header" @click="toggleSubAgent(seg)">
              <div class="subagent-icon">
                <NIcon size="13">
                  <CodeSlash v-if="agentIconType(seg.agent) === 'code'" />
                  <Terminal v-else-if="agentIconType(seg.agent) === 'terminal'" />
                  <DocumentText v-else-if="agentIconType(seg.agent) === 'file'" />
                  <People v-else />
                </NIcon>
              </div>
              <span class="subagent-name">{{ agentLabel(seg.agent) }}</span>
              <span class="subagent-state-pill" :class="{ running: seg.isRunning, done: !seg.isRunning }">
                {{ segmentStatusLabel(seg) }}
              </span>
              <span class="subagent-stats">{{ segmentStats(seg) }}</span>
              <NIcon v-if="!seg.isRunning" size="12" class="subagent-status done"><CheckmarkCircle /></NIcon>
              <NIcon v-else size="12" class="subagent-status running"><Reload /></NIcon>
              <span class="subagent-chevron" :class="{ expanded: isSubAgentExpanded(seg) }">
                <NIcon size="11"><ChevronForward /></NIcon>
              </span>
            </div>

            <div v-if="seg.taskDescription" class="subagent-task-desc">
              {{ truncStr(seg.taskDescription, 120) }}
            </div>
            <div class="subagent-last-action" :class="{ muted: !seg.lastAction }">
              {{ seg.lastAction || t('message.subagent.noActions') }}
            </div>

            <!-- Collapsible body -->
            <transition name="expand">
              <div v-show="isSubAgentExpanded(seg)" class="subagent-body">
                <div class="segment-events">
                  <TimelineEventItem
                    v-for="evt in seg.events"
                    :key="evt.id"
                    :event="evt"
                    :show-agent-badge="false"
                  />
                </div>
              </div>
            </transition>
          </div>

          <!-- Main agent segment -->
          <div v-else class="agent-segment">
            <div class="segment-header" :style="{ '--seg-color': agentColor(seg.agent) }">
              <span class="segment-color-bar"></span>
              <span class="segment-agent-name">{{ agentLabel(seg.agent) }}</span>
            </div>
            <div class="segment-events">
              <TimelineEventItem
                v-for="evt in seg.events"
                :key="evt.id"
                :event="evt"
                :show-agent-badge="false"
              />
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-bubble {
  animation: fadeIn 250ms ease both;
}

.message-user {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.message-assistant {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.message-system {
  display: flex;
  justify-content: center;
  padding: 4px 0;
}

/* System */
.system-msg {
  background: rgba(244, 63, 94, 0.1);
  border: 1px solid rgba(244, 63, 94, 0.2);
  border-radius: var(--radius-md);
  padding: 8px 14px;
  max-width: 80%;
}

.system-text {
  color: var(--accent-rose);
  font-size: 12px;
}

/* User — de-bubbled: flat text with a 3px cyan bar on the right */
.user-bubble {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  max-width: 92%;
}

.user-content {
  background: transparent;
  color: var(--text-primary);
  border-right: 3px solid var(--accent-cyan);
  padding: 2px 14px 2px 16px;
  font-size: var(--fs-md);
  line-height: var(--lh-normal);
  white-space: pre-wrap;
  word-break: break-word;
  text-align: right;
}

/* Assistant */
.assistant-bubble {
  max-width: 100%;
  min-width: 200px;
  width: 100%;
}

.assistant-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.assistant-avatar {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  background: linear-gradient(135deg, var(--accent-cyan-dim), var(--accent-cyan));
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.bubble-content {
  font-size: 13.5px;
  line-height: 1.7;
}

.assistant-bubble > .bubble-content {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 4px var(--radius-lg) var(--radius-lg) var(--radius-lg);
  padding: 12px 16px;
}

.msg-time {
  font-size: 10px;
  color: var(--text-faint);
  margin-top: 4px;
  flex-shrink: 0;
}

.copy-btn {
  background: none;
  border: none;
  color: var(--text-faint);
  cursor: pointer;
  padding: 3px 6px;
  border-radius: 4px;
  transition: color var(--transition-ui), background var(--transition-ui), opacity var(--transition-ui);
  display: flex;
  align-items: center;
  margin-left: auto;
  opacity: 0.6;
}

.assistant-bubble:hover .copy-btn,
.copy-btn:focus-visible {
  opacity: 1;
}

.copy-btn:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
  opacity: 1;
}

/* Timeline container */
.timeline-container {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* Transfer divider */
.transfer-divider {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
}

.transfer-line {
  flex: 1;
  height: 1px;
  background: var(--border-subtle);
}

.transfer-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  font-family: var(--font-mono);
  white-space: nowrap;
}

.transfer-arrow {
  color: var(--text-faint);
  font-size: 13px;
}

/* Main agent segment */
.agent-segment {
  margin-bottom: 4px;
}

.segment-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0 2px;
}

.segment-color-bar {
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--seg-color);
  flex-shrink: 0;
}

.segment-agent-name {
  font-size: 11px;
  font-weight: 700;
  font-family: var(--font-mono);
  color: var(--seg-color);
  letter-spacing: 0.3px;
}

.segment-events {
  padding-left: 11px;
  border-left: 1px solid var(--border-subtle);
  margin-left: 1px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* ==================== Sub-agent collapsible segment ==================== */
.subagent-segment {
  background: color-mix(in srgb, var(--agent-color) 4%, var(--bg-deepest));
  border: 1px solid color-mix(in srgb, var(--agent-color) 15%, transparent);
  border-radius: var(--radius-md);
  margin: 6px 0;
  overflow: hidden;
  transition: border-color 200ms ease;
}

.subagent-segment:hover {
  border-color: color-mix(in srgb, var(--agent-color) 30%, transparent);
}

.subagent-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
  transition: background 150ms ease;
}

.subagent-header:hover {
  background: color-mix(in srgb, var(--agent-color) 6%, transparent);
}

.subagent-icon {
  width: 22px;
  height: 22px;
  border-radius: 5px;
  background: color-mix(in srgb, var(--agent-color) 15%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--agent-color);
  flex-shrink: 0;
}

.subagent-name {
  font-size: 12px;
  font-weight: 700;
  font-family: var(--font-mono);
  color: var(--agent-color);
  letter-spacing: 0.3px;
  flex-shrink: 0;
}

.subagent-state-pill {
  font-size: 10px;
  font-weight: 700;
  font-family: var(--font-mono);
  padding: 1px 6px;
  border-radius: 999px;
  border: 1px solid transparent;
  flex-shrink: 0;
}

.subagent-state-pill.running {
  color: var(--accent-violet);
  border-color: rgba(167, 139, 250, 0.35);
  background: rgba(167, 139, 250, 0.1);
}

.subagent-state-pill.done {
  color: var(--accent-emerald);
  border-color: rgba(16, 185, 129, 0.35);
  background: rgba(16, 185, 129, 0.1);
}

.subagent-stats {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  margin-left: auto;
}

.subagent-status.done {
  color: var(--accent-emerald);
  flex-shrink: 0;
}

.subagent-status.running {
  color: var(--text-faint);
  animation: spin 1.5s linear infinite;
  flex-shrink: 0;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.subagent-chevron {
  color: var(--text-faint);
  flex-shrink: 0;
  transition: transform 200ms ease;
  display: flex;
  align-items: center;
}

.subagent-chevron.expanded {
  transform: rotate(90deg);
}

.subagent-task-desc {
  padding: 0 12px 6px 42px;
  font-size: 11.5px;
  line-height: 1.5;
  color: var(--text-secondary);
  overflow-wrap: break-word;
}

.subagent-last-action {
  padding: 0 12px 8px 42px;
  font-size: 11px;
  line-height: 1.4;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.subagent-last-action.muted {
  color: var(--text-faint);
}

.subagent-body {
  border-top: 1px solid color-mix(in srgb, var(--agent-color) 10%, transparent);
  padding: 8px 12px 10px;
}

.subagent-body .segment-events {
  padding-left: 8px;
  border-left-color: color-mix(in srgb, var(--agent-color) 20%, transparent);
}

/* Expand/collapse transition */
.expand-enter-active,
.expand-leave-active {
  transition: all 200ms ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}
</style>
