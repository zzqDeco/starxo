package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	toolutils "github.com/cloudwego/eino/components/tool/utils"

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
	Query string `json:"query" jsonschema:"description=search query"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=max result lines"`
}

type webSearchOutput struct {
	Query   string   `json:"query"`
	URL     string   `json:"url"`
	Results []string `json:"results"`
}

func newRuntimeWebCatalogEntries() ([]tools.CatalogEntry, error) {
	fetch, err := toolutils.InferTool(tools.RuntimeToolWebFetch,
		"Fetch a URL with HTTP GET and return compact text.",
		func(ctx context.Context, input webFetchInput) (webFetchOutput, error) {
			return runWebFetch(ctx, input)
		})
	if err != nil {
		return nil, err
	}
	search, err := toolutils.InferTool(tools.RuntimeToolWebSearch,
		"Search the web using DuckDuckGo HTML results and return compact links.",
		func(ctx context.Context, input webSearchInput) (webSearchOutput, error) {
			return runWebSearch(ctx, input)
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

func runWebSearch(ctx context.Context, input webSearchInput) (webSearchOutput, error) {
	if strings.TrimSpace(input.Query) == "" {
		return webSearchOutput{}, fmt.Errorf("query is required")
	}
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
	return webSearchOutput{Query: input.Query, URL: searchURL, Results: results}, nil
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
