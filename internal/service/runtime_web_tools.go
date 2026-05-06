package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	toolutils "github.com/cloudwego/eino/components/tool/utils"

	"starxo/internal/config"
	"starxo/internal/tools"
)

type webFetchInput struct {
	URL     string `json:"url" jsonschema:"description=URL to fetch with HTTP GET"`
	Limit   int    `json:"limit,omitempty" jsonschema:"description=max response bytes; default 20000"`
	Timeout int    `json:"timeout_ms,omitempty" jsonschema:"description=request timeout in milliseconds"`
}

type webFetchOutput struct {
	URL        string `json:"url"`
	StatusCode int    `json:"statusCode"`
	Content    string `json:"content"`
	Truncated  bool   `json:"truncated"`
}

type webSearchInput struct {
	Query    string `json:"query" jsonschema:"description=search query"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=max result lines"`
	Provider string `json:"provider,omitempty" jsonschema:"description=optional configured provider name; defaults to settings agent.webSearch.defaultProvider"`
	Location string `json:"location,omitempty" jsonschema:"description=optional country code for providers that support geo-targeting, for example US or GB"`
	Language string `json:"language,omitempty" jsonschema:"description=optional language code for providers that support language targeting, for example en or fr"`
	Page     int    `json:"page,omitempty" jsonschema:"description=optional zero-based page number for providers that support pagination"`
}

type webSearchOutput struct {
	Query    string   `json:"query"`
	Provider string   `json:"provider"`
	URL      string   `json:"url"`
	Results  []string `json:"results"`
}

func newRuntimeWebCatalogEntries(cfg config.WebSearchConfig) ([]tools.CatalogEntry, error) {
	fetch, err := toolutils.InferTool(tools.RuntimeToolWebFetch,
		"Fetch a URL with HTTP GET and return compact text.",
		func(ctx context.Context, input webFetchInput) (webFetchOutput, error) {
			return runWebFetch(ctx, input)
		})
	if err != nil {
		return nil, err
	}
	search, err := toolutils.InferTool(tools.RuntimeToolWebSearch,
		"Search the web using the configured WebSearch provider and return compact results.",
		func(ctx context.Context, input webSearchInput) (webSearchOutput, error) {
			return runWebSearch(ctx, cfg, input)
		})
	if err != nil {
		return nil, err
	}
	return []tools.CatalogEntry{
		tools.RuntimeWebFetchCatalogEntry(fetch),
		tools.RuntimeWebSearchCatalogEntry(search),
	}, nil
}

func runWebFetch(ctx context.Context, input webFetchInput) (webFetchOutput, error) {
	if strings.TrimSpace(input.URL) == "" {
		return webFetchOutput{}, fmt.Errorf("url is required")
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 20000
	}
	timeout := time.Duration(input.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, input.URL, nil)
	if err != nil {
		return webFetchOutput{}, err
	}
	req.Header.Set("User-Agent", "Starxo/RuntimeV2")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return webFetchOutput{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit+1)))
	if err != nil {
		return webFetchOutput{}, err
	}
	truncated := len(data) > limit
	if truncated {
		data = data[:limit]
	}
	return webFetchOutput{
		URL:        input.URL,
		StatusCode: resp.StatusCode,
		Content:    compactHTML(string(data)),
		Truncated:  truncated,
	}, nil
}

func runWebSearch(ctx context.Context, cfg config.WebSearchConfig, input webSearchInput) (webSearchOutput, error) {
	if strings.TrimSpace(input.Query) == "" {
		return webSearchOutput{}, fmt.Errorf("query is required")
	}
	if cfg.Enabled != nil && !*cfg.Enabled {
		return webSearchOutput{}, fmt.Errorf("web search is disabled in settings")
	}
	providerName := strings.TrimSpace(input.Provider)
	if providerName == "" {
		providerName = strings.TrimSpace(cfg.DefaultProvider)
	}
	if providerName == "" {
		providerName = "duckduckgo"
	}
	provider, ok := resolveWebSearchProvider(cfg, providerName)
	if ok && provider.Disabled {
		return webSearchOutput{}, fmt.Errorf("web search provider %q is disabled", providerName)
	}
	if ok {
		return runConfiguredWebSearch(ctx, provider, input)
	}
	if strings.EqualFold(providerName, "tinyfish") {
		return runConfiguredWebSearch(ctx, config.WebSearchProviderConfig{Name: "tinyfish", Type: "tinyfish"}, input)
	}
	if !ok && !strings.EqualFold(providerName, "duckduckgo") {
		return webSearchOutput{}, fmt.Errorf("web search provider %q is not configured", providerName)
	}
	return runDuckDuckGoWebSearch(ctx, input)
}

