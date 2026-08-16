package agentapi

import "context"

// Scope values for the Identity contract. The strings must match aigateway's
// ScopeContentRead / ScopeContentWrite constants exactly: they are a
// cross-module wire contract, and agentapi deliberately does not import
// aigateway (the dependency direction is the other way).
const (
	// scopeContentRead gates the two read-only tools.
	scopeContentRead = "content:read"
	// scopeContentWrite gates creating, updating and image upload.
	scopeContentWrite = "content:write"
)

// Identity is the gateway credential on whose behalf a tool runs. The
// aigateway handler puts it into the request context before dispatching, so
// the tools never see the concrete key type — only this narrow view.
type Identity interface {
	KeyID() int
	// AuthorName is the byline articles get signed with. The empty string
	// means "fall back to the key name", and that fallback lives in the
	// implementing side (aigateway), not here.
	AuthorName() string
	HasScope(scope string) bool
}

type identityCtxKey struct{}

// IdentityContext stashes the caller's identity for the duration of one tool
// call. mcp.Tool only receives a context, so the transport layer hands the
// credential through here.
func IdentityContext(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{}, identity)
}

// IdentityFrom reads the caller back out of the context.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityCtxKey{}).(Identity)
	return identity, ok
}
