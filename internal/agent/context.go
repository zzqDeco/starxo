package agent

// AgentContext provides runtime environment information that is injected
// into agent prompts and tool defaults. This removes all hard-coded
// "/workspace" references and makes agents aware of their container.
type AgentContext struct {
	WorkspacePath string // e.g. "/workspace"
	ContainerName string // e.g. "eino-sandbox-abc123"
	ContainerID   string // short Docker container ID
	SSHHost       string // e.g. "192.168.1.100"
	SSHPort       int    // e.g. 22
	SSHUser       string // e.g. "root"

	// OnToolEvent is called by sub-agent tool wrappers to emit timeline events.
	// Parameters: agentName, eventType ("tool_call"/"tool_result"), toolName, toolArgs, toolID, result.
	OnToolEvent func(agentName, eventType, toolName, toolArgs, toolID, result string)
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
