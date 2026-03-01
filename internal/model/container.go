package model

// ContainerStatus tracks the known state of a container.
type ContainerStatus string

const (
	ContainerRunning   ContainerStatus = "running"
	ContainerStopped   ContainerStatus = "stopped"
	ContainerUnknown   ContainerStatus = "unknown"
	ContainerDestroyed ContainerStatus = "destroyed"
)

// Container represents a persisted container registry entry.
type Container struct {
	ID            string          `json:"id"`
	DockerID      string          `json:"dockerID"`
	Name          string          `json:"name"`
	Image         string          `json:"image"`
	SSHHost       string          `json:"sshHost"`
	SSHPort       int             `json:"sshPort"`
	Status        ContainerStatus `json:"status"`
	SetupComplete bool            `json:"setupComplete"`
	CreatedAt     int64           `json:"createdAt"`
	LastUsedAt    int64           `json:"lastUsedAt"`
}
