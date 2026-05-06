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
		DefaultProvider: "tinyfish",
		Providers: []config.WebSearchProviderConfig{{
			Name:        "tinyfish",
			Type:        "tinyfish",
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
	if out.Provider != "tinyfish" || len(out.Results) != 1 || !strings.Contains(out.Results[0], "TinyFish result") {
		t.Fatalf("unexpected output: %#v", out)
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
