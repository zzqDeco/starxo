package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"starxo/internal/config"
	"starxo/internal/tools"
)

type WebSearchDiagnosticsResult struct {
	Enabled         bool                          `json:"enabled"`
	DefaultProvider string                        `json:"defaultProvider"`
	Available       bool                          `json:"available"`
	Summary         string                        `json:"summary"`
	Checks          []WebSearchDiagnosticCheck    `json:"checks"`
	Providers       []WebSearchProviderDiagnostic `json:"providers"`
}

type WebSearchDiagnosticCheck struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type WebSearchProviderDiagnostic struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Endpoint       string `json:"endpoint"`
	Status         string `json:"status"`
	Message        string `json:"message"`
	APIKeyEnv      string `json:"apiKeyEnv,omitempty"`
	APIKeyPresent  bool   `json:"apiKeyPresent,omitempty"`
	PublicEndpoint bool   `json:"publicEndpoint"`
}

type WebSearchSmokeResult struct {
	OK          bool     `json:"ok"`
	Query       string   `json:"query"`
	Provider    string   `json:"provider"`
	URL         string   `json:"url"`
	Results     []string `json:"results"`
	ResultCount int      `json:"resultCount"`
	DurationMs  int64    `json:"durationMs"`
	Message     string   `json:"message"`
}

func (s *SettingsService) DiagnoseWebSearch(cfg config.AppConfig) (WebSearchDiagnosticsResult, error) {
	config.MigrateLegacyDockerConfig(&cfg)
	config.NormalizeAppConfig(&cfg)
	return diagnoseWebSearchConfig(s.ctx, cfg.Agent.WebSearch), nil
}

func (s *SettingsService) TestWebSearch(cfg config.AppConfig, query string) (WebSearchSmokeResult, error) {
	config.MigrateLegacyDockerConfig(&cfg)
	config.NormalizeAppConfig(&cfg)
	query = strings.TrimSpace(query)
	if query == "" {
		query = "Starxo runtime search diagnostic"
	}
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	start := time.Now()
	out, err := runWebSearch(ctx, cfg.Agent.WebSearch, webSearchInput{Query: query, Limit: 3})
	return webSearchSmokeResult(query, time.Since(start), out, err), nil
}

func webSearchSmokeResult(query string, duration time.Duration, out webSearchOutput, err error) WebSearchSmokeResult {
	result := WebSearchSmokeResult{
		Query:      query,
		DurationMs: duration.Milliseconds(),
	}
	if err != nil {
		result.Message = err.Error()
		return result
	}
	result.Provider = out.Provider
	result.URL = out.URL
	result.Results = out.Results
	result.ResultCount = len(out.Results)
	if result.ResultCount == 0 {
		result.Message = fmt.Sprintf("WebSearch returned zero results from %s; check provider configuration and result parsing.", out.Provider)
		return result
	}
	result.OK = true
	result.Message = fmt.Sprintf("WebSearch returned %d result(s) from %s.", len(out.Results), out.Provider)
	return result
}

func diagnoseWebSearchConfig(ctx context.Context, cfg config.WebSearchConfig) WebSearchDiagnosticsResult {
	if ctx == nil {
		ctx = context.Background()
	}
	enabled := cfg.Enabled == nil || *cfg.Enabled
	defaultProvider := strings.TrimSpace(cfg.DefaultProvider)
	if defaultProvider == "" {
		defaultProvider = "duckduckgo"
	}
	result := WebSearchDiagnosticsResult{
		Enabled:         enabled,
		DefaultProvider: defaultProvider,
	}
	if !enabled {
		result.Summary = "WebSearch is disabled."
		result.Checks = append(result.Checks, WebSearchDiagnosticCheck{
			ID:      "enabled",
			Label:   "WebSearch enabled",
			Status:  "skipped",
			Message: "WebSearch is disabled in settings.",
		})
		return result
	}
	result.Checks = append(result.Checks, WebSearchDiagnosticCheck{
		ID:      "enabled",
		Label:   "WebSearch enabled",
		Status:  "pass",
		Message: "WebSearch is enabled.",
	})

	providers := webSearchDiagnosticProviders(cfg, defaultProvider)
	seenDefault := false
	var defaultDiagnostic WebSearchProviderDiagnostic
	failures := 0
	warnings := 0
	for _, provider := range providers {
		diag := diagnoseWebSearchProvider(ctx, provider)
		if strings.EqualFold(diag.Name, defaultProvider) {
			seenDefault = true
			defaultDiagnostic = diag
		}
		switch diag.Status {
		case "fail":
			failures++
		case "warn":
			warnings++
		}
		result.Providers = append(result.Providers, diag)
	}
	if !seenDefault {
		failures++
		result.Checks = append(result.Checks, WebSearchDiagnosticCheck{
			ID:      "default-provider",
			Label:   "Default provider",
			Status:  "fail",
			Message: fmt.Sprintf("Default provider %q is not configured.", defaultProvider),
		})
	} else {
		switch defaultDiagnostic.Status {
		case "skipped":
			failures++
			result.Checks = append(result.Checks, WebSearchDiagnosticCheck{
				ID:      "default-provider",
				Label:   "Default provider",
				Status:  "fail",
				Message: fmt.Sprintf("Default provider %q is disabled.", defaultProvider),
			})
		case "fail":
			result.Checks = append(result.Checks, WebSearchDiagnosticCheck{
				ID:      "default-provider",
				Label:   "Default provider",
				Status:  "fail",
				Message: fmt.Sprintf("Default provider %q is not usable: %s", defaultProvider, defaultDiagnostic.Message),
			})
		case "warn":
			result.Checks = append(result.Checks, WebSearchDiagnosticCheck{
				ID:      "default-provider",
				Label:   "Default provider",
				Status:  "warn",
				Message: fmt.Sprintf("Default provider %q requires runtime approval: %s", defaultProvider, defaultDiagnostic.Message),
			})
		default:
			result.Checks = append(result.Checks, WebSearchDiagnosticCheck{
				ID:      "default-provider",
				Label:   "Default provider",
				Status:  "pass",
				Message: fmt.Sprintf("Default provider %q is available in configuration.", defaultProvider),
			})
		}
	}
	result.Available = failures == 0
	switch {
	case failures > 0:
		result.Summary = fmt.Sprintf("WebSearch has %d blocking issue(s).", failures)
	case warnings > 0:
		result.Summary = fmt.Sprintf("WebSearch is usable with %d warning(s).", warnings)
	default:
		result.Summary = "WebSearch configuration looks ready."
	}
	return result
}

