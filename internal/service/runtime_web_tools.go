package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

type runtimeWebHostResolver func(ctx context.Context, host string) ([]net.IPAddr, error)

type runtimeWebEndpointGuard struct {
	permissionProvider tools.ToolExecutionPermissionProvider
	resolveIPAddrs     runtimeWebHostResolver
}

type runtimeWebRequestGuard struct {
	base          *runtimeWebEndpointGuard
	toolName      string
	permissionCtx context.Context
	permissionMu  *sync.Mutex
	approved      map[string]struct{}
}

type runtimeWebEndpointDecision struct {
	URL     string
	Host    string
	Port    string
	Address string
	Reason  string
}

type runtimeWebEndpointPermissionInput struct {
	URL    string `json:"url"`
	Host   string `json:"host"`
	Reason string `json:"reason"`
}

var runtimeWebMetadataAddr = netip.MustParseAddr("169.254.169.254")

func newRuntimeWebEndpointGuard(permissionProvider tools.ToolExecutionPermissionProvider) *runtimeWebEndpointGuard {
	resolver := net.DefaultResolver
	return &runtimeWebEndpointGuard{
		permissionProvider: permissionProvider,
		resolveIPAddrs: func(ctx context.Context, host string) ([]net.IPAddr, error) {
			return resolver.LookupIPAddr(ctx, host)
		},
	}
}

func (g *runtimeWebEndpointGuard) requestGuard(toolName string) *runtimeWebRequestGuard {
	if g == nil {
		g = newRuntimeWebEndpointGuard(nil)
	}
	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		toolName = tools.RuntimeToolWebFetch
	}
	return &runtimeWebRequestGuard{
		base:         g,
		toolName:     toolName,
		permissionMu: &sync.Mutex{},
		approved:     make(map[string]struct{}),
	}
}

func (g *runtimeWebRequestGuard) withPermissionContext(ctx context.Context) *runtimeWebRequestGuard {
	if g == nil {
		return g
	}
	next := *g
	next.permissionCtx = ctx
	return &next
}

