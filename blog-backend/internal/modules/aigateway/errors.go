package aigateway

import (
	"errors"
	"net/http"

	"dh-blog/internal/platform/search"
)

// GatewayError is the error contract the gateway exposes to agents: a stable
// machine-readable type plus the HTTP status that carries the same meaning.
type GatewayError struct {
	Status    int
	Type      string
	Message   string
	Provider  string
	logStatus string
}

func (e *GatewayError) Error() string { return e.Message }

// LogStatus is the value written to the request log's status column.
func (e *GatewayError) LogStatus() string {
	if e.logStatus != "" {
		return e.logStatus
	}
	switch e.Type {
	case "invalid_request":
		return StatusInvalidRequest
	case "provider_not_allowed":
		return StatusProviderNotAllowed
	case "provider_not_found":
		// Kept distinct from no_provider_available: naming a disabled provider
		// is a caller mistake, and folding it in would make the dashboard read
		// as if the gateway itself had nothing available.
		return StatusProviderNotFound
	case "rate_limit_exceeded":
		return StatusRateLimited
	case "no_provider_available":
		return StatusNoProvider
	default:
		return StatusProviderError
	}
}

func newGatewayError(status int, errorType, message, provider string) *GatewayError {
	return &GatewayError{Status: status, Type: errorType, Message: message, Provider: provider}
}

// asGatewayError normalizes any error raised on the request path.
func asGatewayError(err error) *GatewayError {
	var gatewayErr *GatewayError
	if errors.As(err, &gatewayErr) {
		return gatewayErr
	}
	var providerErr *search.Error
	if errors.As(err, &providerErr) {
		return gatewayErrorFromProvider(providerErr)
	}
	return newGatewayError(http.StatusInternalServerError, "internal_error", err.Error(), "")
}

// gatewayErrorFromProvider maps a classified upstream failure onto the
// gateway's own contract.
func gatewayErrorFromProvider(err *search.Error) *GatewayError {
	switch err.Kind {
	case search.KindBadRequest:
		// The upstream rejected the parameters, which came from the caller.
		return newGatewayError(http.StatusBadRequest, "invalid_request", err.Message, err.Provider)
	case search.KindRateLimited:
		return newGatewayError(http.StatusTooManyRequests, "rate_limit_exceeded", err.Message, err.Provider)
	case search.KindQuotaExceeded:
		gatewayErr := newGatewayError(http.StatusServiceUnavailable, "no_provider_available", err.Message, err.Provider)
		gatewayErr.logStatus = StatusQuotaExceeded
		return gatewayErr
	case search.KindTimeout:
		return newGatewayError(http.StatusGatewayTimeout, "provider_timeout", err.Message, err.Provider)
	default:
		// Auth failures land here too: a broken credential is the operator's
		// problem, not something the caller can fix by changing the request.
		return newGatewayError(http.StatusBadGateway, "provider_error", err.Message, err.Provider)
	}
}

// routeGatewayError maps a routing policy rejection.
func routeGatewayError(err error) *GatewayError {
	switch {
	case errors.Is(err, ErrProviderNotFound):
		return newGatewayError(http.StatusNotFound, "provider_not_found", err.Error(), "")
	case errors.Is(err, ErrProviderNotAllowed):
		return newGatewayError(http.StatusForbidden, "provider_not_allowed", err.Error(), "")
	default:
		return newGatewayError(http.StatusServiceUnavailable, "no_provider_available", err.Error(), "")
	}
}

// authGatewayError maps a credential rejection. Every variant is a 401 so a
// caller cannot probe which keys exist.
func authGatewayError(err error) *GatewayError {
	switch {
	case errors.Is(err, ErrMissingAPIKey):
		return newGatewayError(http.StatusUnauthorized, "invalid_api_key", ErrMissingAPIKey.Error(), "")
	case errors.Is(err, ErrAPIKeyRevoked):
		return newGatewayError(http.StatusUnauthorized, "invalid_api_key", ErrAPIKeyRevoked.Error(), "")
	case errors.Is(err, ErrAPIKeyExpired):
		return newGatewayError(http.StatusUnauthorized, "invalid_api_key", ErrAPIKeyExpired.Error(), "")
	case errors.Is(err, ErrInvalidAPIKey):
		return newGatewayError(http.StatusUnauthorized, "invalid_api_key", ErrInvalidAPIKey.Error(), "")
	default:
		return newGatewayError(http.StatusInternalServerError, "internal_error", err.Error(), "")
	}
}
