package search

// Metadata is the static, human-facing description of a provider: where it
// lives, where its documentation is, and where an operator goes to obtain a
// credential. It is fixed knowledge about the upstream rather than user
// configuration, so it lives in code next to the adapters instead of in the
// database.
type Metadata struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	HomeURL     string `json:"homeUrl"`
	DocsURL     string `json:"docsUrl"`
	ConsoleURL  string `json:"consoleUrl"`
	LogoURL     string `json:"logoUrl"`
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
		LogoURL:     "https://cdn.search.brave.com/search-api/web/v1/client/favicon.png",
		Billing:     "按请求计费，每次调用 1 次额度；免费版 1 次/秒、2000 次/月",
	},
	ProviderTavily: {
		Name:        ProviderTavily,
		DisplayName: "Tavily",
		HomeURL:     "https://tavily.com",
		DocsURL:     "https://docs.tavily.com/documentation/api-reference/endpoint/search",
		ConsoleURL:  "https://app.tavily.com/home",
		LogoURL:     "https://tavily.com/favicon.ico",
		Billing:     "按 credit 计费，basic/fast 1 credit、advanced 2 credit",
	},
	ProviderExa: {
		Name:        ProviderExa,
		DisplayName: "Exa",
		HomeURL:     "https://exa.ai",
		DocsURL:     "https://docs.exa.ai/reference/search",
		ConsoleURL:  "https://dashboard.exa.ai/api-keys",
		LogoURL:     "https://exa.ai/images/favicon-32x32.png",
		Billing:     "按美元计费，单次费用随搜索类型浮动，网关按微美元记录实际花费",
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
