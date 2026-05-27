package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/schema"

	agentctx "starxo/internal/context"
	"starxo/internal/model"
	"starxo/internal/tools"
)

const (
	runtimeCompactRecentMessages = 12
	runtimeCompactMaxDiffs       = 20
	runtimeCompactMaxFileReads   = 40
	runtimeCompactMaxTaskItems   = 20
)

type pendingRuntimeToolCall struct {
	name string
	args string
}

func (s *ChatService) prepareMessagesForRun(sessionID string, run *SessionRun) []*schema.Message {
	compact := s.refreshRuntimeContextCompact(sessionID, run)
	return run.prepareMessagesWithCompact(compact)
}

func (s *ChatService) refreshRuntimeContextCompact(sessionID string, run *SessionRun) *model.RuntimeContextCompact {
	if run == nil {
		return nil
	}
	now := s.now().UnixMilli()
	taskSnapshots := []model.RuntimeTaskCompact(nil)
	taskItems := []model.RuntimeTaskItemCompact(nil)
	if s.runtimeTasks != nil {
		taskSnapshots = s.runtimeTasks.CompactSnapshots(sessionID)
		taskItems = s.runtimeTasks.CompactTaskItems(sessionID)
	}

	var workspace *model.RuntimeWorkspaceCompact
	if s.runtimeWorkspaces != nil {
		workspace = s.runtimeWorkspaces.CompactSnapshot(sessionID, "")
	}

	run.stateMu.Lock()
	defer run.stateMu.Unlock()

	discovered := make([]model.DiscoveredToolRecord, 0, len(run.discoveredTools))
	for _, record := range run.discoveredTools {
		discovered = append(discovered, record)
	}
	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].CanonicalName < discovered[j].CanonicalName
	})
	grants := make([]model.RuntimePermissionGrant, 0, len(run.permissionGrants))
	for _, grant := range run.permissionGrants {
		grants = append(grants, grant)
	}
	sort.Slice(grants, func(i, j int) bool {
		return grants[i].ToolName < grants[j].ToolName
	})
	permissionAudit := append([]model.RuntimePermissionAudit(nil), run.permissionAudit...)
	sort.Slice(permissionAudit, func(i, j int) bool {
		if permissionAudit[i].ResolvedAt == permissionAudit[j].ResolvedAt {
			return permissionAudit[i].RequestID < permissionAudit[j].RequestID
		}
		return permissionAudit[i].ResolvedAt > permissionAudit[j].ResolvedAt
	})
	if len(permissionAudit) > 40 {
		permissionAudit = permissionAudit[:40]
	}
	fileReads := make([]model.RuntimeFileReadState, 0, len(run.fileReadState))
	for _, state := range run.fileReadState {
		fileReads = append(fileReads, state)
	}
	sort.Slice(fileReads, func(i, j int) bool {
		if fileReads[i].LastReadAt == fileReads[j].LastReadAt {
			return fileReads[i].FilePath < fileReads[j].FilePath
		}
		return fileReads[i].LastReadAt > fileReads[j].LastReadAt
	})
	if len(fileReads) > runtimeCompactMaxFileReads {
		fileReads = fileReads[:runtimeCompactMaxFileReads]
	}
	diffs := append([]model.RuntimeDiffSummary(nil), run.diffSummaries...)
	if len(diffs) > runtimeCompactMaxDiffs {
		diffs = diffs[len(diffs)-runtimeCompactMaxDiffs:]
	}

	messages := run.ctxEngine.ExportMessages()
	compact := &model.RuntimeContextCompact{
		Version:              model.RuntimeContextCompactVersion,
		CreatedAt:            now,
		UpdatedAt:            now,
		OriginalMessageCount: len(messages),
		OmittedMessageCount:  runtimeCompactOmittedCount(messages),
		TokenEstimate:        agentctx.EstimatePersistedMessagesTokens(messages),
		ToolSearch: model.RuntimeToolSearchCompact{
			DiscoveredTools:           discovered,
			DeferredAnnouncementState: cloneDeferredAnnouncementState(run.deferredAnnouncementState),
			MCPInstructionsDeltaState: cloneMCPInstructionsDeltaState(run.mcpInstructionsDeltaState),
		},
		PermissionGrants: grants,
		PermissionAudit:  permissionAudit,
		Tasks:            taskSnapshots,
		TaskItems:        taskItems,
		FileReadState:    fileReads,
		DiffSummaries:    diffs,
		Todos:            tools.SnapshotTodosForSession(sessionID),
		PlanDocument:     model.ClonePlanDocument(run.planDocument),
		Workspace:        workspace,
		ActiveObjective:  cloneRunObjective(run.activeObjective),
	}
	if run.runtimeContextCompact != nil && run.runtimeContextCompact.CreatedAt > 0 {
		compact.CreatedAt = run.runtimeContextCompact.CreatedAt
	}
	compact.Summary = buildRuntimeCompactSummary(messages, compact)
	if !runtimeCompactHasContent(compact) {
		run.runtimeContextCompact = nil
		return nil
	}
	run.runtimeContextCompact = model.CloneRuntimeContextCompact(compact)
	return model.CloneRuntimeContextCompact(compact)
}

