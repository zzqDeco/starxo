import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Message, TurnEvent, InterruptEvent, PlanStepDTO } from '@/types/message'

export interface TodoItem {
  id: string
  title: string
  status: 'pending' | 'in_progress' | 'done' | 'failed' | 'blocked'
  depends_on?: string[]
}

export type UnifiedTaskStatus = 'todo' | 'doing' | 'done' | 'blocked'

export interface UnifiedTaskItem {
  id: string
  title: string
  status: UnifiedTaskStatus
  detail?: string
  source: 'todo' | 'plan'
}

function normalizeTaskStatus(status: string): UnifiedTaskStatus {
  if (status === 'doing' || status === 'in_progress') return 'doing'
  if (status === 'done' || status === 'skipped') return 'done'
  if (status === 'failed' || status === 'blocked') return 'blocked'
  return 'todo'
}

export const useChatStore = defineStore('chat', () => {
  const messages = ref<Message[]>([])
  const isStreaming = ref(false)
  const currentAgent = ref('')
  const agentDone = ref(false)
  const activeTurnId = ref<string | null>(null)

  // Interrupt state
  const pendingInterrupt = ref<InterruptEvent | null>(null)

  // Plan mode state
  const agentMode = ref<'default' | 'plan'>('default')
  const planSteps = ref<PlanStepDTO[]>([])

  // Persistent todo state (latest snapshot)
  const latestTodos = ref<TodoItem[]>([])

  const unifiedTasks = computed<UnifiedTaskItem[]>(() => {
    if (latestTodos.value.length > 0) {
      return latestTodos.value.map((todo) => ({
        id: todo.id,
        title: todo.title,
        status: normalizeTaskStatus(todo.status),
        source: 'todo'
      }))
    }

    return planSteps.value.map((step) => ({
      id: String(step.taskId),
      title: step.desc,
      status: normalizeTaskStatus(step.status),
      detail: step.execResult,
      source: 'plan'
    }))
  })

  const unifiedTaskStats = computed(() => {
    const counts: Record<UnifiedTaskStatus, number> = {
      todo: 0,
      doing: 0,
      done: 0,
      blocked: 0
    }

    for (const task of unifiedTasks.value) {
      counts[task.status] += 1
    }

    const total = unifiedTasks.value.length
    const done = counts.done
    const progressPercent = total > 0 ? Math.round((done / total) * 100) : 0
    const currentTask = unifiedTasks.value.find((task) => task.status === 'doing')
      || unifiedTasks.value.find((task) => task.status === 'todo')
      || unifiedTasks.value.find((task) => task.status === 'blocked')
      || unifiedTasks.value[unifiedTasks.value.length - 1]

    return {
      counts,
      total,
      done,
      progressPercent,
      currentTask
    }
  })

  const lastMessage = computed(() =>
    messages.value.length > 0 ? messages.value[messages.value.length - 1] : null
  )

  // Filter out empty assistant messages (no content and no events)
  const visibleMessages = computed(() =>
    messages.value.filter(m => {
      if (m.role === 'assistant' && !m.content && (!m.events || m.events.length === 0)) {
        return false
      }
      return true
    })
  )

  const hasInterrupt = computed(() => pendingInterrupt.value !== null)

  /** Get or create the active assistant turn message */
  function getOrCreateTurn(): Message {
    if (activeTurnId.value) {
      const existing = messages.value.find(m => m.id === activeTurnId.value)
      if (existing) return existing
    }
    const msg: Message = {
      id: crypto.randomUUID(),
      role: 'assistant',
      content: '',
      agent: currentAgent.value || 'coding_agent',
      timestamp: Date.now(),
      events: []
    }
    messages.value.push(msg)
    activeTurnId.value = msg.id
    return msg
  }

  function addMessage(message: Message) {
    if (!message.events) message.events = []
    messages.value.push(message)
  }

  function addUserMessage(content: string) {
    const msg: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
      timestamp: Date.now(),
      events: []
    }
    messages.value.push(msg)
    activeTurnId.value = null
    agentDone.value = false
    return msg
  }

  /** Add a timeline event to the current assistant turn */
  function addTimelineEvent(evt: TurnEvent) {
    const turn = getOrCreateTurn()

    // --- thinking event management ---
    // When a thinking event arrives: replace any previous thinking from the same agent
    if (evt.type === 'thinking') {
      const idx = turn.events.findLastIndex(
        (e: TurnEvent) => e.type === 'thinking' && e.agent === evt.agent
      )
      if (idx !== -1) turn.events.splice(idx, 1)
      turn.events.push(evt)
      return
    }
    // When any non-thinking event arrives: clear thinking from the same agent
    const thinkIdx = turn.events.findLastIndex(
      (e: TurnEvent) => e.type === 'thinking' && e.agent === evt.agent
    )
    if (thinkIdx !== -1) turn.events.splice(thinkIdx, 1)

    // Stream chunk: accumulate into existing streaming message or create new one
    if (evt.type === 'stream_chunk') {
      const lastEvt = turn.events.length > 0 ? turn.events[turn.events.length - 1] : null
      if (lastEvt && lastEvt.type === 'message' && lastEvt.isStreaming && lastEvt.agent === evt.agent) {
        // Append to existing streaming message
        lastEvt.content += evt.content
        return
      }
      // Create new streaming message event
      turn.events.push({
        id: evt.id,
        type: 'message',
        agent: evt.agent,
        content: evt.content,
        timestamp: evt.timestamp,
        isStreaming: true
      })
      return
    }

    // Stream end: finalize the streaming message
    if (evt.type === 'stream_end') {
      for (let i = turn.events.length - 1; i >= 0; i--) {
        if (turn.events[i].type === 'message' && turn.events[i].isStreaming && turn.events[i].agent === evt.agent) {
          turn.events[i].isStreaming = false
          break
        }
      }
      return
    }

    // For tool_result, find matching tool_call and attach result
    if (evt.type === 'tool_result' && evt.toolId) {
      for (let i = turn.events.length - 1; i >= 0; i--) {
        if (turn.events[i].type === 'tool_call' && turn.events[i].toolId === evt.toolId) {
          turn.events[i].toolResult = evt.content
          tryUpdateTodosFromResult(evt)
          return
        }
      }
    }

    // Extract latest todo state from todo tool events
    if (evt.type === 'tool_call' && evt.toolName === 'write_todos') {
      try {
        const args = JSON.parse(evt.toolArgs || '{}')
        if (args?.todos && Array.isArray(args.todos)) {
          latestTodos.value = args.todos
        }
      } catch { /* ignore parse errors */ }
    }

    turn.events.push(evt)
  }

  /** Update latestTodos when a tool_result for update_todo arrives */
  function tryUpdateTodosFromResult(evt: TurnEvent) {
    if (evt.type !== 'tool_result' || !evt.toolId) return
    const turn = getOrCreateTurn()
    for (let i = turn.events.length - 1; i >= 0; i--) {
      if (turn.events[i].toolId === evt.toolId && turn.events[i].toolName === 'update_todo') {
        const parts = evt.content.split('---\n')
        if (parts.length >= 2) {
          try {
            const todos = JSON.parse(parts[parts.length - 1])
            if (Array.isArray(todos)) latestTodos.value = todos
          } catch { /* ignore */ }
        }
        break
      }
    }
  }

  /** Set interrupt state from agent:interrupt event */
  function setInterrupt(evt: InterruptEvent) {
    pendingInterrupt.value = evt
    isStreaming.value = false
  }

  /** Clear interrupt state after user responds */
  function clearInterrupt() {
    pendingInterrupt.value = null
  }

  /** Update plan steps from agent:plan event */
  function updatePlanSteps(steps: PlanStepDTO[]) {
    planSteps.value = steps
  }

  /** Set agent mode */
  function setMode(mode: 'default' | 'plan') {
    agentMode.value = mode
  }

  function setGenerating(generating: boolean, agent?: string) {
    if (generating) {
      agentDone.value = false
    } else {
      agentDone.value = true
      activeTurnId.value = null
    }
    isStreaming.value = generating
    currentAgent.value = generating ? (agent || '') : ''
  }

  function clearMessages() {
    messages.value = []
    isStreaming.value = false
    currentAgent.value = ''
    agentDone.value = false
    activeTurnId.value = null
    pendingInterrupt.value = null
    planSteps.value = []
    latestTodos.value = []
  }

  /** Scan all restored messages to extract the latest todos (for session restore) */
  function restoreTodosFromMessages() {
    for (const msg of messages.value) {
      if (!msg.events) continue
      for (const evt of msg.events) {
        if (evt.type === 'tool_call' && evt.toolName === 'write_todos') {
          try {
            const args = JSON.parse(evt.toolArgs || '{}')
            if (args?.todos && Array.isArray(args.todos)) {
              latestTodos.value = args.todos
            }
          } catch { /* ignore */ }
        }
        // update_todo: check toolResult for updated list
        if (evt.type === 'tool_call' && evt.toolName === 'update_todo' && evt.toolResult) {
          const parts = evt.toolResult.split('---\n')
          if (parts.length >= 2) {
            try {
              const todos = JSON.parse(parts[parts.length - 1])
              if (Array.isArray(todos)) latestTodos.value = todos
            } catch { /* ignore */ }
          }
        }
      }
    }
  }

  return {
    messages,
    isStreaming,
    currentAgent,
    agentDone,
    activeTurnId,
    pendingInterrupt,
    agentMode,
    planSteps,
    lastMessage,
    visibleMessages,
    hasInterrupt,
    latestTodos,
    unifiedTasks,
    unifiedTaskStats,
    addMessage,
    addUserMessage,
    addTimelineEvent,
    setInterrupt,
    clearInterrupt,
    updatePlanSteps,
    setMode,
    setGenerating,
    clearMessages,
    restoreTodosFromMessages
  }
})
