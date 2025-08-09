package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// Authenticator interface defines authentication operations
type Authenticator interface {
	Authenticate(username, password string) bool
	Middleware(next http.Handler) http.Handler
	IsEnabled() bool
}

// BasicAuthenticator implements basic authentication
type BasicAuthenticator struct {
	enabled  bool
	username string
	password string
}

// NewBasicAuthenticator creates a new basic authenticator
func NewBasicAuthenticator(enabled bool, username, password string) Authenticator {
	return &BasicAuthenticator{
		enabled:  enabled,
		username: username,
		password: password,
	}
}

// IsEnabled returns whether authentication is enabled
func (ba *BasicAuthenticator) IsEnabled() bool {
	return ba.enabled
}

// Authenticate validates username and password credentials
func (ba *BasicAuthenticator) Authenticate(username, password string) bool {
	if !ba.enabled {
		return true // Authentication disabled, allow all
	}

	// Use constant-time comparison to prevent timing attacks
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(ba.username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(ba.password)) == 1

	return usernameMatch && passwordMatch
}

// Middleware returns an HTTP middleware that enforces basic authentication
func (ba *BasicAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If authentication is disabled, proceed without checking
		if !ba.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Extract credentials from Authorization header
		username, password, ok := ba.extractCredentials(r)
		if !ok {
			ba.sendUnauthorized(w)
			return
		}

		// Validate credentials
		if !ba.Authenticate(username, password) {
			ba.sendUnauthorized(w)
			return
		}

		// Authentication successful, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

// extractCredentials extracts username and password from Basic Auth header
func (ba *BasicAuthenticator) extractCredentials(r *http.Request) (username, password string, ok bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "", false
	}

	// Check if it's Basic authentication
	const basicPrefix = "Basic "
	if !strings.HasPrefix(authHeader, basicPrefix) {
		return "", "", false
	}

	// Decode base64 credentials
	encoded := authHeader[len(basicPrefix):]
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", false
	}

	// Split username:password
	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// sendUnauthorized sends a 401 Unauthorized response with WWW-Authenticate header
func (ba *BasicAuthenticator) sendUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Mini HTTP Service"`)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, "401 Unauthorized\n")
}

// NoOpAuthenticator is an authenticator that always allows access
type NoOpAuthenticator struct{}

// NewNoOpAuthenticator creates a new no-op authenticator
func NewNoOpAuthenticator() Authenticator {
	return &NoOpAuthenticator{}
}

// IsEnabled always returns false for no-op authenticator
func (noa *NoOpAuthenticator) IsEnabled() bool {
	return false
}

// Authenticate always returns true for no-op authenticator
func (noa *NoOpAuthenticator) Authenticate(username, password string) bool {
	return true
}

// Middleware returns a pass-through middleware for no-op authenticator
func (noa *NoOpAuthenticator) Middleware(next http.Handler) http.Handler {
	return next
}