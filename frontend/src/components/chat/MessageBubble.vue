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

interface EventSegment {
  agent: string
  events: TurnEvent[]
  isTransfer: boolean
  isSubAgent: boolean
  taskDescription?: string
  isCompleted: boolean
}

function tryParseArgs(args?: string): any {
  if (!args) return null
  try { return JSON.parse(args) } catch { return null }
}

function truncStr(s: string, max: number): string {
  return s.length <= max ? s : s.substring(0, max) + '...'
}

function buildSegment(agent: string, events: TurnEvent[]): EventSegment {
  const isSubAgent = !mainAgents.has(agent)
  const isCompleted = events.every(evt => {
    if (evt.type === 'tool_call') return !!evt.toolResult
    if (evt.type === 'message' && evt.isStreaming) return false
    return true
  })
  return { agent, events, isTransfer: false, isSubAgent, isCompleted }
}

const segments = computed<EventSegment[]>(() => {
  if (!props.message.events || props.message.events.length === 0) return []

  const segs: EventSegment[] = []
  let currentAgent = ''
  let currentEvents: TurnEvent[] = []

  for (const evt of props.message.events) {
    if (evt.type === 'transfer') {
      if (currentEvents.length > 0) {
        segs.push(buildSegment(currentAgent, [...currentEvents]))
        currentEvents = []
      }
      segs.push({ agent: evt.agent, events: [evt], isTransfer: true, isSubAgent: false, isCompleted: true })
      currentAgent = evt.content
    } else {
      const evtAgent = evt.agent || ''
      if (evtAgent !== currentAgent && currentEvents.length > 0) {
        segs.push(buildSegment(currentAgent, [...currentEvents]))
        currentEvents = []
      }
      currentAgent = evtAgent
      currentEvents.push(evt)
    }
  }
  if (currentEvents.length > 0) {
    segs.push(buildSegment(currentAgent, currentEvents))
  }

  // Post-process: extract task descriptions for sub-agent segments
  for (let i = 0; i < segs.length; i++) {
    if (segs[i].isSubAgent) {
      for (let j = i - 1; j >= 0; j--) {
        if (segs[j].isTransfer) continue
        for (let k = segs[j].events.length - 1; k >= 0; k--) {
          const ev = segs[j].events[k]
          if (ev.type === 'tool_call' && ev.toolName === 'task') {
            const args = tryParseArgs(ev.toolArgs)
            if (args?.description) {
              segs[i].taskDescription = args.description
            }
            break
          }
        }
        break
      }
    }
  }

  return segs
})

// ---------- Sub-agent expand/collapse ----------
const subAgentToggled = ref<Record<number, boolean>>({})

function isSubAgentExpanded(index: number): boolean {
  if (index in subAgentToggled.value) {
    return subAgentToggled.value[index]
  }
  // Default: expanded if active, collapsed if completed
  const seg = segments.value[index]
  return seg ? !seg.isCompleted : false
}

function toggleSubAgent(index: number) {
  subAgentToggled.value = { ...subAgentToggled.value, [index]: !isSubAgentExpanded(index) }
}

function segmentStats(seg: EventSegment): string {
  const toolCalls = seg.events.filter(e => e.type === 'tool_call').length
  const messages = seg.events.filter(e => e.type === 'message').length
  const parts: string[] = []
  if (toolCalls > 0) parts.push(`${toolCalls} tools`)
  if (messages > 0) parts.push(`${messages} msgs`)
  return parts.join(' · ') || `${seg.events.length} events`
}

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
        <button class="copy-btn" @click="copyContent" :title="t('message.copy')">
          <NIcon size="12"><Clipboard /></NIcon>
        </button>
      </div>

      <!-- Legacy: show content if no events (e.g. restored from persistence) -->
      <div v-if="!hasEvents && message.content" class="bubble-content markdown-body" v-html="renderedContent"></div>

      <!-- Segmented timeline events -->
      <div v-if="hasEvents" class="timeline-container">
        <template v-for="(seg, si) in segments" :key="si">
          <!-- Transfer separator -->
          <div v-if="seg.isTransfer" class="transfer-divider">
            <span class="transfer-line"></span>
            <span class="transfer-label">
              <span :style="{ color: agentColor(seg.events[0].agent) }">{{ agentLabel(seg.events[0].agent) }}</span>
              <span class="transfer-arrow">&rarr;</span>
              <span :style="{ color: agentColor(seg.events[0].content) }">{{ agentLabel(seg.events[0].content) }}</span>
            </span>
            <span class="transfer-line"></span>
          </div>

          <!-- Sub-agent collapsible segment -->
          <div v-else-if="seg.isSubAgent" class="subagent-segment" :style="{ '--agent-color': agentColor(seg.agent) }">
            <div class="subagent-header" @click="toggleSubAgent(si)">
              <div class="subagent-icon">
                <NIcon size="13">
                  <CodeSlash v-if="agentIconType(seg.agent) === 'code'" />
                  <Terminal v-else-if="agentIconType(seg.agent) === 'terminal'" />
                  <DocumentText v-else-if="agentIconType(seg.agent) === 'file'" />
                  <People v-else />
                </NIcon>
              </div>
              <span class="subagent-name">{{ agentLabel(seg.agent) }}</span>
              <span class="subagent-stats">{{ segmentStats(seg) }}</span>
              <NIcon v-if="seg.isCompleted" size="12" class="subagent-status done"><CheckmarkCircle /></NIcon>
              <NIcon v-else size="12" class="subagent-status running"><Reload /></NIcon>
              <span class="subagent-chevron" :class="{ expanded: isSubAgentExpanded(si) }">
                <NIcon size="11"><ChevronForward /></NIcon>
              </span>
            </div>
            <div v-if="seg.taskDescription" class="subagent-task-desc">
              {{ truncStr(seg.taskDescription, 120) }}
            </div>
            <!-- Collapsible body -->
            <transition name="expand">
              <div v-show="isSubAgentExpanded(si)" class="subagent-body">
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

/* User */
.user-bubble {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  max-width: 75%;
}

.user-content {
  background: linear-gradient(135deg, #1a2744, #1e3a5f);
  color: #e0eaff;
  border-radius: var(--radius-lg) var(--radius-lg) 4px var(--radius-lg);
  padding: 10px 16px;
  font-size: 13.5px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  box-shadow: 0 2px 8px rgba(30, 58, 95, 0.3);
}

/* Assistant */
.assistant-bubble {
  max-width: 90%;
  min-width: 200px;
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
  padding: 2px 4px;
  border-radius: 4px;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  margin-left: auto;
}

.copy-btn:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
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

.subagent-stats {
  font-size: 10px;
  color: var(--text-faint);
  font-family: var(--font-mono);
  margin-left: 4px;
}

.subagent-status.done {
  color: var(--accent-emerald);
  flex-shrink: 0;
  margin-left: auto;
}

.subagent-status.running {
  color: var(--text-faint);
  animation: spin 1.5s linear infinite;
  flex-shrink: 0;
  margin-left: auto;
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
  padding: 0 12px 8px 42px;
  font-size: 11.5px;
  line-height: 1.5;
  color: var(--text-secondary);
  overflow-wrap: break-word;
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