func webSearchDiagnosticProviders(cfg config.WebSearchConfig, defaultProvider string) []config.WebSearchProviderConfig {
	out := make([]config.WebSearchProviderConfig, 0, len(cfg.Providers)+2)
	out = append(out, cfg.Providers...)
	if strings.EqualFold(defaultProvider, "duckduckgo") && !webSearchProviderNamed(out, "duckduckgo") {
		out = append(out, config.WebSearchProviderConfig{Name: "duckduckgo", Type: "duckduckgo"})
	}
	if strings.EqualFold(defaultProvider, "tinyfish") && !webSearchProviderNamed(out, "tinyfish") {
		out = append(out, config.WebSearchProviderConfig{Name: "tinyfish", Type: "tinyfish"})
	}
	return out
}

func webSearchProviderNamed(providers []config.WebSearchProviderConfig, name string) bool {
	for _, provider := range providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name), name) {
			return true
		}
	}
	return false
}

func diagnoseWebSearchProvider(ctx context.Context, provider config.WebSearchProviderConfig) WebSearchProviderDiagnostic {
	name := strings.TrimSpace(provider.Name)
	if name == "" {
		name = strings.TrimSpace(provider.Type)
	}
	providerType := strings.ToLower(strings.TrimSpace(provider.Type))
	if providerType == "" {
		providerType = "http"
	}
	diag := WebSearchProviderDiagnostic{
		Name:   name,
		Type:   providerType,
		Status: "pass",
	}
	if provider.Disabled {
		diag.Status = "skipped"
		diag.Message = "Provider is disabled."
		return diag
	}
	switch providerType {
	case "duckduckgo":
		diag.Endpoint = tools.WebSearchURL("starxo-diagnostic")
	case "tinyfish":
		diag.Endpoint = strings.TrimSpace(provider.Endpoint)
		if diag.Endpoint == "" {
			diag.Endpoint = "https://api.search.tinyfish.ai"
		}
		diag.APIKeyEnv, diag.APIKeyPresent = webSearchAPIKeyState(provider, "TINYFISH_API_KEY")
		if !diag.APIKeyPresent {
			diag.Status = "fail"
			diag.Message = fmt.Sprintf("TinyFish requires %s or an X-API-Key header.", diag.APIKeyEnv)
		}
	case "http":
		diag.Endpoint = strings.TrimSpace(provider.Endpoint)
		if diag.Endpoint == "" {
			diag.Status = "fail"
			diag.Message = "HTTP provider endpoint is required."
			return diag
		}
	default:
		diag.Status = "fail"
		diag.Message = fmt.Sprintf("Unsupported provider type %q.", provider.Type)
		return diag
	}
	if diag.Endpoint == "" {
		diag.Status = "fail"
		diag.Message = "Provider endpoint is required."
		return diag
	}
	guard := newRuntimeWebEndpointGuard(nil)
	decision, err := guard.inspectURL(ctx, diag.Endpoint)
	if err != nil {
		diag.Status = "fail"
		diag.Message = err.Error()
		return diag
	}
	if decision.Reason != "" {
		diag.PublicEndpoint = false
		if diag.Status != "fail" {
			diag.Status = "warn"
			diag.Message = "Endpoint is non-public and will require runtime approval."
		}
		if diag.Message != "" {
			diag.Message += " "
		}
		diag.Message += decision.Reason
		return diag
	}
	diag.PublicEndpoint = true
	if diag.Message == "" {
		diag.Message = "Provider endpoint is public and passes static validation."
	}
	return diag
}

func webSearchAPIKeyState(provider config.WebSearchProviderConfig, defaultEnv string) (string, bool) {
	for k, v := range provider.Headers {
		if strings.EqualFold(strings.TrimSpace(k), "X-API-Key") && strings.TrimSpace(v) != "" {
			return "", true
		}
	}
	envName := strings.TrimSpace(provider.APIKeyEnv)
	if envName == "" {
		envName = defaultEnv
	}
	return envName, strings.TrimSpace(os.Getenv(envName)) != ""
}
