package agent

import "context"

// AgentContext provides runtime environment information that is injected
// into agent prompts and tool defaults. This removes all hard-coded
// "/workspace" references and makes agents aware of their sandbox.
type AgentContext struct {
	WorkspacePath string // e.g. "/workspace"
	ContainerName string // compatibility field; sandbox name
	ContainerID   string // compatibility field; sandbox runtime ID
	SSHHost       string // e.g. "192.168.1.100"
	SSHPort       int    // e.g. 22
	SSHUser       string // e.g. "root"

	// OnToolEvent is called by sub-agent tool wrappers to emit timeline events.
	// The ctx carries session identity (use SessionIDFromContext to extract).
	// Parameters: ctx, agentName, eventType ("tool_call"/"tool_result"), toolName, toolArgs, toolID, result.
	OnToolEvent func(ctx context.Context, agentName, eventType, toolName, toolArgs, toolID, result string)
}

// DefaultAgentContext returns a fallback context when no session binding exists.
func DefaultAgentContext() AgentContext {
	return AgentContext{
		WorkspacePath: "/workspace",
		ContainerName: "unknown",
		ContainerID:   "",
		SSHHost:       "",
		SSHPort:       22,
		SSHUser:       "root",
	}
}
