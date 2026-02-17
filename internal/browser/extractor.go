package browser

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-rod/rod"
)

// SearchResult represents a search engine result
type SearchResult struct {
	Title       string
	URL         string
	Description string
}

// PageContent represents extracted page content
type PageContent struct {
	Title   string
	URL     string
	Content string
}

// ExtractDuckDuckGoResults extracts search results from DuckDuckGo
func ExtractDuckDuckGoResults(page *rod.Page, count int) ([]SearchResult, error) {
	// Wait for results to load
	if err := page.WaitStable(500); err != nil {
		return nil, fmt.Errorf("page not stable: %w", err)
	}

	var results []SearchResult

	// DuckDuckGo organic results selector
	// Try multiple selectors as DDG structure may vary
	selectors := []string{
		"article[data-testid='result']",
		"div.result",
		"div[data-result]",
	}

	var elements rod.Elements
	for _, sel := range selectors {
		els, err := page.Elements(sel)
		if err == nil && len(els) > 0 {
			elements = els
			break
		}
	}

	if len(elements) == 0 {
		// Fallback: try to find any links with result-like structure
		return extractFallbackResults(page, count)
	}

	for i, el := range elements {
		if i >= count {
			break
		}

		result := SearchResult{}

		// Extract title and URL from anchor
		if link, err := el.Element("a"); err == nil && link != nil {
			if title, err := link.Text(); err == nil {
				result.Title = strings.TrimSpace(title)
			}
			if href, err := link.Property("href"); err == nil {
				result.URL = href.Str()
			}
		}

		// Extract description
		descSelectors := []string{
			"span[data-result]",
			".result__snippet",
			"p",
		}
		for _, descSel := range descSelectors {
			if descEl, err := el.Element(descSel); err == nil && descEl != nil {
				if desc, err := descEl.Text(); err == nil && desc != "" {
					result.Description = strings.TrimSpace(desc)
					break
				}
			}
		}

		// Only add if we have at least a title or URL
		if result.Title != "" || result.URL != "" {
			results = append(results, result)
		}
	}

	return results, nil
}

// extractFallbackResults tries to extract results when standard selectors fail
func extractFallbackResults(page *rod.Page, count int) ([]SearchResult, error) {
	var results []SearchResult

	// Get all links from the page that look like search results
	links, err := page.Elements("a[href^='http']")
	if err != nil {
		return results, nil
	}

	seen := make(map[string]bool)
	for _, link := range links {
		if len(results) >= count {
			break
		}

		href, err := link.Property("href")
		if err != nil {
			continue
		}
		url := href.Str()

		// Skip DDG internal links and duplicates
		if strings.Contains(url, "duckduckgo.com") || seen[url] {
			continue
		}
		seen[url] = true

		title, _ := link.Text()
		title = strings.TrimSpace(title)

		if title != "" && len(title) > 5 {
			results = append(results, SearchResult{
				Title: title,
				URL:   url,
			})
		}
	}

	return results, nil
}

// ExtractGoogleResults extracts search results from Google
func ExtractGoogleResults(page *rod.Page, count int) ([]SearchResult, error) {
	// Wait for page to load and stabilize
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("page load failed: %w", err)
	}
	page.WaitStable(1000)

	var results []SearchResult

	// Try multiple selector strategies for Google results
	selectors := []string{
		"div.g",                    // Standard result container
		"div[data-hveid] div.g",    // Nested in data-hveid
		"div.MjjYud",               // Alternative container
		"div[data-sokoban-container]", // Another variant
	}

	var elements rod.Elements
	for _, sel := range selectors {
		els, err := page.Elements(sel)
		if err == nil && len(els) > 0 {
			elements = els
			break
		}
	}

	// If standard selectors fail, try extracting from any h3 with links
	if len(elements) == 0 {
		return extractGoogleFallback(page, count)
	}

	for i, el := range elements {
		if i >= count {
			break
		}

		result := SearchResult{}

		// Extract title from h3
		if h3, err := el.Element("h3"); err == nil && h3 != nil {
			if title, err := h3.Text(); err == nil {
				result.Title = strings.TrimSpace(title)
			}
		}

		// Extract URL from anchor
		if link, err := el.Element("a"); err == nil && link != nil {
			if href, err := link.Property("href"); err == nil {
				url := href.Str()
				// Skip Google internal links
				if !strings.Contains(url, "google.com/search") {
					result.URL = url
				}
			}
		}

		// Extract description from various possible containers
		descSelectors := []string{
			"div[data-sncf]",
			"div.VwiC3b",
			"span.st",
			"div[style*='line-clamp']",
			"div.IsZvec",
		}
		for _, descSel := range descSelectors {
			if descEl, err := el.Element(descSel); err == nil && descEl != nil {
				if desc, err := descEl.Text(); err == nil && desc != "" {
					result.Description = strings.TrimSpace(desc)
					break
				}
			}
		}

		if result.Title != "" && result.URL != "" {
			results = append(results, result)
		}
	}

	return results, nil
}

