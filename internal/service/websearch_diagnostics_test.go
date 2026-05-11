package service

import (
	"strings"
	"testing"

	"starxo/internal/config"
)

func TestDiagnoseWebSearchConfigPassesPublicHTTPProvider(t *testing.T) {
	enabled := true
	result := diagnoseWebSearchConfig(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "custom",
		Providers: []config.WebSearchProviderConfig{{
			Name:     "custom",
			Type:     "http",
			Endpoint: "http://93.184.216.34/search",
		}},
	})
	if !result.Available || result.Summary == "" {
		t.Fatalf("expected available diagnostics, got %#v", result)
	}
	if len(result.Providers) != 1 || result.Providers[0].Status != "pass" || !result.Providers[0].PublicEndpoint {
		t.Fatalf("unexpected provider diagnostics: %#v", result.Providers)
	}
}

func TestDiagnoseWebSearchConfigReportsTinyFishAPIKey(t *testing.T) {
	enabled := true
	result := diagnoseWebSearchConfig(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "tinyfish",
		Providers: []config.WebSearchProviderConfig{{
			Name:      "tinyfish",
			Type:      "tinyfish",
			Endpoint:  "http://93.184.216.34",
			APIKeyEnv: "STARXO_TEST_TINYFISH_MISSING",
		}},
	})
	if result.Available {
		t.Fatalf("expected unavailable diagnostics, got %#v", result)
	}
	if len(result.Providers) != 1 || result.Providers[0].Status != "fail" || result.Providers[0].APIKeyEnv != "STARXO_TEST_TINYFISH_MISSING" {
		t.Fatalf("unexpected TinyFish diagnostics: %#v", result.Providers)
	}
	if !strings.Contains(result.Providers[0].Message, "TinyFish requires") {
		t.Fatalf("expected missing key message, got %q", result.Providers[0].Message)
	}
}

func TestDiagnoseWebSearchConfigFailsDisabledDefaultProvider(t *testing.T) {
	enabled := true
	result := diagnoseWebSearchConfig(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "custom",
		Providers: []config.WebSearchProviderConfig{{
			Name:     "custom",
			Type:     "http",
			Endpoint: "http://93.184.216.34/search",
			Disabled: true,
		}},
	})
	if result.Available {
		t.Fatalf("expected disabled default provider to be unavailable, got %#v", result)
	}
	check := webSearchDiagnosticCheckByID(result.Checks, "default-provider")
	if check.Status != "fail" || !strings.Contains(check.Message, "disabled") {
		t.Fatalf("expected disabled default-provider failure, got %#v", check)
	}
}

func TestDiagnoseWebSearchConfigWarnsForNonPublicEndpoint(t *testing.T) {
	enabled := true
	result := diagnoseWebSearchConfig(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "local",
		Providers: []config.WebSearchProviderConfig{{
			Name:     "local",
			Type:     "http",
			Endpoint: "http://127.0.0.1:8080/search",
		}},
	})
	if !result.Available {
		t.Fatalf("non-public endpoints should warn instead of fail because runtime approval can allow them: %#v", result)
	}
	if len(result.Providers) != 1 || result.Providers[0].Status != "warn" || result.Providers[0].PublicEndpoint {
		t.Fatalf("unexpected non-public diagnostics: %#v", result.Providers)
	}
}

func webSearchDiagnosticCheckByID(checks []WebSearchDiagnosticCheck, id string) WebSearchDiagnosticCheck {
	for _, check := range checks {
		if check.ID == id {
			return check
		}
	}
	return WebSearchDiagnosticCheck{}
}
