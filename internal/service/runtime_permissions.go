package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"starxo/internal/model"
	"starxo/internal/tools"
)

const (
	permissionRiskWrite       = "write"
	permissionRiskExecute     = "execute"
	permissionRiskDestructive = "destructive"
	permissionRequestTimeout  = 10 * time.Minute
)

type runtimePermissionRequest struct {
	request tools.ToolPermissionRequest
	entry   tools.CatalogEntry
	result  chan tools.ToolPermissionResolution
}

func (p *deferredMCPProvider) RequestToolPermission(ctx context.Context, entry tools.CatalogEntry, argumentsInJSON string) (tools.ToolPermissionResolution, error) {
	sessionID := SessionIDFromContext(ctx)
	if sessionID == "" {
		sessionID = sessionIDFromGenericContext(ctx)
	}
	mode := model.ModeDefault
	if sessionID != "" {
		p.chat.mu.Lock()
		run := p.chat.sessions[sessionID]
		p.chat.mu.Unlock()
		if run != nil {
			run.stateMu.RLock()
			mode = run.mode
			run.stateMu.RUnlock()
		}
	}
	if override, ok := runtimeModeOverrideFromContext(ctx); ok {
		mode = override
	}
	return p.chat.requestToolPermission(ctx, sessionID, mode, entry, argumentsInJSON)
}

func (s *ChatService) requestToolPermission(ctx context.Context, sessionID, mode string, entry tools.CatalogEntry, argumentsInJSON string) (tools.ToolPermissionResolution, error) {
	if entry.ReadOnlyEligible() || mode == "bypassPermissions" {
		return tools.ToolPermissionResolution{Decision: tools.ToolPermissionDecisionAllowOnce}, nil
	}
	if sessionID != "" && s.hasPermissionGrant(sessionID, entry.CanonicalName) {
		return tools.ToolPermissionResolution{Decision: tools.ToolPermissionDecisionAllowSession}, nil
	}
	if s.ctx == nil {
		return tools.ToolPermissionResolution{}, fmt.Errorf("tool %s requires permission but no UI context is available", entry.CanonicalName)
	}

	requestID := fmt.Sprintf("perm-%d", s.now().UnixNano())
	request := tools.ToolPermissionRequest{
		RequestID:   requestID,
		SessionID:   sessionID,
		ToolName:    entry.CanonicalName,
		Title:       runtimeFirstNonEmpty(entry.Title, entry.CanonicalName),
		Description: entry.Description,
		ToolClass:   entry.ToolClass,
		Source:      entry.Source,
		Risk:        permissionRiskForEntry(entry),
		Input:       truncatePermissionInput(argumentsInJSON),
		CreatedAt:   s.now().UnixMilli(),
	}
	pending := &runtimePermissionRequest{
		request: request,
		entry:   entry,
		result:  make(chan tools.ToolPermissionResolution, 1),
	}
	s.permissionMu.Lock()
	s.permissionRequests[requestID] = pending
	s.permissionMu.Unlock()
	wailsEmit(s.ctx, "runtime:permission_request", request)

	waitCtx := ctx
	if _, ok := waitCtx.Deadline(); !ok {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, permissionRequestTimeout)
		defer cancel()
	}
	select {
	case resolution := <-pending.result:
		s.removePermissionRequest(requestID)
		resolution.Decision = normalizePermissionDecision(resolution.Decision)
		if resolution.Decision == tools.ToolPermissionDecisionAllowSession && sessionID != "" {
			s.addPermissionGrant(sessionID, entry)
		}
		return resolution, nil
	case <-waitCtx.Done():
		s.removePermissionRequest(requestID)
		wailsEmit(s.ctx, "runtime:permission_canceled", map[string]string{"requestId": requestID})
		return tools.ToolPermissionResolution{}, fmt.Errorf("tool %s permission request canceled: %w", entry.CanonicalName, waitCtx.Err())
	}
}

func (s *ChatService) resolvePermissionRequest(requestID, decision string) error {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return fmt.Errorf("requestID is required")
	}
	resolution := tools.ToolPermissionResolution{Decision: normalizePermissionDecision(decision)}
	s.permissionMu.Lock()
	pending, ok := s.permissionRequests[requestID]
	if ok {
		delete(s.permissionRequests, requestID)
	}
	s.permissionMu.Unlock()
	if !ok {
		return fmt.Errorf("permission request %s not found", requestID)
	}
	select {
	case pending.result <- resolution:
	default:
	}
	wailsEmit(s.ctx, "runtime:permission_resolved", map[string]string{
		"requestId": requestID,
		"decision":  resolution.Decision,
	})
	return nil
}

func (s *ChatService) removePermissionRequest(requestID string) {
	s.permissionMu.Lock()
	delete(s.permissionRequests, requestID)
	s.permissionMu.Unlock()
}

func (s *ChatService) hasPermissionGrant(sessionID, toolName string) bool {
	s.mu.Lock()
	run := s.sessions[sessionID]
	s.mu.Unlock()
	if run == nil {
		return false
	}
	run.stateMu.RLock()
	defer run.stateMu.RUnlock()
	grant, ok := run.permissionGrants[toolName]
	return ok && grant.Decision == tools.ToolPermissionDecisionAllowSession
}

