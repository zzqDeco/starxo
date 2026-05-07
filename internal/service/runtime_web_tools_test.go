package service

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"starxo/internal/config"
	"starxo/internal/tools"
)

type recordingRuntimeWebPermissionProvider struct {
	decisions    []string
	entries      []tools.CatalogEntry
	inputs       []runtimeWebEndpointPermissionInput
	rawInputs    []string
	deadlineSeen []bool
}

func (p *recordingRuntimeWebPermissionProvider) RequestToolPermission(ctx context.Context, entry tools.CatalogEntry, argumentsInJSON string) (tools.ToolPermissionResolution, error) {
	p.entries = append(p.entries, entry)
	p.rawInputs = append(p.rawInputs, argumentsInJSON)
	_, hasDeadline := ctx.Deadline()
	p.deadlineSeen = append(p.deadlineSeen, hasDeadline)
	var input runtimeWebEndpointPermissionInput
	_ = json.Unmarshal([]byte(argumentsInJSON), &input)
	p.inputs = append(p.inputs, input)
	decision := tools.ToolPermissionDecisionAllowOnce
	if len(p.decisions) > 0 {
		decision = p.decisions[0]
		p.decisions = p.decisions[1:]
	}
	return tools.ToolPermissionResolution{Decision: decision}, nil
}

func TestRunWebFetchPermissionUsesOuterContextNotRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("allowed"))
	}))
	defer server.Close()

	provider := &recordingRuntimeWebPermissionProvider{}
	_, err := runWebFetchWithGuard(t.Context(), webFetchInput{URL: server.URL, Timeout: 1000}, allowRuntimeWebGuard(provider))
	if err != nil {
		t.Fatalf("run web fetch: %v", err)
	}
	if len(provider.deadlineSeen) != 1 {
		t.Fatalf("expected one permission request, got %#v", provider.deadlineSeen)
	}
	if provider.deadlineSeen[0] {
		t.Fatalf("permission prompt should use the outer run context, not the HTTP request timeout")
	}
}

func allowRuntimeWebGuard(provider *recordingRuntimeWebPermissionProvider) *runtimeWebEndpointGuard {
	if provider == nil {
		provider = &recordingRuntimeWebPermissionProvider{}
	}
	return newRuntimeWebEndpointGuard(provider)
}

func TestRunWebSearchUsesConfiguredHTTPProvider(t *testing.T) {
	var gotQuery string
	var gotLimit string
	permissionProvider := &recordingRuntimeWebPermissionProvider{}
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
	out, err := runWebSearchWithGuard(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "custom",
		Providers: []config.WebSearchProviderConfig{{
			Name:        "custom",
			Type:        "http",
			Endpoint:    server.URL + "/search",
			QueryParam:  "q",
			LimitParam:  "n",
			Headers:     map[string]string{"X-API-Key": "secret-provider-key"},
			ResultsPath: "data.results",
			TitlePath:   "title",
			URLPath:     "url",
			SnippetPath: "content",
		}},
	}, webSearchInput{Query: "starxo", Limit: 3}, allowRuntimeWebGuard(permissionProvider))
	if err != nil {
		t.Fatalf("run web search: %v", err)
	}
	if gotQuery != "starxo" || gotLimit != "3" {
		t.Fatalf("unexpected query params q=%q n=%q", gotQuery, gotLimit)
	}
	if out.Provider != "custom" || len(out.Results) != 1 || !strings.Contains(out.Results[0], "TinyFish result") {
		t.Fatalf("unexpected output: %#v", out)
	}
	if len(permissionProvider.inputs) != 1 || permissionProvider.entries[0].CanonicalName != tools.RuntimeToolWebSearch {
		t.Fatalf("expected one WebSearch endpoint permission request, got entries=%#v inputs=%#v", permissionProvider.entries, permissionProvider.inputs)
	}
	if permissionProvider.inputs[0].URL == "" || permissionProvider.inputs[0].Host == "" || permissionProvider.inputs[0].Reason == "" {
		t.Fatalf("permission input must include url/host/reason, got %#v", permissionProvider.inputs[0])
	}
	if strings.Contains(permissionProvider.rawInputs[0], "secret-provider-key") {
		t.Fatalf("permission input leaked provider header value: %s", permissionProvider.rawInputs[0])
	}
}