func runtimeCompactOmittedCount(messages []model.PersistedMessage) int {
	if len(messages) <= runtimeCompactRecentMessages {
		return 0
	}
	return len(messages) - runtimeCompactRecentMessages
}

func runtimeCompactHasContent(compact *model.RuntimeContextCompact) bool {
	if compact == nil {
		return false
	}
	return compact.OmittedMessageCount > 0 ||
		len(compact.ToolSearch.DiscoveredTools) > 0 ||
		len(compact.PermissionGrants) > 0 ||
		len(compact.PermissionAudit) > 0 ||
		len(compact.Tasks) > 0 ||
		len(compact.TaskItems) > 0 ||
		len(compact.FileReadState) > 0 ||
		len(compact.DiffSummaries) > 0 ||
		len(compact.Todos) > 0 ||
		(compact.PlanDocument != nil && strings.TrimSpace(compact.PlanDocument.Markdown) != "") ||
		(compact.Workspace != nil && compact.Workspace.Active) ||
		(compact.ActiveObjective != nil && strings.TrimSpace(compact.ActiveObjective.Objective) != "")
}

func buildRuntimeCompactSummary(messages []model.PersistedMessage, compact *model.RuntimeContextCompact) string {
	var b strings.Builder
	if compact.OmittedMessageCount > 0 {
		b.WriteString("Conversation summary before the recent window:\n")
		limit := compact.OmittedMessageCount
		if limit > len(messages) {
			limit = len(messages)
		}
		start := limit - 8
		if start < 0 {
			start = 0
		}
		for _, msg := range messages[start:limit] {
			line := persistedMessageOneLine(msg)
			if line == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if len(compact.FileReadState) > 0 {
		b.WriteString(fmt.Sprintf("The agent has read %d file range(s); keep mtime/hash/range state in mind before editing.\n", len(compact.FileReadState)))
	}
	if len(compact.DiffSummaries) > 0 {
		b.WriteString(fmt.Sprintf("The agent has %d recent file modification summary item(s).\n", len(compact.DiffSummaries)))
	}
	if len(compact.Tasks) > 0 {
		b.WriteString(fmt.Sprintf("There are %d runtime task snapshot(s) with output paths preserved.\n", len(compact.Tasks)))
	}
	if len(compact.TaskItems) > 0 {
		b.WriteString(fmt.Sprintf("There are %d persistent task graph item(s) from TaskCreate/TaskUpdate.\n", len(compact.TaskItems)))
		limit := len(compact.TaskItems)
		if limit > runtimeCompactMaxTaskItems {
			limit = runtimeCompactMaxTaskItems
		}
		for _, item := range compact.TaskItems[:limit] {
			line := runtimeTaskItemCompactPromptLine(item)
			if line == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		if len(compact.TaskItems) > limit {
			b.WriteString(fmt.Sprintf("- ... %d more task graph item(s) omitted from prompt\n", len(compact.TaskItems)-limit))
		}
	}
	if len(compact.PermissionAudit) > 0 {
		b.WriteString(fmt.Sprintf("There are %d recent permission decision audit record(s).\n", len(compact.PermissionAudit)))
	}
	return strings.TrimSpace(b.String())
}

func runtimeTaskItemCompactPromptLine(item model.RuntimeTaskItemCompact) string {
	id := strings.TrimSpace(item.ID)
	title := strings.Join(strings.Fields(item.Title), " ")
	if id == "" || title == "" {
		return ""
	}
	status := strings.TrimSpace(item.Status)
	if status == "" {
		status = runtimeTaskItemStatusTodo
	}
	parts := []string{
		"id=" + id,
		"status=" + status,
		"title=" + fmt.Sprintf("%q", truncateRuntimeCompactText(title, 120)),
	}
	if len(item.DependsOn) > 0 {
		deps := make([]string, 0, len(item.DependsOn))
		for _, dep := range item.DependsOn {
			dep = strings.TrimSpace(dep)
			if dep != "" {
				deps = append(deps, dep)
			}
		}
		if len(deps) > 0 {
			parts = append(parts, "depends_on=["+strings.Join(deps, ",")+"]")
		}
	}
	if owner := strings.TrimSpace(item.Owner); owner != "" {
		parts = append(parts, "owner="+owner)
	}
	if priority := strings.TrimSpace(item.Priority); priority != "" {
		parts = append(parts, "priority="+priority)
	}
	return strings.Join(parts, " ")
}

func truncateRuntimeCompactText(text string, maxLen int) string {
	if maxLen <= 0 || len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

func persistedMessageOneLine(msg model.PersistedMessage) string {
	role := strings.TrimSpace(msg.Role)
	if role == "" {
		role = "message"
	}
	if len(msg.ToolCalls) > 0 {
		names := make([]string, 0, len(msg.ToolCalls))
		for _, call := range msg.ToolCalls {
			if call.Function.Name != "" {
				names = append(names, call.Function.Name)
			}
		}
		if len(names) > 0 {
			return fmt.Sprintf("%s called %s", role, strings.Join(names, ", "))
		}
	}
	content := strings.Join(strings.Fields(msg.Content), " ")
	if content == "" {
		return ""
	}
	if len(content) > 180 {
		content = content[:177] + "..."
	}
	return fmt.Sprintf("%s: %s", role, content)
}

func (r *SessionRun) recordRuntimeToolResult(toolName, argsJSON, resultJSON string, nowMillis int64) {
	canonical := canonicalRuntimeToolName(toolName)
	switch canonical {
	case tools.RuntimeToolRead:
		r.recordReadToolResult(argsJSON, resultJSON, nowMillis)
	case tools.RuntimeToolWrite:
		r.recordWriteToolResult(argsJSON, resultJSON, nowMillis)
	case tools.RuntimeToolEdit:
		r.recordEditToolResult(argsJSON, resultJSON, nowMillis)
	}
}

func (r *SessionRun) recordReadToolResult(argsJSON, resultJSON string, nowMillis int64) {
	var out tools.ReadOutput
	if err := json.Unmarshal([]byte(resultJSON), &out); err != nil {
		var args tools.ReadInput
		_ = json.Unmarshal([]byte(argsJSON), &args)
		if strings.TrimSpace(args.FilePath) == "" {
			return
		}
		out.FilePath = args.FilePath
	}
	if strings.TrimSpace(out.FilePath) == "" {
		return
	}
	hash := ""
	if out.Content != "" {
		sum := sha256.Sum256([]byte(out.Content))
		hash = hex.EncodeToString(sum[:])
	}
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.fileReadState == nil {
		r.fileReadState = make(map[string]model.RuntimeFileReadState)
	}
	r.fileReadState[out.FilePath] = model.RuntimeFileReadState{
		FilePath:    out.FilePath,
		StartLine:   out.StartLine,
		NumLines:    out.NumLines,
		TotalLines:  out.TotalLines,
		ContentHash: hash,
		LastReadAt:  nowMillis,
	}
}

func (r *SessionRun) recordWriteToolResult(argsJSON, resultJSON string, nowMillis int64) {
	var out tools.WriteOutput
	if err := json.Unmarshal([]byte(resultJSON), &out); err != nil {
		var args tools.WriteInput
		_ = json.Unmarshal([]byte(argsJSON), &args)
		if strings.TrimSpace(args.FilePath) == "" {
			return
		}
		out.FilePath = args.FilePath
		out.Bytes = len(args.Content)
		out.LinesAdded = len(strings.Split(strings.TrimSuffix(args.Content, "\n"), "\n"))
	}
	if strings.TrimSpace(out.FilePath) == "" {
		return
	}
	summary := fmt.Sprintf("Write %s (%d bytes, +%d -%d)", out.FilePath, out.Bytes, out.LinesAdded, out.LinesRemoved)
	r.appendDiffSummary(model.RuntimeDiffSummary{
		FilePath:     out.FilePath,
		ToolName:     tools.RuntimeToolWrite,
		LinesAdded:   out.LinesAdded,
		LinesRemoved: out.LinesRemoved,
		Patch:        out.Patch,
		Summary:      summary,
		UpdatedAt:    nowMillis,
	})
}

func (r *SessionRun) recordEditToolResult(argsJSON, resultJSON string, nowMillis int64) {
	var out tools.EditOutput
	if err := json.Unmarshal([]byte(resultJSON), &out); err != nil {
		var args tools.EditInput
		_ = json.Unmarshal([]byte(argsJSON), &args)
		if strings.TrimSpace(args.FilePath) == "" {
			return
		}
		out.FilePath = args.FilePath
	}
	if strings.TrimSpace(out.FilePath) == "" {
		return
	}
	summary := fmt.Sprintf("Edit %s (%d replacement(s), +%d -%d)", out.FilePath, out.Replacements, out.LinesAdded, out.LinesRemoved)
	r.appendDiffSummary(model.RuntimeDiffSummary{
		FilePath:     out.FilePath,
		ToolName:     tools.RuntimeToolEdit,
		LinesAdded:   out.LinesAdded,
		LinesRemoved: out.LinesRemoved,
		Replacements: out.Replacements,
		Patch:        out.Patch,
		Summary:      summary,
		UpdatedAt:    nowMillis,
	})
}

func (r *SessionRun) appendDiffSummary(summary model.RuntimeDiffSummary) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	r.diffSummaries = append(r.diffSummaries, summary)
	if len(r.diffSummaries) > runtimeCompactMaxDiffs {
		r.diffSummaries = append([]model.RuntimeDiffSummary(nil), r.diffSummaries[len(r.diffSummaries)-runtimeCompactMaxDiffs:]...)
	}
}

func canonicalRuntimeToolName(name string) string {
	switch strings.TrimSpace(name) {
	case tools.RuntimeToolRead, "read_file":
		return tools.RuntimeToolRead
	case tools.RuntimeToolWrite, "write_file":
		return tools.RuntimeToolWrite
	case tools.RuntimeToolEdit, "str_replace_editor":
		return tools.RuntimeToolEdit
	default:
		return strings.TrimSpace(name)
	}
}