func (s *ChatService) addPermissionGrant(sessionID string, entry tools.CatalogEntry) {
	s.mu.Lock()
	run := s.getOrCreateRun(sessionID)
	s.mu.Unlock()
	run.stateMu.Lock()
	if run.permissionGrants == nil {
		run.permissionGrants = make(map[string]model.RuntimePermissionGrant)
	}
	run.permissionGrants[entry.CanonicalName] = model.RuntimePermissionGrant{
		ToolName:  entry.CanonicalName,
		ToolClass: entry.ToolClass,
		Source:    entry.Source,
		Decision:  tools.ToolPermissionDecisionAllowSession,
		CreatedAt: s.now().UnixMilli(),
	}
	run.stateMu.Unlock()
	wailsEmit(s.ctx, "runtime:permission_grants_changed", map[string]string{"sessionId": sessionID})
	s.saveSessionPermissionState(sessionID)
}

func (s *ChatService) saveSessionPermissionState(sessionID string) {
	s.mu.Lock()
	ss := s.sessionService
	s.mu.Unlock()
	if ss != nil {
		go func() { _ = ss.SaveSessionByID(sessionID) }()
	}
}

func (s *ChatService) ListToolPermissionRequests(sessionID string) ([]tools.ToolPermissionRequest, error) {
	sessionID = strings.TrimSpace(sessionID)
	s.permissionMu.Lock()
	defer s.permissionMu.Unlock()
	requests := make([]tools.ToolPermissionRequest, 0, len(s.permissionRequests))
	for _, pending := range s.permissionRequests {
		if pending == nil {
			continue
		}
		request := pending.request
		if sessionID != "" && request.SessionID != "" && request.SessionID != sessionID {
			continue
		}
		requests = append(requests, request)
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].CreatedAt == requests[j].CreatedAt {
			return requests[i].RequestID < requests[j].RequestID
		}
		return requests[i].CreatedAt < requests[j].CreatedAt
	})
	return requests, nil
}

func (s *ChatService) ListToolPermissionGrants(sessionID string) ([]model.RuntimePermissionGrant, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID is required")
	}
	s.mu.Lock()
	run := s.sessions[sessionID]
	s.mu.Unlock()
	if run == nil {
		return nil, nil
	}
	run.stateMu.RLock()
	defer run.stateMu.RUnlock()
	grants := make([]model.RuntimePermissionGrant, 0, len(run.permissionGrants))
	for _, grant := range run.permissionGrants {
		if grant.Decision == tools.ToolPermissionDecisionAllowSession {
			grants = append(grants, grant)
		}
	}
	sort.Slice(grants, func(i, j int) bool {
		if grants[i].CreatedAt == grants[j].CreatedAt {
			return grants[i].ToolName < grants[j].ToolName
		}
		return grants[i].CreatedAt > grants[j].CreatedAt
	})
	return grants, nil
}

func (s *ChatService) RevokeToolPermissionGrant(sessionID string, toolName string) error {
	sessionID = strings.TrimSpace(sessionID)
	toolName = strings.TrimSpace(toolName)
	if sessionID == "" {
		return fmt.Errorf("sessionID is required")
	}
	if toolName == "" {
		return fmt.Errorf("toolName is required")
	}
	s.mu.Lock()
	run := s.sessions[sessionID]
	s.mu.Unlock()
	if run == nil {
		return nil
	}
	run.stateMu.Lock()
	delete(run.permissionGrants, toolName)
	run.stateMu.Unlock()
	wailsEmit(s.ctx, "runtime:permission_grants_changed", map[string]string{"sessionId": sessionID})
	s.saveSessionPermissionState(sessionID)
	return nil
}

func (s *ChatService) ClearToolPermissionGrants(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("sessionID is required")
	}
	s.mu.Lock()
	run := s.sessions[sessionID]
	s.mu.Unlock()
	if run == nil {
		return nil
	}
	run.stateMu.Lock()
	run.permissionGrants = make(map[string]model.RuntimePermissionGrant)
	run.stateMu.Unlock()
	wailsEmit(s.ctx, "runtime:permission_grants_changed", map[string]string{"sessionId": sessionID})
	s.saveSessionPermissionState(sessionID)
	return nil
}

func normalizePermissionDecision(decision string) string {
	switch strings.TrimSpace(decision) {
	case tools.ToolPermissionDecisionAllowSession, "allowSession", "session":
		return tools.ToolPermissionDecisionAllowSession
	case tools.ToolPermissionDecisionDeny:
		return tools.ToolPermissionDecisionDeny
	default:
		return tools.ToolPermissionDecisionAllowOnce
	}
}

func permissionRiskForEntry(entry tools.CatalogEntry) string {
	switch entry.ToolClass {
	case tools.ToolClassRuntimeExec:
		return permissionRiskExecute
	case tools.ToolClassRuntimeTask:
		return permissionRiskDestructive
	case tools.ToolClassRuntimeFile:
		return permissionRiskWrite
	default:
		if entry.Source == tools.ToolSourceMCP {
			return permissionRiskExecute
		}
		return permissionRiskWrite
	}
}

func truncatePermissionInput(input string) string {
	const limit = 4000
	input = strings.TrimSpace(input)
	if len(input) <= limit {
		return input
	}
	return input[:limit] + "\n[truncated]"
}

func runtimeFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sessionIDFromGenericContext(ctx context.Context) string {
	if v, ok := ctx.Value("sessionID").(string); ok {
		return v
	}
	return ""
}
