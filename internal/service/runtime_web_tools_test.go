package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"starxo/internal/config"
)

func TestRunWebSearchUsesConfiguredHTTPProvider(t *testing.T) {
	var gotQuery string
	var gotLimit string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		gotLimit = r.URL.Query().Get("n")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"results": []map[string]any{{
					"title":   "TinyFish result",
					"url":     "https://example.com/tinyfish",
					"content": "custom provider content",
				}},
			},
		})
	}))
	defer server.Close()

	enabled := true
	out, err := runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "custom",
		Providers: []config.WebSearchProviderConfig{{
			Name:        "custom",
			Type:        "http",
			Endpoint:    server.URL + "/search",
			QueryParam:  "q",
			LimitParam:  "n",
			ResultsPath: "data.results",
			TitlePath:   "title",
			URLPath:     "url",
			SnippetPath: "content",
		}},
	}, webSearchInput{Query: "starxo", Limit: 3})
	if err != nil {
		t.Fatalf("run web search: %v", err)
	}
	if gotQuery != "starxo" || gotLimit != "3" {
		t.Fatalf("unexpected query params q=%q n=%q", gotQuery, gotLimit)
	}
	if out.Provider != "custom" || len(out.Results) != 1 || !strings.Contains(out.Results[0], "TinyFish result") {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestRunWebSearchUsesTinyFishProviderContract(t *testing.T) {
	t.Setenv("STARXO_TINYFISH_TEST_KEY", "tf-test-key")

	var gotAPIKey string
	var gotQuery string
	var gotLocation string
	var gotLanguage string
	var gotPage string
	var gotLimit string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-API-Key")
		gotQuery = r.URL.Query().Get("query")
		gotLocation = r.URL.Query().Get("location")
		gotLanguage = r.URL.Query().Get("language")
		gotPage = r.URL.Query().Get("page")
		gotLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"query":         "starxo",
			"total_results": 10,
			"results": []map[string]any{
				{
					"position":  1,
					"site_name": "example.com",
					"title":     "TinyFish official result",
					"snippet":   "structured ranked web result",
					"url":       "https://example.com/tinyfish",
				},
				{
					"position":  2,
					"site_name": "example.org",
					"title":     "Extra result",
					"snippet":   "should be limited out",
					"url":       "https://example.org/extra",
				},
			},
		})
	}))
	defer server.Close()

	enabled := true
	out, err := runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "tinyfish",
		Providers: []config.WebSearchProviderConfig{{
			Name:      "tinyfish",
			Type:      "tinyfish",
			Endpoint:  server.URL,
			APIKeyEnv: "STARXO_TINYFISH_TEST_KEY",
			Location:  "US",
			Language:  "en",
		}},
	}, webSearchInput{Query: "starxo", Limit: 1, Page: 2})
	if err != nil {
		t.Fatalf("run web search: %v", err)
	}
	if gotAPIKey != "tf-test-key" {
		t.Fatalf("unexpected api key header %q", gotAPIKey)
	}
	if gotQuery != "starxo" || gotLocation != "US" || gotLanguage != "en" || gotPage != "2" {
		t.Fatalf("unexpected tinyfish query params query=%q location=%q language=%q page=%q", gotQuery, gotLocation, gotLanguage, gotPage)
	}
	if gotLimit != "" {
		t.Fatalf("tinyfish provider must not send unsupported limit param, got %q", gotLimit)
	}
	if out.Provider != "tinyfish" || len(out.Results) != 1 || !strings.Contains(out.Results[0], "TinyFish official result") {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestRunWebSearchTinyFishRequiresAPIKey(t *testing.T) {
	enabled := true
	_, err := runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "tinyfish",
		Providers: []config.WebSearchProviderConfig{{
			Name:      "tinyfish",
			Type:      "tinyfish",
			Endpoint:  "https://api.search.tinyfish.ai",
			APIKeyEnv: "STARXO_TINYFISH_MISSING_KEY",
		}},
	}, webSearchInput{Query: "starxo"})
	if err == nil || !strings.Contains(err.Error(), "STARXO_TINYFISH_MISSING_KEY") {
		t.Fatalf("expected missing api key error, got %v", err)
	}
}

func TestRunWebSearchPOSTTemplateProvider(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":["one","two"]}`))
	}))
	defer server.Close()

	enabled := true
	out, err := runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "custom",
		Providers: []config.WebSearchProviderConfig{{
			Name:         "custom",
			Type:         "http",
			Method:       "POST",
			Endpoint:     server.URL,
			BodyTemplate: `{"q":"{query}","limit":{limit}}`,
			ResultsPath:  "items",
		}},
	}, webSearchInput{Query: "hello", Limit: 2})
	if err != nil {
		t.Fatalf("run web search: %v", err)
	}
	if body != `{"q":"hello","limit":2}` {
		t.Fatalf("unexpected body %q", body)
	}
	if len(out.Results) != 2 || out.Results[0] != "one" || out.Results[1] != "two" {
		t.Fatalf("unexpected output: %#v", out)
	}
}
