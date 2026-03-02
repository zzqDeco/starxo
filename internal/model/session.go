package model

// Session represents a persisted chat session.
type Session struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Containers        []string `json:"containers"`                  // container registry IDs owned by this session
	ActiveContainerID string   `json:"activeContainerID,omitempty"` // currently active container registry ID
	WorkspacePath     string   `json:"workspacePath,omitempty"`
	CreatedAt         int64    `json:"createdAt"`
	UpdatedAt         int64    `json:"updatedAt"`
	MessageCount      int      `json:"messageCount"`
}

// HasContainer returns true if the session owns the given container registry ID.
func (s *Session) HasContainer(containerID string) bool {
	for _, id := range s.Containers {
		if id == containerID {
			return true
		}
	}
	return false
}

// AddContainer adds a container registry ID to the session's list (no duplicates).
func (s *Session) AddContainer(containerID string) {
	if !s.HasContainer(containerID) {
		s.Containers = append(s.Containers, containerID)
	}
}

// RemoveContainer removes a container registry ID from the session's list.
func (s *Session) RemoveContainer(containerID string) {
	for i, id := range s.Containers {
		if id == containerID {
			s.Containers = append(s.Containers[:i], s.Containers[i+1:]...)
			if s.ActiveContainerID == containerID {
				s.ActiveContainerID = ""
			}
			return
		}
	}
}