func runDuckDuckGoWebSearch(ctx context.Context, input webSearchInput) (webSearchOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 8
	}
	searchURL := tools.WebSearchURL(input.Query)
	fetched, err := runWebFetch(ctx, webFetchInput{URL: searchURL, Limit: 60000, Timeout: 20000})
	if err != nil {
		return webSearchOutput{}, err
	}
	lines := nonEmptyRuntimeLines(fetched.Content)
	results := make([]string, 0, limit)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(strings.ToLower(line), "duckduckgo") {
			continue
		}
		results = append(results, line)
		if len(results) >= limit {
			break
		}
	}
	return webSearchOutput{Query: input.Query, Provider: "duckduckgo", URL: searchURL, Results: results}, nil
}

func resolveWebSearchProvider(cfg config.WebSearchConfig, name string) (config.WebSearchProviderConfig, bool) {
	for _, provider := range cfg.Providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name), strings.TrimSpace(name)) {
			return provider, true
		}
	}
	return config.WebSearchProviderConfig{}, false
}

func runConfiguredWebSearch(ctx context.Context, provider config.WebSearchProviderConfig, input webSearchInput) (webSearchOutput, error) {
	providerType := strings.ToLower(strings.TrimSpace(provider.Type))
	if providerType == "" {
		providerType = "http"
	}
	if providerType == "duckduckgo" {
		return runDuckDuckGoWebSearch(ctx, input)
	}
	if providerType == "tinyfish" {
		return runTinyFishWebSearch(ctx, provider, input)
	}
	if providerType != "http" {
		return webSearchOutput{}, fmt.Errorf("unsupported web search provider type %q", provider.Type)
	}
	if strings.TrimSpace(provider.Endpoint) == "" {
		return webSearchOutput{}, fmt.Errorf("web search provider %s endpoint is required", provider.Name)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = provider.MaxResults
	}
	if limit <= 0 {
		limit = 8
	}

	method := strings.ToUpper(strings.TrimSpace(provider.Method))
	if method == "" {
		method = http.MethodGet
	}
	timeout := time.Duration(provider.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	endpoint := expandWebSearchTemplate(provider.Endpoint, input.Query, limit)
	var body io.Reader
	if method != http.MethodGet {
		bodyText := provider.BodyTemplate
		if strings.TrimSpace(bodyText) == "" {
			payload, _ := json.Marshal(map[string]any{"query": input.Query, "limit": limit})
			bodyText = string(payload)
		}
		body = bytes.NewBufferString(expandWebSearchTemplate(bodyText, input.Query, limit))
	}
	if method == http.MethodGet {
		endpoint = appendWebSearchQuery(endpoint, provider, input.Query, limit)
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, method, endpoint, body)
	if err != nil {
		return webSearchOutput{}, err
	}
	req.Header.Set("User-Agent", "Starxo/RuntimeV2")
	if method != http.MethodGet && strings.TrimSpace(provider.BodyTemplate) == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range provider.Headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, expandWebSearchTemplate(v, input.Query, limit))
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return webSearchOutput{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return webSearchOutput{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(compactHTML(string(data)))
		if len(message) > 500 {
			message = message[:500]
		}
		return webSearchOutput{}, fmt.Errorf("web search provider %s returned HTTP %d: %s", provider.Name, resp.StatusCode, message)
	}
	results := parseConfiguredWebSearchResults(data, provider, limit)
	if len(results) == 0 {
		results = fallbackWebSearchLines(string(data), limit)
	}
	return webSearchOutput{
		Query:    input.Query,
		Provider: provider.Name,
		URL:      endpoint,
		Results:  results,
	}, nil
}

func runTinyFishWebSearch(ctx context.Context, provider config.WebSearchProviderConfig, input webSearchInput) (webSearchOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = provider.MaxResults
	}
	if limit <= 0 {
		limit = 8
	}
	page := input.Page
	if page == 0 && provider.Page > 0 {
		page = provider.Page
	}
	if page < 0 || page > 10 {
		return webSearchOutput{}, fmt.Errorf("tinyfish search page must be between 0 and 10")
	}

	endpoint := strings.TrimSpace(provider.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.search.tinyfish.ai"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return webSearchOutput{}, err
	}
	values := u.Query()
	values.Set("query", input.Query)
	location := firstNonEmpty(input.Location, provider.Location)
	if location != "" {
		values.Set("location", location)
	}
	language := firstNonEmpty(input.Language, provider.Language)
	if language != "" {
		values.Set("language", language)
	}
	if page > 0 {
		values.Set("page", strconv.Itoa(page))
	}
	u.RawQuery = values.Encode()

	timeout := time.Duration(provider.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u.String(), nil)
	if err != nil {
		return webSearchOutput{}, err
	}
	req.Header.Set("User-Agent", "Starxo/RuntimeV2")
	hasAPIKeyHeader := false
	for k, v := range provider.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		if strings.EqualFold(k, "X-API-Key") {
			hasAPIKeyHeader = true
		}
		req.Header.Set(k, expandWebSearchTemplate(v, input.Query, limit))
	}
	if !hasAPIKeyHeader {
		apiKeyEnv := strings.TrimSpace(provider.APIKeyEnv)
		if apiKeyEnv == "" {
			apiKeyEnv = "TINYFISH_API_KEY"
		}
		apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
		if apiKey == "" {
			return webSearchOutput{}, fmt.Errorf("tinyfish search requires %s to be set or X-API-Key to be configured", apiKeyEnv)
		}
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return webSearchOutput{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return webSearchOutput{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(compactHTML(string(data)))
		if len(message) > 500 {
			message = message[:500]
		}
		return webSearchOutput{}, fmt.Errorf("tinyfish search returned HTTP %d: %s", resp.StatusCode, message)
	}

	resultProvider := provider
	if strings.TrimSpace(resultProvider.ResultsPath) == "" {
		resultProvider.ResultsPath = "results"
	}
	if strings.TrimSpace(resultProvider.TitlePath) == "" {
		resultProvider.TitlePath = "title"
	}
	if strings.TrimSpace(resultProvider.URLPath) == "" {
		resultProvider.URLPath = "url"
	}
	if strings.TrimSpace(resultProvider.SnippetPath) == "" {
		resultProvider.SnippetPath = "snippet"
	}
	results := parseConfiguredWebSearchResults(data, resultProvider, limit)
	if len(results) == 0 {
		results = fallbackWebSearchLines(string(data), limit)
	}
	providerName := strings.TrimSpace(provider.Name)
	if providerName == "" {
		providerName = "tinyfish"
	}
	return webSearchOutput{
		Query:    input.Query,
		Provider: providerName,
		URL:      u.String(),
		Results:  results,
	}, nil
}

func appendWebSearchQuery(endpoint string, provider config.WebSearchProviderConfig, query string, limit int) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	values := u.Query()
	queryParam := provider.QueryParam
	if queryParam == "" {
		queryParam = "q"
	}
	limitParam := provider.LimitParam
	if limitParam == "" {
		limitParam = "limit"
	}
	if !hasWebSearchTemplatePlaceholder(endpoint, "query") && queryParam != "-" && values.Get(queryParam) == "" {
		values.Set(queryParam, query)
	}
	if !hasWebSearchTemplatePlaceholder(endpoint, "limit") && limitParam != "-" && values.Get(limitParam) == "" {
		values.Set(limitParam, strconv.Itoa(limit))
	}
	u.RawQuery = values.Encode()
	return u.String()
}

