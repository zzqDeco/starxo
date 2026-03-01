import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Message, TurnEvent, InterruptEvent, PlanStepDTO } from '@/types/message'

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
          return
        }
      }
    }

    turn.events.push(evt)
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
    addMessage,
    addUserMessage,
    addTimelineEvent,
    setInterrupt,
    clearInterrupt,
    updatePlanSteps,
    setMode,
    setGenerating,
    clearMessages
  }
})
