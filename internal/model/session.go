package model

// Session represents a persisted chat session.
type Session struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	ContainerID   string `json:"containerID"`
	WorkspacePath string `json:"workspacePath,omitempty"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
	MessageCount  int    `json:"messageCount"`
}
