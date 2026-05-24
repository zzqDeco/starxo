package service

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"

	"starxo/internal/agent"
	"starxo/internal/config"
	"starxo/internal/logger"
)

func buildTopLevelRuntimeAgents(
	ctx context.Context,
	cfg *config.AppConfig,
	mdl einomodel.ToolCallingChatModel,
	op commandline.Operator,
	extraTools []einotool.BaseTool,
	ac agent.AgentContext,
	defaultHandlers []adk.ChatModelAgentMiddleware,
	planHandlers []adk.ChatModelAgentMiddleware,
	unknownToolsHandler func(ctx context.Context, name, input string) (string, error),
	subagentRegistry *agent.SubagentRegistry,
) (adk.Agent, adk.Agent, error) {
	if cfg != nil && cfg.Agent.Runtime.EnableBuiltinDeepTransferFallback {
		defaultAgent, err := agent.BuildDeepAgentForMode(
			ctx, mdl, op, extraTools, ac, agent.DeepAgentModeDefault,
			defaultHandlers,
			unknownToolsHandler,
			true,
			subagentRegistry,
		)
		if err != nil {
			logger.Error("[RUNNER] Failed to build fallback default deep agent", err)
			return nil, nil, fmt.Errorf("failed to build fallback default deep agent: %w", err)
		}
		planAgent, err := agent.BuildDeepAgentForMode(
			ctx, mdl, op, extraTools, ac, agent.DeepAgentModePlan,
			planHandlers,
			unknownToolsHandler,
			true,
			subagentRegistry,
		)
		if err != nil {
			logger.Error("[RUNNER] Failed to build fallback plan deep agent", err)
			return nil, nil, fmt.Errorf("failed to build fallback plan deep agent: %w", err)
		}
		return defaultAgent, planAgent, nil
	}

	defaultAgent, err := agent.BuildRuntimeAgent(
		ctx, mdl, op, extraTools, ac, agent.DeepAgentModeDefault,
		defaultHandlers,
		unknownToolsHandler,
		subagentRegistry,
	)
	if err != nil {
		logger.Error("[RUNNER] Failed to build default runtime agent", err)
		return nil, nil, fmt.Errorf("failed to build default runtime agent: %w", err)
	}
	planAgent, err := agent.BuildRuntimeAgent(
		ctx, mdl, op, extraTools, ac, agent.DeepAgentModePlan,
		planHandlers,
		unknownToolsHandler,
		subagentRegistry,
	)
	if err != nil {
		logger.Error("[RUNNER] Failed to build plan runtime agent", err)
		return nil, nil, fmt.Errorf("failed to build plan runtime agent: %w", err)
	}
	return defaultAgent, planAgent, nil
}
