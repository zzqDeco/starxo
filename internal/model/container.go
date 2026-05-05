package model

// ContainerStatus tracks the known state of a container.
type ContainerStatus string

const (
	ContainerRunning     ContainerStatus = "running"
	ContainerStopped     ContainerStatus = "stopped"
	ContainerUnknown     ContainerStatus = "unknown"
	ContainerDestroyed   ContainerStatus = "destroyed"
	ContainerUnavailable ContainerStatus = "unavailable"
)

// Container represents a persisted sandbox registry entry. The type name and
// DockerID field are retained for compatibility with existing Wails bindings
// and legacy containers.json files.
type Container struct {
	ID            string          `json:"id"`
	RuntimeID     string          `json:"runtimeID,omitempty"`
	Runtime       string          `json:"runtime,omitempty"`
	WorkspacePath string          `json:"workspacePath,omitempty"`
	DockerID      string          `json:"dockerID,omitempty"`
	Name          string          `json:"name"`
	Image         string          `json:"image,omitempty"`
	SSHHost       string          `json:"sshHost"`
	SSHPort       int             `json:"sshPort"`
	Status        ContainerStatus `json:"status"`
	SetupComplete bool            `json:"setupComplete"`
	SessionID     string          `json:"sessionID"` // owning session ID
	CreatedAt     int64           `json:"createdAt"`
	LastUsedAt    int64           `json:"lastUsedAt"`
}
