package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *slog.Logger
	logFile  *os.File
	mu       sync.Mutex
)

// Init initializes the global logger. Logs are written to both stderr (for wails dev console)
// and a daily-rotated file under <projectRoot>/logs/agent-YYYY-MM-DD.log.
func Init(projectRoot string) error {
	mu.Lock()
	defer mu.Unlock()

	logDir := filepath.Join(projectRoot, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	fileName := fmt.Sprintf("agent-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, fileName)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", logPath, err)
	}

	if logFile != nil {
		logFile.Close()
	}
	logFile = f

	// Write to both file and stderr
	multiWriter := io.MultiWriter(os.Stderr, f)

	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	})

	instance = slog.New(handler)
	slog.SetDefault(instance)

	instance.Info("Logger initialized",
		"logPath", logPath,
		"pid", os.Getpid(),
	)

	return nil
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Sync()
		logFile.Close()
		logFile = nil
	}
}

// L returns the global logger instance.
func L() *slog.Logger {
	if instance == nil {
		return slog.Default()
	}
	return instance
}

// --- Convenience wrappers with domain-specific context ---

// AgentEvent logs an agent lifecycle event.
func AgentEvent(event string, agent string, attrs ...any) {
	args := []any{"event", event, "agent", agent}
	args = append(args, attrs...)
	L().Info("[AGENT]", args...)
}

// Transfer logs an agent-to-agent transfer.
func Transfer(from string, to string, attrs ...any) {
	args := []any{"from", from, "to", to}
	args = append(args, attrs...)
	L().Info("[TRANSFER]", args...)
}

// ToolCall logs a tool invocation.
func ToolCall(agent string, tool string, args string) {
	truncArgs := truncate(args, 500)
	L().Info("[TOOL_CALL]",
		"agent", agent,
		"tool", tool,
		"args", truncArgs,
	)
}

// ToolResult logs a tool execution result.
func ToolResult(agent string, tool string, result string, duration time.Duration) {
	truncResult := truncate(result, 800)
	L().Info("[TOOL_RESULT]",
		"agent", agent,
		"tool", tool,
		"result", truncResult,
		"duration_ms", duration.Milliseconds(),
	)
}

// ToolError logs a tool execution error.
func ToolError(agent string, tool string, err error, duration time.Duration) {
	L().Error("[TOOL_ERROR]",
		"agent", agent,
		"tool", tool,
		"error", err.Error(),
		"duration_ms", duration.Milliseconds(),
	)
}

// ModelCall logs an LLM model invocation.
func ModelCall(agent string, messageCount int, attrs ...any) {
	args := []any{"agent", agent, "message_count", messageCount}
	args = append(args, attrs...)
	L().Info("[MODEL_CALL]", args...)
}

// ModelResult logs an LLM model response.
func ModelResult(agent string, hasToolCalls bool, contentLen int, attrs ...any) {
	args := []any{
		"agent", agent,
		"has_tool_calls", hasToolCalls,
		"content_length", contentLen,
	}
	args = append(args, attrs...)
	L().Info("[MODEL_RESULT]", args...)
}

// TokenUsage logs token consumption for a model call.
func TokenUsage(agent string, promptTokens, completionTokens, totalTokens int64) {
	L().Info("[TOKEN_USAGE]",
		"agent", agent,
		"prompt_tokens", promptTokens,
		"completion_tokens", completionTokens,
		"total_tokens", totalTokens,
	)
}

// SessionEvent logs a session lifecycle event.
func SessionEvent(event string, sessionID string, attrs ...any) {
	args := []any{"event", event, "session_id", sessionID}
	args = append(args, attrs...)
	L().Info("[SESSION]", args...)
}

// RunnerEvent logs a runner lifecycle event.
func RunnerEvent(event string, attrs ...any) {
	args := []any{"event", event}
	args = append(args, attrs...)
	L().Info("[RUNNER]", args...)
}

// Error logs a general error.
func Error(msg string, err error, attrs ...any) {
	args := []any{"error", err.Error()}
	args = append(args, attrs...)
	L().Error(msg, args...)
}

// Debug logs a debug message.
func Debug(msg string, attrs ...any) {
	L().Debug(msg, attrs...)
}

// Info logs an info message.
func Info(msg string, attrs ...any) {
	L().Info(msg, attrs...)
}

// Warn logs a warning message.
func Warn(msg string, attrs ...any) {
	L().Warn(msg, attrs...)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}
