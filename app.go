package main

import (
	"context"
	"os"
	"path/filepath"

	"starxo/internal/config"
	agentctx "starxo/internal/context"
	"starxo/internal/logger"
	"starxo/internal/sandbox"
	"starxo/internal/service"
	"starxo/internal/storage"
)

// App is the main application struct that holds all services and state.
type App struct {
	ctx              context.Context
	store            *config.Store
	sessionStore     *storage.SessionStore
	containerStore   *storage.ContainerStore
	chatService      *service.ChatService
	sandboxService   *service.SandboxService
	fileService      *service.FileService
	settingsService  *service.SettingsService
	sessionService   *service.SessionService
	containerService *service.ContainerService
	ctxEngine        *agentctx.Engine
}

// NewApp creates a new App with all services initialized.
func NewApp() *App {
	store, _ := config.NewStore()
	sessionStore, _ := storage.NewSessionStore()
	containerStore, _ := storage.NewContainerStore()

	sandboxSvc := service.NewSandboxService(store, containerStore)
	containerSvc := service.NewContainerService(containerStore, sandboxSvc)
	sessionSvc := service.NewSessionService(sessionStore, containerStore)

	return &App{
		store:            store,
		sessionStore:     sessionStore,
		containerStore:   containerStore,
		chatService:      service.NewChatService(store),
		sandboxService:   sandboxSvc,
		fileService:      service.NewFileService(sandboxSvc),
		settingsService:  service.NewSettingsService(store),
		sessionService:   sessionSvc,
		containerService: containerSvc,
		ctxEngine:        agentctx.NewEngine("You are an intelligent coding agent that helps users write, debug, and execute code in a sandboxed environment. You have access to tools for file operations, shell commands, and code execution. Always explain your approach before taking action.", 8000),
	}
}

// startup is called when the Wails app starts. It distributes the Wails runtime
// context to all services and wires up dependencies.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize persistent logger
	execPath, _ := os.Executable()
	projectRoot := filepath.Dir(execPath)
	if wd, err := os.Getwd(); err == nil {
		projectRoot = wd
	}
	if err := logger.Init(projectRoot); err != nil {
		logger.Warn("Failed to initialize file logger, using stderr only", "error", err)
	}
	logger.RegisterGlobalCallbacks()
	logger.Info("Application starting", "projectRoot", projectRoot)

	// Set Wails context on all services
	a.chatService.SetContext(ctx)
	a.sandboxService.SetContext(ctx)
	a.fileService.SetContext(ctx)
	a.settingsService.SetContext(ctx)
	a.sessionService.SetContext(ctx)
	a.containerService.SetContext(ctx)

	// Wire up session service dependencies
	a.sessionService.SetCtxEngine(a.ctxEngine)

	// Wire up chat service dependencies.
	// Manager may be nil at startup since sandbox is not yet connected.
	a.chatService.SetDependencies(a.sandboxService.Manager(), a.ctxEngine)
	a.chatService.SetSessionService(a.sessionService)
	a.fileService.SetSessionService(a.sessionService)

	// Give sandbox service access to session service for container ownership
	a.sandboxService.SetSessionService(a.sessionService)
	a.containerService.SetSessionService(a.sessionService)

	// When sandbox connects, update the chat service with the new manager
	a.sandboxService.SetOnConnect(func(mgr *sandbox.SandboxManager) {
		a.chatService.UpdateSandbox(mgr)
	})

	// When a container is connected, bind it to the current session
	a.sandboxService.SetOnContainerBound(func(regID, wsPath string) {
		a.sessionService.BindContainer(regID, wsPath)
	})

	// When the active session switches, auto-reconnect its container or disconnect
	a.sessionService.SetOnSessionSwitch(func(containerRegID string) {
		if containerRegID != "" {
			go a.sandboxService.ConnectExisting(containerRegID)
		} else {
			// No container bound — disconnect current sandbox to avoid cross-session leakage
			_ = a.sandboxService.Disconnect()
		}
	})

	// When a session is deleted, cascade-destroy its containers on remote
	a.sessionService.SetOnDestroyContainer(func(containerRegID string) error {
		return a.containerService.DestroyContainer(containerRegID)
	})

	// When settings are saved, invalidate the runner so it rebuilds with new config
	a.settingsService.SetOnSettingsSave(func() {
		a.chatService.InvalidateRunner()
	})

	// Auto-save session after agent finishes
	a.chatService.SetOnAgentDone(func() {
		_ = a.sessionService.SaveCurrentSession()
	})

	// Load or create default session
	_ = a.sessionService.EnsureDefaultSession()

	// Start background health monitor for sandbox connection
	a.sandboxService.StartHealthMonitor(ctx)
}

// shutdown is called when the Wails app is closing.
// It saves the current session and disconnects SSH (keeps containers alive).
func (a *App) shutdown(ctx context.Context) {
	logger.Info("Application shutting down")
	// Save current session
	_ = a.sessionService.SaveCurrentSession()

	// Disconnect SSH but keep containers alive for future reconnection
	if a.sandboxService.Manager() != nil {
		_ = a.sandboxService.Disconnect()
	}
	logger.Close()
}
