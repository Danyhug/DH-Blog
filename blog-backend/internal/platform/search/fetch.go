package search

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// FetchRequest asks for the main content of one page.
type FetchRequest struct {
	URL string
	// Timeout is how long the upstream may spend on the page. It is handed to
	// the provider so the provider gives up before the gateway's own HTTP
	// client does: a page the provider finishes after the caller has hung up
	// is still billed, and nobody receives it.
	Timeout time.Duration
}

// FetchResponse is one page, reduced to text an agent can read.
type FetchResponse struct {
	// URL is where the page ended up after redirects.
	URL         string
	Title       string
	Description string
	// Content is the page's main content as Markdown.
	Content     string
	ContentType string
	// StatusCode is the target site's own answer. A 404 page is still a page
	// the provider fetched — and billed — so it comes back as a response, not
	// an error, and the caller decides what a 404 means.
	StatusCode int
	Credits    int
}

// Fetcher is a provider that can read a single URL, as opposed to searching.
type Fetcher interface {
	Fetch(ctx context.Context, req FetchRequest) (FetchResponse, error)
}

// flexString reads a metadata field that some pages give as a string and
// others as a list of strings (a page with two <title> tags, say).
type flexString string

func (f *flexString) UnmarshalJSON(raw []byte) error {
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		*f = flexString(strings.TrimSpace(single))
		return nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		for _, item := range list {
			if item = strings.TrimSpace(item); item != "" {
				*f = flexString(item)
				return nil
			}
		}
	}
	*f = ""
	return nil
}