func TestRunWebSearchUsesTinyFishProviderContract(t *testing.T) {
	t.Setenv("STARXO_TINYFISH_TEST_KEY", "tf-test-key")

	permissionProvider := &recordingRuntimeWebPermissionProvider{}
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
	out, err := runWebSearchWithGuard(t.Context(), config.WebSearchConfig{
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
	}, webSearchInput{Query: "starxo", Limit: 1, Page: 2}, allowRuntimeWebGuard(permissionProvider))
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
	if len(permissionProvider.rawInputs) != 1 || strings.Contains(permissionProvider.rawInputs[0], "tf-test-key") {
		t.Fatalf("permission input should omit TinyFish API key, got %#v", permissionProvider.rawInputs)
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

func TestRunWebSearchTinyFishLive(t *testing.T) {
	if strings.TrimSpace(os.Getenv("TINYFISH_API_KEY")) == "" {
		t.Skip("TINYFISH_API_KEY is not set")
	}

	enabled := true
	out, err := runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "tinyfish",
		Providers: []config.WebSearchProviderConfig{{
			Name:     "tinyfish",
			Type:     "tinyfish",
			Location: "US",
			Language: "en",
		}},
	}, webSearchInput{Query: "OpenAI API documentation", Limit: 2})
	if err != nil {
		t.Fatalf("run live tinyfish search: %v", err)
	}
	if out.Provider != "tinyfish" {
		t.Fatalf("unexpected provider %q", out.Provider)
	}
	if len(out.Results) == 0 {
		t.Fatalf("expected live tinyfish results, got %#v", out)
	}
	if strings.Contains(out.URL, "limit=") || !strings.Contains(out.URL, "query=") {
		t.Fatalf("unexpected tinyfish request URL %q", out.URL)
	}
}

func TestRunWebSearchPOSTTemplateProvider(t *testing.T) {
	var body string
	permissionProvider := &recordingRuntimeWebPermissionProvider{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":["one","two"]}`))
	}))
	defer server.Close()

	enabled := true
	out, err := runWebSearchWithGuard(t.Context(), config.WebSearchConfig{
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
	}, webSearchInput{Query: "hello", Limit: 2}, allowRuntimeWebGuard(permissionProvider))
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

func TestRunWebFetchRejectsUnsafeURLs(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr string
	}{
		{name: "unsafe scheme", rawURL: "file:///etc/passwd", wantErr: "only http and https"},
		{name: "empty host", rawURL: "https:///missing-host", wantErr: "host is required"},
		{name: "userinfo", rawURL: "https://user:pass@example.com", wantErr: "userinfo is not allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &recordingRuntimeWebPermissionProvider{}
			_, err := runWebFetchWithGuard(t.Context(), webFetchInput{URL: tt.rawURL}, allowRuntimeWebGuard(provider))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected %q error, got %v", tt.wantErr, err)
			}
			if len(provider.inputs) != 0 {
				t.Fatalf("unsafe URL should fail before permission request, got %#v", provider.inputs)
			}
		})
	}
}

func TestRunWebFetchBlocksNonPublicHostsByDefault(t *testing.T) {
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte("blocked"))
	}))
	defer server.Close()

	_, err := runWebFetch(t.Context(), webFetchInput{URL: server.URL})
	if err == nil || !strings.Contains(err.Error(), "permission provider is unavailable") {
		t.Fatalf("expected missing permission provider error, got %v", err)
	}
	if hit {
		t.Fatalf("non-public endpoint was contacted without permission")
	}

	for _, rawURL := range []string{
		"http://localhost:1/",
		"http://10.0.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/",
		"http://[fc00::1]/",
	} {
		t.Run(rawURL, func(t *testing.T) {
			_, err := runWebFetch(t.Context(), webFetchInput{URL: rawURL, Timeout: 100})
			if err == nil || !strings.Contains(err.Error(), "blocked") {
				t.Fatalf("expected non-public host to be blocked, got %v", err)
			}
		})
	}
}

func TestRunWebFetchValidatesDialedAddress(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("rebound target"))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	_, port, err := net.SplitHostPort(serverURL.Host)
	if err != nil {
		t.Fatalf("parse test server host: %v", err)
	}
	targetURL := "http://rebind.test:" + port + "/"

	resolverCalls := 0
	rebindResolver := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		resolverCalls++
		if resolverCalls == 1 {
			return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
		}
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	}

	blockGuard := newRuntimeWebEndpointGuard(nil)
	blockGuard.resolveIPAddrs = rebindResolver
	_, err = runWebFetchWithGuard(t.Context(), webFetchInput{URL: targetURL}, blockGuard)
	if err == nil || !strings.Contains(err.Error(), "permission provider is unavailable") {
		t.Fatalf("expected dialed private address to require permission, got %v", err)
	}
	if hits != 0 {
		t.Fatalf("rebound endpoint should not be contacted without permission, hits=%d", hits)
	}

	resolverCalls = 0
	allowProvider := &recordingRuntimeWebPermissionProvider{}
	allowGuard := newRuntimeWebEndpointGuard(allowProvider)
	allowGuard.resolveIPAddrs = rebindResolver
	out, err := runWebFetchWithGuard(t.Context(), webFetchInput{URL: targetURL}, allowGuard)
	if err != nil {
		t.Fatalf("run web fetch after approving dial target: %v", err)
	}
	if hits != 1 || !strings.Contains(out.Content, "rebound target") {
		t.Fatalf("expected approved rebound fetch, hits=%d output=%#v", hits, out)
	}
	if len(allowProvider.inputs) != 1 || !strings.Contains(allowProvider.inputs[0].Reason, "dial target is loopback") {
		t.Fatalf("expected one dial-target permission request, got %#v", allowProvider.inputs)
	}
}

