package service

import "starxo/internal/model"

// MessageEvent is sent to the frontend when a complete message is available.
type MessageEvent struct {
	ID        string `json:"id"`
	Agent     string `json:"agent"`
	Content   string `json:"content"`
	Role      string `json:"role"`
	Timestamp int64  `json:"timestamp"`
}

// StreamChunkEvent is sent to the frontend during streaming output.
type StreamChunkEvent struct {
	Agent   string `json:"agent"`
	Content string `json:"content"`
	Role    string `json:"role"`
}

// AgentActionEvent is sent when the agent performs an action (e.g. tool call).
type AgentActionEvent struct {
	Type      string `json:"type"`
	AgentName string `json:"agentName"`
	Details   string `json:"details"`
	ToolID    string `json:"toolId,omitempty"`
}

// ToolResultEvent is sent when a tool call completes with a result.
type ToolResultEvent struct {
	AgentName  string `json:"agentName"`
	ToolCallID string `json:"toolCallId"`
	Content    string `json:"content"`
}

// TerminalOutputEvent is sent when a command execution produces output.
type TerminalOutputEvent struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

// SandboxProgressEvent is sent during sandbox connection setup.
type SandboxProgressEvent struct {
	Step    string `json:"step"`
	Percent int    `json:"percent"`
}

// FileInfoDTO is the file information data transfer object for the frontend.
type FileInfoDTO struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsOutput bool   `json:"isOutput"`
}

// SandboxStatusDTO represents the current sandbox connection status.
type SandboxStatusDTO struct {
	SSHConnected        bool   `json:"sshConnected"`
	DockerRunning       bool   `json:"dockerRunning"`
	ContainerID         string `json:"containerID"`
	ActiveContainerID   string `json:"activeContainerID"`
	ActiveContainerName string `json:"activeContainerName"`
	DockerAvailable     bool   `json:"dockerAvailable"`
}

// SessionSwitchedEvent is emitted when the active session changes.
type SessionSwitchedEvent struct {
	Session      model.Session   `json:"session"`
	ContainerID  string          `json:"containerID,omitempty"`
	AgentRunning bool            `json:"agentRunning"`
	CurrentAgent string          `json:"currentAgent,omitempty"`
	Mode         string          `json:"mode"`
	HasInterrupt bool            `json:"hasInterrupt"`
	Interrupt    *InterruptEvent `json:"interrupt,omitempty"`
}

// TimelineEvent is the unified event type for the chat timeline.
// Every meaningful action (message, tool call, transfer, etc.) is sent as a TimelineEvent.
type TimelineEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // "message" | "tool_call" | "tool_result" | "transfer" | "info" | "interrupt" | "reasoning" | "thinking"
	Agent     string `json:"agent"`
	Content   string `json:"content"`
	ToolName  string `json:"toolName,omitempty"`
	ToolArgs  string `json:"toolArgs,omitempty"`
	ToolID    string `json:"toolId,omitempty"`
	Timestamp int64  `json:"timestamp"`
	SessionID string `json:"sessionId,omitempty"`
}

// InterruptEvent is emitted when the agent interrupts execution to ask the user.
type InterruptEvent struct {
	Type         string            `json:"type"`         // "followup" | "choice"
	InterruptID  string            `json:"interruptId"`  // used for ResumeWithParams
	CheckpointID string            `json:"checkpointId"` // used for ResumeWithParams
	Questions    []string          `json:"questions,omitempty"`
	Options      []InterruptOption `json:"options,omitempty"`
	Question     string            `json:"question,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
}

// InterruptOption represents a single choice option in an interrupt.
type InterruptOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// PlanEvent is emitted when the plan state changes.
type PlanEvent struct {
	Steps []PlanStepDTO `json:"steps"`
}

// PlanStepDTO represents a single plan step for the frontend.
type PlanStepDTO struct {
	TaskID     int    `json:"taskId"`
	Status     string `json:"status"` // "todo" | "doing" | "done" | "failed" | "skipped"
	Desc       string `json:"desc"`
	ExecResult string `json:"execResult,omitempty"`
}

// ModeChangedEvent is emitted when the agent mode switches.
type ModeChangedEvent struct {
	Mode      string `json:"mode"` // "default" | "plan"
	SessionID string `json:"sessionId,omitempty"`
}