func expandWebSearchTemplate(s string, query string, limit int) string {
	queryJSON, _ := json.Marshal(query)
	replacer := strings.NewReplacer(
		"{query}", query,
		"{{query}}", query,
		"{query_json}", string(queryJSON),
		"{{query_json}}", string(queryJSON),
		"{query_url}", url.QueryEscape(query),
		"{{query_url}}", url.QueryEscape(query),
		"{limit}", strconv.Itoa(limit),
		"{{limit}}", strconv.Itoa(limit),
	)
	return replacer.Replace(s)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func hasWebSearchTemplatePlaceholder(s string, name string) bool {
	return strings.Contains(s, "{"+name+"}") ||
		strings.Contains(s, "{{"+name+"}}") ||
		strings.Contains(s, "{"+name+"_url}") ||
		strings.Contains(s, "{{"+name+"_url}}") ||
		strings.Contains(s, "{"+name+"_json}") ||
		strings.Contains(s, "{{"+name+"_json}}")
}

func parseConfiguredWebSearchResults(data []byte, provider config.WebSearchProviderConfig, limit int) []string {
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil
	}
	var candidates []any
	if provider.ResultsPath != "" {
		candidates = extractWebSearchCandidates(decoded, provider.ResultsPath)
	} else {
		for _, path := range []string{"results", "data.results", "data", "items", "organic_results"} {
			candidates = extractWebSearchCandidates(decoded, path)
			if len(candidates) > 0 {
				break
			}
		}
		if len(candidates) == 0 {
			candidates = extractWebSearchCandidates(decoded, "")
		}
	}
	results := make([]string, 0, min(limit, len(candidates)))
	for _, candidate := range candidates {
		line := formatWebSearchCandidate(candidate, provider)
		if strings.TrimSpace(line) == "" {
			continue
		}
		results = append(results, line)
		if len(results) >= limit {
			break
		}
	}
	return results
}

