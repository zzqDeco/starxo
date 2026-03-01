package llm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"

	"starxo/internal/config"
)

type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return t.base.RoundTrip(req)
}

func httpClientWithHeaders(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return nil
	}
	return &http.Client{
		Transport: &headerTransport{
			base:    http.DefaultTransport,
			headers: headers,
		},
	}
}

// NewChatModel creates a ToolCallingChatModel based on the provider config.
func NewChatModel(ctx context.Context, cfg config.LLMConfig) (model.ToolCallingChatModel, error) {
	switch cfg.Type {
	case "openai", "deepseek":
		modelCfg := &openai.ChatModelConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
		}
		if client := httpClientWithHeaders(cfg.Headers); client != nil {
			modelCfg.HTTPClient = client
		}
		return openai.NewChatModel(ctx, modelCfg)
	case "ark":
		arkCfg := &ark.ChatModelConfig{
			APIKey: cfg.APIKey,
			Model:  cfg.Model,
		}
		if client := httpClientWithHeaders(cfg.Headers); client != nil {
			arkCfg.HTTPClient = client
		}
		return ark.NewChatModel(ctx, arkCfg)
	case "ollama":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		ollamaCfg := &ollama.ChatModelConfig{
			BaseURL: baseURL,
			Model:   cfg.Model,
		}
		if client := httpClientWithHeaders(cfg.Headers); client != nil {
			ollamaCfg.HTTPClient = client
		}
		return ollama.NewChatModel(ctx, ollamaCfg)
	default:
		return nil, fmt.Errorf("unsupported LLM provider type: %s", cfg.Type)
	}
}