func newRuntimeWebCatalogEntries(cfg config.WebSearchConfig, permissionProvider tools.ToolExecutionPermissionProvider) ([]tools.CatalogEntry, error) {
	guard := newRuntimeWebEndpointGuard(permissionProvider)
	fetch, err := toolutils.InferTool(tools.RuntimeToolWebFetch,
		"Fetch a URL with HTTP GET and return compact text.",
		func(ctx context.Context, input webFetchInput) (webFetchOutput, error) {
			return runWebFetchWithGuard(ctx, input, guard)
		})
	if err != nil {
		return nil, err
	}
	search, err := toolutils.InferTool(tools.RuntimeToolWebSearch,
		"Search the web using the configured WebSearch provider and return compact results.",
		func(ctx context.Context, input webSearchInput) (webSearchOutput, error) {
			return runWebSearchWithGuard(ctx, cfg, input, guard)
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
	return runWebFetchWithGuard(ctx, input, nil)
}

func runWebFetchWithGuard(ctx context.Context, input webFetchInput, guard *runtimeWebEndpointGuard) (webFetchOutput, error) {
	return runWebFetchWithGuardForTool(ctx, input, guard, tools.RuntimeToolWebFetch)
}

func runWebFetchWithGuardForTool(ctx context.Context, input webFetchInput, guard *runtimeWebEndpointGuard, toolName string) (webFetchOutput, error) {
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
	requestGuard := guard.requestGuard(toolName).withPermissionContext(ctx)
	if err := requestGuard.ensureURLAllowed(ctx, input.URL); err != nil {
		return webFetchOutput{}, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, input.URL, nil)
	if err != nil {
		return webFetchOutput{}, err
	}
	req.Header.Set("User-Agent", "Starxo/RuntimeV2")
	resp, err := requestGuard.httpClient().Do(req)
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
	return runWebSearchWithGuard(ctx, cfg, input, nil)
}

func runWebSearchWithGuard(ctx context.Context, cfg config.WebSearchConfig, input webSearchInput, guard *runtimeWebEndpointGuard) (webSearchOutput, error) {
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
		return runConfiguredWebSearch(ctx, provider, input, guard)
	}
	if strings.EqualFold(providerName, "tinyfish") {
		return runConfiguredWebSearch(ctx, config.WebSearchProviderConfig{Name: "tinyfish", Type: "tinyfish"}, input, guard)
	}
	if !ok && !strings.EqualFold(providerName, "duckduckgo") {
		return webSearchOutput{}, fmt.Errorf("web search provider %q is not configured", providerName)
	}
	return runDuckDuckGoWebSearch(ctx, input, guard)
}

func runDuckDuckGoWebSearch(ctx context.Context, input webSearchInput, guard *runtimeWebEndpointGuard) (webSearchOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 8
	}
	searchURL := tools.WebSearchURL(input.Query)
	fetched, err := runWebFetchWithGuardForTool(ctx, webFetchInput{URL: searchURL, Limit: 60000, Timeout: 20000}, guard, tools.RuntimeToolWebSearch)
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

func runConfiguredWebSearch(ctx context.Context, provider config.WebSearchProviderConfig, input webSearchInput, guard *runtimeWebEndpointGuard) (webSearchOutput, error) {
	providerType := strings.ToLower(strings.TrimSpace(provider.Type))
	if providerType == "" {
		providerType = "http"
	}
	if providerType == "duckduckgo" {
		return runDuckDuckGoWebSearch(ctx, input, guard)
	}
	if providerType == "tinyfish" {
		return runTinyFishWebSearch(ctx, provider, input, guard)
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

	requestGuard := guard.requestGuard(tools.RuntimeToolWebSearch).withPermissionContext(ctx)
	if err := requestGuard.ensureURLAllowed(ctx, endpoint); err != nil {
		return webSearchOutput{}, err
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
	resp, err := requestGuard.httpClient().Do(req)
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

func runTinyFishWebSearch(ctx context.Context, provider config.WebSearchProviderConfig, input webSearchInput, guard *runtimeWebEndpointGuard) (webSearchOutput, error) {
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
	hasAPIKeyHeader := false
	for k := range provider.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		if strings.EqualFold(k, "X-API-Key") {
			hasAPIKeyHeader = true
		}
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
	}

	requestGuard := guard.requestGuard(tools.RuntimeToolWebSearch).withPermissionContext(ctx)
	if err := requestGuard.ensureURLAllowed(ctx, u.String()); err != nil {
		return webSearchOutput{}, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u.String(), nil)
	if err != nil {
		return webSearchOutput{}, err
	}
	req.Header.Set("User-Agent", "Starxo/RuntimeV2")
	for k, v := range provider.Headers {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, expandWebSearchTemplate(v, input.Query, limit))
		}
	}
	if !hasAPIKeyHeader {
		apiKeyEnv := strings.TrimSpace(provider.APIKeyEnv)
		if apiKeyEnv == "" {
			apiKeyEnv = "TINYFISH_API_KEY"
		}
		req.Header.Set("X-API-Key", strings.TrimSpace(os.Getenv(apiKeyEnv)))
	}

	resp, err := requestGuard.httpClient().Do(req)
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

func (g *runtimeWebRequestGuard) httpClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
		return g.dialContext(ctx, dialer, network, address)
	}
	return &http.Client{
		Transport: runtimeWebRequestTransport{base: transport},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return g.ensureURLAllowed(g.permissionContext(req.Context()), req.URL.String())
		},
	}
}

type runtimeWebRequestURLContextKey struct{}

type runtimeWebRequestTransport struct {
	base http.RoundTripper
}

func (t runtimeWebRequestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	ctx := context.WithValue(req.Context(), runtimeWebRequestURLContextKey{}, req.URL.String())
	return base.RoundTrip(req.WithContext(ctx))
}

func (g *runtimeWebRequestGuard) permissionContext(fallback context.Context) context.Context {
	if g != nil && g.permissionCtx != nil {
		return g.permissionCtx
	}
	return fallback
}