// extractGoogleFallback extracts results when standard selectors fail
func extractGoogleFallback(page *rod.Page, count int) ([]SearchResult, error) {
	var results []SearchResult

	// Find all h3 elements that are likely search result titles
	h3s, err := page.Elements("h3")
	if err != nil {
		return results, nil
	}

	seen := make(map[string]bool)
	for _, h3 := range h3s {
		if len(results) >= count {
			break
		}

		title, err := h3.Text()
		if err != nil || title == "" {
			continue
		}
		title = strings.TrimSpace(title)

		// Find parent link
		parent, err := h3.Parent()
		if err != nil {
			continue
		}

		// Look for anchor in parent or grandparent
		var link *rod.Element
		if a, err := parent.Element("a"); err == nil {
			link = a
		} else if grandparent, err := parent.Parent(); err == nil {
			if a, err := grandparent.Element("a"); err == nil {
				link = a
			}
		}

		if link == nil {
			continue
		}

		href, err := link.Property("href")
		if err != nil {
			continue
		}
		url := href.Str()

		// Skip Google internal URLs and duplicates
		if strings.Contains(url, "google.com") || seen[url] || url == "" {
			continue
		}
		seen[url] = true

		results = append(results, SearchResult{
			Title: title,
			URL:   url,
		})
	}

	return results, nil
}

// ExtractBingResults extracts search results from Bing
func ExtractBingResults(page *rod.Page, count int) ([]SearchResult, error) {
	// Wait for page to load
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("page load failed: %w", err)
	}
	page.WaitStable(1000)

	var results []SearchResult

	// Bing search result selectors
	selectors := []string{
		"li.b_algo",           // Main result container
		"div.b_algo",          // Alternative container
		"ol#b_results li.b_algo",
	}

	var elements rod.Elements
	for _, sel := range selectors {
		els, err := page.Elements(sel)
		if err == nil && len(els) > 0 {
			elements = els
			break
		}
	}

	if len(elements) == 0 {
		return extractBingFallback(page, count)
	}

	for i, el := range elements {
		if i >= count {
			break
		}

		result := SearchResult{}

		// Extract title from h2 > a using Elements (non-blocking)
		if h2Els, err := el.Elements("h2"); err == nil && len(h2Els) > 0 {
			if aEls, err := h2Els[0].Elements("a"); err == nil && len(aEls) > 0 {
				a := aEls[0]
				// Use textContent via Eval (rod's Text() doesn't work with Bing)
				if titleObj, err := a.Eval(`() => this.textContent`); err == nil && titleObj != nil {
					result.Title = strings.TrimSpace(titleObj.Value.Str())
				}
				if href, err := a.Property("href"); err == nil {
					url := href.Str()
					// Extract actual URL from Bing tracking URL
					result.URL = extractBingActualURL(url)
				}
			}
		}

		// Also try to get cite element for clean URL
		if result.URL == "" || strings.Contains(result.URL, "bing.com") {
			if citeEls, err := el.Elements("cite"); err == nil && len(citeEls) > 0 {
				if citeTextObj, err := citeEls[0].Eval(`() => this.textContent`); err == nil && citeTextObj != nil {
					citeText := strings.TrimSpace(citeTextObj.Value.Str())
					if strings.HasPrefix(citeText, "http") {
						result.URL = citeText
					} else if citeText != "" {
						result.URL = "https://" + strings.Split(citeText, " ")[0]
					}
				}
			}
		}

		// Extract description using Elements (non-blocking)
		descSelectors := []string{
			"div.b_caption p",
			"p",
			".b_algoSlug",
		}
		for _, descSel := range descSelectors {
			if descEls, err := el.Elements(descSel); err == nil && len(descEls) > 0 {
				if descObj, err := descEls[0].Eval(`() => this.textContent`); err == nil && descObj != nil {
					desc := strings.TrimSpace(descObj.Value.Str())
					if desc != "" {
						result.Description = desc
						break
					}
				}
			}
		}

		if result.Title != "" && result.URL != "" && !strings.Contains(result.URL, "bing.com") {
			results = append(results, result)
		}
	}

	return results, nil
}

