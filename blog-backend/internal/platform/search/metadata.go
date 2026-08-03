package search

// Metadata is the static, human-facing description of a provider: where it
// lives, where its documentation is, and where an operator goes to obtain a
// credential. It is fixed knowledge about the upstream rather than user
// configuration, so it lives in code next to the adapters instead of in the
// database.
//
// There is deliberately no logo here. A logo is not knowledge the backend has
// to hold: shipping a URL meant every admin page load fetched four images from
// four third-party hosts, and a host going away left the page full of broken
// images. The frontend bundles the marks as static assets instead.
type Metadata struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	HomeURL     string `json:"homeUrl"`
	DocsURL     string `json:"docsUrl"`
	ConsoleURL  string `json:"consoleUrl"`
	// Billing explains what one unit of Credits means for this provider, so
	// the admin dashboard does not present incomparable numbers as if they
	// were the same currency.
	Billing string `json:"billing"`
}

var providerMetadata = map[string]Metadata{
	ProviderBrave: {
		Name:        ProviderBrave,
		DisplayName: "Brave Search",
		HomeURL:     "https://brave.com/search/api/",
		DocsURL:     "https://api-dashboard.search.brave.com/app/documentation",
		ConsoleURL:  "https://api-dashboard.search.brave.com/app/keys",
		Billing:     "按请求计费，每次调用 1 次额度；免费版 1 次/秒、2000 次/月",
	},
	ProviderTavily: {
		Name:        ProviderTavily,
		DisplayName: "Tavily",
		HomeURL:     "https://tavily.com",
		DocsURL:     "https://docs.tavily.com/documentation/api-reference/endpoint/search",
		ConsoleURL:  "https://app.tavily.com/home",
		Billing:     "按 credit 计费，basic/fast 1 credit、advanced 2 credit",
	},
	ProviderExa: {
		Name:        ProviderExa,
		DisplayName: "Exa",
		HomeURL:     "https://exa.ai",
		DocsURL:     "https://docs.exa.ai/reference/search",
		ConsoleURL:  "https://dashboard.exa.ai/api-keys",
		Billing:     "按美元计费，单次费用随搜索类型浮动，网关按微美元记录实际花费",
	},
	ProviderFirecrawl: {
		Name:        ProviderFirecrawl,
		DisplayName: "Firecrawl",
		HomeURL:     "https://firecrawl.dev",
		DocsURL:     "https://docs.firecrawl.dev/api-reference/endpoint/search",
		ConsoleURL:  "https://www.firecrawl.dev/app/api-keys",
		Billing:     "按 credit 计费，搜索每 10 条结果 2 credit（不足 10 条按 10 条算）；要正文时每抓一页再加 1 credit",
	},
}

// MetaFor returns the static description of a provider. An unknown name yields
// a zero-value Metadata carrying just the name.
func MetaFor(name string) Metadata {
	if meta, ok := providerMetadata[name]; ok {
		return meta
	}
	return Metadata{Name: name, DisplayName: name}
}