func TestRunWebFetchNonPublicPermissionAllowDeny(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("<p>allowed content</p>"))
	}))
	defer server.Close()

	denyProvider := &recordingRuntimeWebPermissionProvider{decisions: []string{tools.ToolPermissionDecisionDeny}}
	_, err := runWebFetchWithGuard(t.Context(), webFetchInput{URL: server.URL}, allowRuntimeWebGuard(denyProvider))
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denied permission error, got %v", err)
	}
	if hits != 0 {
		t.Fatalf("denied endpoint should not be contacted, hits=%d", hits)
	}

	allowProvider := &recordingRuntimeWebPermissionProvider{}
	out, err := runWebFetchWithGuard(t.Context(), webFetchInput{URL: server.URL}, allowRuntimeWebGuard(allowProvider))
	if err != nil {
		t.Fatalf("run web fetch with allow: %v", err)
	}
	if hits != 1 || !strings.Contains(out.Content, "allowed content") {
		t.Fatalf("expected one allowed fetch, hits=%d output=%#v", hits, out)
	}
	if len(allowProvider.inputs) != 1 || allowProvider.inputs[0].URL == "" || allowProvider.inputs[0].Host == "" || allowProvider.inputs[0].Reason == "" {
		t.Fatalf("expected sanitized permission input with url/host/reason, got %#v", allowProvider.inputs)
	}
	if allowProvider.entries[0].CanonicalName != tools.RuntimeToolWebFetch {
		t.Fatalf("expected WebFetch permission entry, got %#v", allowProvider.entries[0])
	}
}

func TestRunWebFetchValidatesRedirectTarget(t *testing.T) {
	targetHits := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHits++
		_, _ = w.Write([]byte("target"))
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/private", http.StatusFound)
	}))
	defer redirector.Close()

	provider := &recordingRuntimeWebPermissionProvider{
		decisions: []string{tools.ToolPermissionDecisionAllowOnce, tools.ToolPermissionDecisionDeny},
	}
	_, err := runWebFetchWithGuard(t.Context(), webFetchInput{URL: redirector.URL}, allowRuntimeWebGuard(provider))
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected redirect target denial, got %v", err)
	}
	if targetHits != 0 {
		t.Fatalf("redirect target should not be contacted after denial, hits=%d", targetHits)
	}
	if len(provider.inputs) != 2 || !strings.Contains(provider.inputs[1].URL, "/private") {
		t.Fatalf("expected permission requests for initial and redirect URLs, got %#v", provider.inputs)
	}
}

func TestRunWebSearchProviderEndpointValidation(t *testing.T) {
	enabled := true
	_, err := runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "custom",
		Providers: []config.WebSearchProviderConfig{{
			Name:     "custom",
			Type:     "http",
			Endpoint: "file:///tmp/search",
		}},
	}, webSearchInput{Query: "starxo"})
	if err == nil || !strings.Contains(err.Error(), "only http and https") {
		t.Fatalf("expected custom provider scheme validation error, got %v", err)
	}

	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	_, err = runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "custom",
		Providers: []config.WebSearchProviderConfig{{
			Name:        "custom",
			Type:        "http",
			Endpoint:    server.URL,
			ResultsPath: "items",
		}},
	}, webSearchInput{Query: "starxo"})
	if err == nil || !strings.Contains(err.Error(), "permission provider is unavailable") {
		t.Fatalf("expected custom provider non-public endpoint block, got %v", err)
	}
	if hit {
		t.Fatalf("custom provider endpoint was contacted without permission")
	}

	_, err = runWebSearch(t.Context(), config.WebSearchConfig{
		Enabled:         &enabled,
		DefaultProvider: "tinyfish",
		Providers: []config.WebSearchProviderConfig{{
			Name:     "tinyfish",
			Type:     "tinyfish",
			Endpoint: "http://127.0.0.1:9",
			Headers:  map[string]string{"X-API-Key": "secret-tinyfish-key"},
		}},
	}, webSearchInput{Query: "starxo"})
	if err == nil || !strings.Contains(err.Error(), "permission provider is unavailable") {
		t.Fatalf("expected TinyFish non-public endpoint block, got %v", err)
	}
}
