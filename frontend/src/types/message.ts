export interface TurnEvent {
  id: string
  type: 'message' | 'tool_call' | 'tool_result' | 'transfer' | 'info' | 'interrupt' | 'plan' | 'stream_chunk' | 'stream_end' | 'reasoning' | 'thinking'
  agent: string
  content: string
  toolName?: string
  toolArgs?: string
  toolId?: string
  toolResult?: string
  timestamp: number
  isStreaming?: boolean
}

export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  agent?: string
  timestamp: number
  isStreaming?: boolean
  events: TurnEvent[]
}

export interface TerminalOutputEvent {
  stdout: string
  stderr: string
  exitCode: number
}

export interface PersistedMessage {
  role: string
  content: string
  name?: string
  toolCallId?: string
}

// Interrupt event types
export interface InterruptEvent {
  type: 'followup' | 'choice'
  interruptId: string
  checkpointId: string
  questions?: string[]
  options?: InterruptOption[]
  question?: string
}

export interface InterruptOption {
  label: string
  description: string
}

// Plan event types
export interface PlanEvent {
  steps: PlanStepDTO[]
}

export interface PlanStepDTO {
  taskId: number
  status: 'todo' | 'doing' | 'done' | 'failed' | 'skipped'
  desc: string
  execResult?: string
}

// Mode changed event
export interface ModeChangedEvent {
  mode: 'default' | 'plan'
}