func (g *runtimeWebRequestGuard) ensureURLAllowed(ctx context.Context, rawURL string) error {
	if g == nil || g.base == nil {
		g = (&runtimeWebEndpointGuard{}).requestGuard(tools.RuntimeToolWebFetch)
	}
	decision, err := g.base.inspectURL(ctx, rawURL)
	if err != nil {
		return err
	}
	if decision.Reason == "" {
		return nil
	}
	return g.ensureDecisionAllowed(ctx, decision)
}

func (g *runtimeWebRequestGuard) dialContext(ctx context.Context, dialer *net.Dialer, network string, address string) (net.Conn, error) {
	if g == nil || g.base == nil {
		g = (&runtimeWebEndpointGuard{}).requestGuard(tools.RuntimeToolWebFetch)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parse web dial address %q: %w", address, err)
	}
	if strings.ContainsAny(host, "\x00\r\n\t ") {
		return nil, fmt.Errorf("web dial host %q is invalid", host)
	}
	resolved, err := g.base.resolveHostAddrs(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("resolve web dial host %q: no usable addresses found", host)
	}

	var lastErr error
	for _, candidate := range resolved {
		if reason := runtimeWebNonPublicAddrReason(candidate.addr); reason != "" {
			decision := runtimeWebEndpointDecision{
				URL:     runtimeWebRequestURLFromContext(ctx, address),
				Host:    host,
				Port:    port,
				Address: candidate.addr.String(),
				Reason:  "dial target is " + reason,
			}
			if err := g.ensureDecisionAllowed(g.permissionContext(ctx), decision); err != nil {
				return nil, err
			}
		}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.dialHost(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("dial validated web host %q: %w", host, lastErr)
	}
	return nil, fmt.Errorf("dial validated web host %q: no address was dialed", host)
}

func runtimeWebRequestURLFromContext(ctx context.Context, address string) string {
	if ctx != nil {
		if rawURL, ok := ctx.Value(runtimeWebRequestURLContextKey{}).(string); ok && strings.TrimSpace(rawURL) != "" {
			return rawURL
		}
	}
	return "tcp://" + address
}

func (g *runtimeWebRequestGuard) ensureDecisionAllowed(ctx context.Context, decision runtimeWebEndpointDecision) error {
	if decision.Reason == "" {
		return nil
	}
	if g.isDecisionApproved(decision) {
		return nil
	}
	if err := g.requestNonPublicPermission(ctx, decision); err != nil {
		return err
	}
	g.markDecisionApproved(decision)
	return nil
}

func (g *runtimeWebRequestGuard) isDecisionApproved(decision runtimeWebEndpointDecision) bool {
	if g == nil || g.permissionMu == nil || g.approved == nil {
		return false
	}
	key := runtimeWebDecisionApprovalKey(decision.Host, decision.Port, decision.Address)
	wildcard := runtimeWebDecisionApprovalKey(decision.Host, decision.Port, "*")
	g.permissionMu.Lock()
	defer g.permissionMu.Unlock()
	if _, ok := g.approved[key]; ok {
		return true
	}
	_, ok := g.approved[wildcard]
	return ok
}

func (g *runtimeWebRequestGuard) markDecisionApproved(decision runtimeWebEndpointDecision) {
	if g == nil || g.permissionMu == nil || g.approved == nil {
		return
	}
	address := strings.TrimSpace(decision.Address)
	if address == "" {
		address = "*"
	}
	key := runtimeWebDecisionApprovalKey(decision.Host, decision.Port, address)
	g.permissionMu.Lock()
	defer g.permissionMu.Unlock()
	g.approved[key] = struct{}{}
}

func runtimeWebDecisionApprovalKey(host string, port string, address string) string {
	host = strings.TrimRight(strings.ToLower(strings.TrimSpace(host)), ".")
	port = strings.TrimSpace(port)
	address = strings.ToLower(strings.TrimSpace(address))
	return host + "|" + port + "|" + address
}

func (g *runtimeWebRequestGuard) requestNonPublicPermission(ctx context.Context, decision runtimeWebEndpointDecision) error {
	provider := g.base.permissionProvider
	if provider == nil {
		return fmt.Errorf("web request to non-public host %q blocked: %s; permission provider is unavailable", decision.Host, decision.Reason)
	}
	payload, err := json.Marshal(runtimeWebEndpointPermissionInput{
		URL:    sanitizeRuntimeWebPermissionURL(decision.URL),
		Host:   decision.Host,
		Reason: decision.Reason,
	})
	if err != nil {
		return err
	}
	resolution, err := provider.RequestToolPermission(ctx, runtimeWebEndpointPermissionEntry(g.toolName), string(payload))
	if err != nil {
		return fmt.Errorf("web request to non-public host %q blocked: permission request failed: %w", decision.Host, err)
	}
	switch strings.TrimSpace(resolution.Decision) {
	case tools.ToolPermissionDecisionAllowOnce, tools.ToolPermissionDecisionAllowSession:
		return nil
	case tools.ToolPermissionDecisionDeny:
		return fmt.Errorf("web request to non-public host %q was denied by the user", decision.Host)
	case "":
		return fmt.Errorf("web request to non-public host %q blocked: permission request returned an empty decision", decision.Host)
	default:
		return fmt.Errorf("web request to non-public host %q blocked: unsupported permission decision %q", decision.Host, resolution.Decision)
	}
}

func (g *runtimeWebEndpointGuard) inspectURL(ctx context.Context, rawURL string) (runtimeWebEndpointDecision, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return runtimeWebEndpointDecision{}, fmt.Errorf("url is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return runtimeWebEndpointDecision{}, err
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	if scheme != "http" && scheme != "https" {
		return runtimeWebEndpointDecision{}, fmt.Errorf("web URL scheme %q is not allowed; only http and https are allowed", u.Scheme)
	}
	if u.User != nil {
		return runtimeWebEndpointDecision{}, fmt.Errorf("web URL userinfo is not allowed")
	}
	host := strings.TrimSpace(u.Hostname())
	if strings.TrimSpace(u.Host) == "" || host == "" {
		return runtimeWebEndpointDecision{}, fmt.Errorf("web URL host is required")
	}
	if strings.ContainsAny(host, "\x00\r\n\t ") {
		return runtimeWebEndpointDecision{}, fmt.Errorf("web URL host %q is invalid", host)
	}

	decision := runtimeWebEndpointDecision{URL: rawURL, Host: host, Port: runtimeWebURLPort(u)}
	if isRuntimeWebLocalhostName(host) {
		decision.Reason = "host is localhost"
		return decision, nil
	}
	if addr, ok := parseRuntimeWebHostAddr(host); ok {
		if reason := runtimeWebNonPublicAddrReason(addr); reason != "" {
			decision.Reason = "host is " + reason
			decision.Address = addr.String()
		}
		return decision, nil
	}

	addrs, err := g.resolveHostAddrs(ctx, host)
	if err != nil {
		return runtimeWebEndpointDecision{}, err
	}
	for _, resolved := range addrs {
		if reason := runtimeWebNonPublicAddrReason(resolved.addr); reason != "" {
			decision.Reason = "host resolves to " + reason
			decision.Address = resolved.addr.String()
			return decision, nil
		}
	}
	return decision, nil
}

type runtimeWebResolvedAddr struct {
	addr netip.Addr
	zone string
}

func (a runtimeWebResolvedAddr) dialHost() string {
	host := a.addr.String()
	if strings.TrimSpace(a.zone) != "" {
		host += "%" + strings.TrimSpace(a.zone)
	}
	return host
}

func (g *runtimeWebEndpointGuard) resolveHostAddrs(ctx context.Context, host string) ([]runtimeWebResolvedAddr, error) {
	if addr, ok := parseRuntimeWebHostAddr(host); ok {
		return []runtimeWebResolvedAddr{{addr: addr, zone: runtimeWebHostZone(host)}}, nil
	}
	resolver := g.resolveIPAddrs
	if resolver == nil {
		resolver = net.DefaultResolver.LookupIPAddr
	}
	addrs, err := resolver(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve web URL host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("resolve web URL host %q: no addresses found", host)
	}
	resolved := make([]runtimeWebResolvedAddr, 0, len(addrs))
	for _, ipAddr := range addrs {
		addr, ok := netip.AddrFromSlice(ipAddr.IP)
		if !ok {
			continue
		}
		resolved = append(resolved, runtimeWebResolvedAddr{addr: addr.Unmap(), zone: ipAddr.Zone})
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("resolve web URL host %q: no usable addresses found", host)
	}
	return resolved, nil
}

func runtimeWebURLPort(u *url.URL) string {
	if u == nil {
		return ""
	}
	if port := strings.TrimSpace(u.Port()); port != "" {
		return port
	}
	switch strings.ToLower(strings.TrimSpace(u.Scheme)) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func runtimeWebEndpointPermissionEntry(toolName string) tools.CatalogEntry {
	title := "Web Fetch"
	description := "Allow WebFetch to contact a non-public web endpoint."
	if toolName == tools.RuntimeToolWebSearch {
		title = "Web Search"
		description = "Allow WebSearch to contact a non-public provider endpoint."
	}
	return tools.CatalogEntry{
		CanonicalName: toolName,
		Title:         title,
		Description:   description,
		Source:        tools.ToolSourceRuntime,
		ToolClass:     tools.ToolClassRuntimeExec,
		PermissionSpec: tools.PermissionSpec{
			AllowExecute: true,
		},
	}
}

func parseRuntimeWebHostAddr(host string) (netip.Addr, bool) {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return netip.Addr{}, false
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		if zoneIndex := strings.LastIndex(host, "%"); zoneIndex > 0 {
			addr, err = netip.ParseAddr(host[:zoneIndex])
		}
	}
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func runtimeWebHostZone(host string) string {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if zoneIndex := strings.LastIndex(host, "%"); zoneIndex > 0 {
		return host[zoneIndex+1:]
	}
	return ""
}

func isRuntimeWebLocalhostName(host string) bool {
	normalized := strings.TrimRight(strings.ToLower(strings.TrimSpace(host)), ".")
	return normalized == "localhost" || strings.HasSuffix(normalized, ".localhost")
}

func runtimeWebNonPublicAddrReason(addr netip.Addr) string {
	addr = addr.Unmap()
	switch {
	case addr == runtimeWebMetadataAddr:
		return "cloud metadata address 169.254.169.254"
	case addr.IsUnspecified():
		return "unspecified address " + addr.String()
	case addr.IsLoopback():
		return "loopback address " + addr.String()
	case addr.IsPrivate():
		return "private address " + addr.String()
	case addr.IsLinkLocalUnicast():
		return "link-local address " + addr.String()
	case addr.IsLinkLocalMulticast():
		return "link-local multicast address " + addr.String()
	case addr.IsMulticast():
		return "multicast address " + addr.String()
	default:
		return ""
	}
}

func sanitizeRuntimeWebPermissionURL(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return strings.TrimSpace(rawURL)
	}
	u.User = nil
	u.Fragment = ""
	if u.RawQuery != "" {
		values := u.Query()
		for key, value := range values {
			if !isSensitiveRuntimeWebQueryKey(key) {
				continue
			}
			for i := range value {
				value[i] = "[redacted]"
			}
			values[key] = value
		}
		u.RawQuery = values.Encode()
	}
	return u.String()
}

func isSensitiveRuntimeWebQueryKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.NewReplacer("_", "", "-", "", ".", "").Replace(normalized)
	for _, marker := range []string{"apikey", "token", "secret", "password", "passwd", "credential", "authorization"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return normalized == "auth" || strings.HasSuffix(normalized, "auth") || normalized == "key" || strings.HasSuffix(normalized, "key")
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