// extractBingActualURL extracts the actual URL from Bing's tracking URL
func extractBingActualURL(bingURL string) string {
	// Bing URLs look like: https://www.bing.com/ck/a?...&u=a1aHR0cHM6Ly9...
	// The actual URL is base64 encoded in the 'u' parameter after 'a1'
	if !strings.Contains(bingURL, "bing.com/ck/a") {
		return bingURL
	}

	// Try to find the 'u=' parameter
	idx := strings.Index(bingURL, "&u=a1")
	if idx == -1 {
		idx = strings.Index(bingURL, "?u=a1")
	}
	if idx == -1 {
		return bingURL
	}

	// Extract the base64 part
	start := idx + 5 // Skip "&u=a1" or "?u=a1"
	end := strings.Index(bingURL[start:], "&")
	var encoded string
	if end == -1 {
		encoded = bingURL[start:]
	} else {
		encoded = bingURL[start : start+end]
	}

	// Decode base64 (URL-safe encoding without padding)
	decoded, err := decodeBase64URL(encoded)
	if err != nil {
		return bingURL
	}

	return decoded
}

// decodeBase64URL decodes a base64 URL-safe encoded string
func decodeBase64URL(encoded string) (string, error) {
	// Add padding if needed
	padding := 4 - len(encoded)%4
	if padding != 4 {
		encoded += strings.Repeat("=", padding)
	}

	// Replace URL-safe chars
	encoded = strings.ReplaceAll(encoded, "-", "+")
	encoded = strings.ReplaceAll(encoded, "_", "/")

	// Decode
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

// extractBingFallback extracts results when standard selectors fail
func extractBingFallback(page *rod.Page, count int) ([]SearchResult, error) {
	var results []SearchResult

	// Find all h2 elements with links
	h2s, err := page.Elements("h2 a")
	if err != nil {
		return results, nil
	}

	seen := make(map[string]bool)
	for _, a := range h2s {
		if len(results) >= count {
			break
		}

		// Use textContent via Eval (rod's Text() doesn't work with Bing)
		titleObj, err := a.Eval(`() => this.textContent`)
		if err != nil {
			continue
		}
		title := strings.TrimSpace(titleObj.Value.Str())
		if title == "" {
			continue
		}

		href, err := a.Property("href")
		if err != nil {
			continue
		}
		bingURL := href.Str()

		// Extract actual URL from Bing tracking URL
		url := extractBingActualURL(bingURL)

		// Skip Bing internal URLs and duplicates
		if strings.Contains(url, "bing.com") || seen[url] || url == "" {
			continue
		}
		seen[url] = true

		results = append(results, SearchResult{
			Title: title,
			URL:   url,
		})
	}

	return results, nil
}

// ExtractPageContent extracts main content from a web page
func ExtractPageContent(page *rod.Page, maxLength int) (*PageContent, error) {
	// Wait for page to load
	if err := page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("page load failed: %w", err)
	}

	// Wait for page to stabilize (important for JS-rendered content)
	page.WaitStable(500)

	// Get page title and URL from page info (more reliable than Eval)
	title := ""
	url := ""
	if info, err := page.Info(); err == nil && info != nil {
		title = info.Title
		url = info.URL
	}

	// Remove non-content elements (use arrow function for Eval)
	_, _ = page.Eval(`() => {
		['script', 'style', 'nav', 'header', 'footer', 'aside', 'iframe', 'noscript', 'svg'].forEach(tag => {
			document.querySelectorAll(tag).forEach(el => el.remove());
		});
		// Remove hidden elements
		document.querySelectorAll('[hidden], [aria-hidden="true"], .hidden, .hide').forEach(el => el.remove());
	}`)

	// Try to find main content areas using Elements (non-blocking)
	content := ""
	contentSelectors := []string{
		"article",
		"main",
		"[role='main']",
		".content",
		"#content",
		".post-content",
		".article-content",
		".entry-content",
		".markdown-body",
		".post-body",
		".Article",           // go.dev style
		".Blog-content",      // go.dev blog
		"#main-content",
		".main-content",
		".page-content",
		".doc-content",
		".documentation",
	}

	for _, sel := range contentSelectors {
		// Use Elements (non-blocking) instead of Element (blocking)
		if els, err := page.Elements(sel); err == nil && len(els) > 0 {
			if text, err := els[0].Text(); err == nil && text != "" {
				content = strings.TrimSpace(text)
				if content != "" {
					break
				}
			}
		}
	}

	// Fallback to body (use Elements to avoid blocking)
	if content == "" {
		if els, err := page.Elements("body"); err == nil && len(els) > 0 {
			content, _ = els[0].Text()
		}
	}

	// Clean and truncate content
	content = cleanText(content)
	if maxLength > 0 && len(content) > maxLength {
		content = content[:maxLength] + "\n\n... [content truncated]"
	}

	return &PageContent{
		Title:   title,
		URL:     url,
		Content: content,
	}, nil
}

// cleanText cleans extracted text by normalizing whitespace
func cleanText(text string) string {
	lines := strings.Split(text, "\n")
	var cleaned []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}