func extractWebSearchCandidates(decoded any, path string) []any {
	target := decoded
	if strings.TrimSpace(path) != "" {
		var ok bool
		target, ok = valueAtDottedPath(decoded, path)
		if !ok {
			return nil
		}
	}
	switch v := target.(type) {
	case []any:
		return v
	case map[string]any:
		return []any{v}
	default:
		return []any{v}
	}
}

func formatWebSearchCandidate(candidate any, provider config.WebSearchProviderConfig) string {
	switch v := candidate.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		title := firstWebSearchString(v, provider.TitlePath, "title", "name")
		link := firstWebSearchString(v, provider.URLPath, "url", "link", "href")
		snippet := firstWebSearchString(v, provider.SnippetPath, "snippet", "content", "description", "text")
		parts := make([]string, 0, 3)
		if title != "" {
			parts = append(parts, title)
		}
		if link != "" {
			parts = append(parts, link)
		}
		if snippet != "" {
			parts = append(parts, snippet)
		}
		return strings.Join(parts, " - ")
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func firstWebSearchString(v map[string]any, paths ...string) string {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if value, ok := valueAtDottedPath(v, path); ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func valueAtDottedPath(v any, path string) (any, bool) {
	current := v
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func fallbackWebSearchLines(content string, limit int) []string {
	lines := nonEmptyRuntimeLines(compactHTML(content))
	results := make([]string, 0, min(limit, len(lines)))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		results = append(results, line)
		if len(results) >= limit {
			break
		}
	}
	return results
}

var htmlTagRE = regexp.MustCompile(`<[^>]+>`)
var htmlSpaceRE = regexp.MustCompile(`\s+`)

func compactHTML(s string) string {
	s = htmlTagRE.ReplaceAllString(s, "\n")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	lines := nonEmptyRuntimeLines(s)
	return strings.Join(lines, "\n")
}

func nonEmptyRuntimeLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(htmlSpaceRE.ReplaceAllString(line, " "))
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
