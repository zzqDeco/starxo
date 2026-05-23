package llm

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/components/model"

	"starxo/internal/config"
)

const (
	AgenticProtocolOff    = "off"
	AgenticProtocolAuto   = "auto"
	AgenticProtocolOpenAI = "agentic_openai"
	AgenticProtocolArk    = "agentic_ark"
)

// NewAgenticModel creates an experimental Eino v0.9 AgenticModel. The main
// Starxo runtime still uses *schema.Message by default; this path is only used
// by explicit beta configuration and must be allowed to fall back cleanly.
func NewAgenticModel(ctx context.Context, llmCfg config.LLMConfig, protocol string) (model.AgenticModel, error) {
	switch protocol {
	case AgenticProtocolOpenAI:
		cfg := &agenticopenai.Config{
			BaseURL: llmCfg.BaseURL,
			APIKey:  llmCfg.APIKey,
			Model:   llmCfg.Model,
		}
		if client := httpClientWithHeaders(llmCfg.Headers); client != nil {
			cfg.HTTPClient = client
		}
		return agenticopenai.New(ctx, cfg)
	case AgenticProtocolArk:
		cfg := &agenticark.Config{
			BaseURL: llmCfg.BaseURL,
			APIKey:  llmCfg.APIKey,
			Model:   llmCfg.Model,
		}
		if client := httpClientWithHeaders(llmCfg.Headers); client != nil {
			cfg.HTTPClient = client
		}
		return agenticark.New(ctx, cfg)
	case AgenticProtocolAuto:
		switch llmCfg.Type {
		case "ark":
			return NewAgenticModel(ctx, llmCfg, AgenticProtocolArk)
		case "openai", "deepseek":
			return NewAgenticModel(ctx, llmCfg, AgenticProtocolOpenAI)
		default:
			return nil, fmt.Errorf("agentic protocol auto does not support provider type %s", llmCfg.Type)
		}
	case "", AgenticProtocolOff:
		return nil, fmt.Errorf("agentic protocol is disabled")
	default:
		return nil, fmt.Errorf("unsupported agentic protocol: %s", protocol)
	}
}
