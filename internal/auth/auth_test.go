package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewBasicAuthenticator(t *testing.T) {
	auth := NewBasicAuthenticator(true, "admin", "secret")
	
	if !auth.IsEnabled() {
		t.Error("Expected authenticator to be enabled")
	}
	
	basicAuth, ok := auth.(*BasicAuthenticator)
	if !ok {
		t.Fatal("Expected BasicAuthenticator type")
	}
	
	if basicAuth.username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", basicAuth.username)
	}
	if basicAuth.password != "secret" {
		t.Errorf("Expected password 'secret', got '%s'", basicAuth.password)
	}
}

func TestBasicAuthenticator_IsEnabled(t *testing.T) {
	// Test enabled authenticator
	auth := NewBasicAuthenticator(true, "user", "pass")
	if !auth.IsEnabled() {
		t.Error("Expected authenticator to be enabled")
	}
	
	// Test disabled authenticator
	auth = NewBasicAuthenticator(false, "user", "pass")
	if auth.IsEnabled() {
		t.Error("Expected authenticator to be disabled")
	}
}

func TestBasicAuthenticator_Authenticate(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		configUser     string
		configPass     string
		testUser       string
		testPass       string
		expectedResult bool
	}{
		{
			name:           "disabled auth allows all",
			enabled:        false,
			configUser:     "admin",
			configPass:     "secret",
			testUser:       "wrong",
			testPass:       "wrong",
			expectedResult: true,
		},
		{
			name:           "correct credentials",
			enabled:        true,
			configUser:     "admin",
			configPass:     "secret",
			testUser:       "admin",
			testPass:       "secret",
			expectedResult: true,
		},
		{
			name:           "wrong username",
			enabled:        true,
			configUser:     "admin",
			configPass:     "secret",
			testUser:       "user",
			testPass:       "secret",
			expectedResult: false,
		},
		{
			name:           "wrong password",
			enabled:        true,
			configUser:     "admin",
			configPass:     "secret",
			testUser:       "admin",
			testPass:       "wrong",
			expectedResult: false,
		},
		{
			name:           "empty credentials",
			enabled:        true,
			configUser:     "admin",
			configPass:     "secret",
			testUser:       "",
			testPass:       "",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewBasicAuthenticator(tt.enabled, tt.configUser, tt.configPass)
			result := auth.Authenticate(tt.testUser, tt.testPass)
			
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestBasicAuthenticator_ExtractCredentials(t *testing.T) {
	auth := &BasicAuthenticator{}

	tests := []struct {
		name           string
		authHeader     string
		expectedUser   string
		expectedPass   string
		expectedOK     bool
	}{
		{
			name:           "valid basic auth",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret")),
			expectedUser:   "admin",
			expectedPass:   "secret",
			expectedOK:     true,
		},
		{
			name:           "no auth header",
			authHeader:     "",
			expectedUser:   "",
			expectedPass:   "",
			expectedOK:     false,
		},
		{
			name:           "wrong auth type",
			authHeader:     "Bearer token123",
			expectedUser:   "",
			expectedPass:   "",
			expectedOK:     false,
		},
		{
			name:           "invalid base64",
			authHeader:     "Basic invalid-base64!",
			expectedUser:   "",
			expectedPass:   "",
			expectedOK:     false,
		},
		{
			name:           "no colon separator",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("adminpassword")),
			expectedUser:   "",
			expectedPass:   "",
			expectedOK:     false,
		},
		{
			name:           "empty password",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:")),
			expectedUser:   "admin",
			expectedPass:   "",
			expectedOK:     true,
		},
		{
			name:           "password with colon",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:pass:word")),
			expectedUser:   "admin",
			expectedPass:   "pass:word",
			expectedOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			user, pass, ok := auth.extractCredentials(req)

			if user != tt.expectedUser {
				t.Errorf("Expected user '%s', got '%s'", tt.expectedUser, user)
			}
			if pass != tt.expectedPass {
				t.Errorf("Expected pass '%s', got '%s'", tt.expectedPass, pass)
			}
			if ok != tt.expectedOK {
				t.Errorf("Expected ok %v, got %v", tt.expectedOK, ok)
			}
		})
	}
}

func TestBasicAuthenticator_Middleware(t *testing.T) {
	// Test handler that sets a header to indicate it was called
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Handler-Called", "true")
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name               string
		enabled            bool
		configUser         string
		configPass         string
		authHeader         string
		expectedStatus     int
		expectedHandlerCalled bool
	}{
		{
			name:               "disabled auth allows access",
			enabled:            false,
			configUser:         "admin",
			configPass:         "secret",
			authHeader:         "",
			expectedStatus:     http.StatusOK,
			expectedHandlerCalled: true,
		},
		{
			name:               "valid credentials allow access",
			enabled:            true,
			configUser:         "admin",
			configPass:         "secret",
			authHeader:         "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret")),
			expectedStatus:     http.StatusOK,
			expectedHandlerCalled: true,
		},
		{
			name:               "no auth header returns 401",
			enabled:            true,
			configUser:         "admin",
			configPass:         "secret",
			authHeader:         "",
			expectedStatus:     http.StatusUnauthorized,
			expectedHandlerCalled: false,
		},
		{
			name:               "invalid credentials return 401",
			enabled:            true,
			configUser:         "admin",
			configPass:         "secret",
			authHeader:         "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:wrong")),
			expectedStatus:     http.StatusUnauthorized,
			expectedHandlerCalled: false,
		},
		{
			name:               "malformed auth header returns 401",
			enabled:            true,
			configUser:         "admin",
			configPass:         "secret",
			authHeader:         "Basic invalid-base64!",
			expectedStatus:     http.StatusUnauthorized,
			expectedHandlerCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewBasicAuthenticator(tt.enabled, tt.configUser, tt.configPass)
			middleware := auth.Middleware(testHandler)

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			handlerCalled := rr.Header().Get("X-Handler-Called") == "true"
			if handlerCalled != tt.expectedHandlerCalled {
				t.Errorf("Expected handler called %v, got %v", tt.expectedHandlerCalled, handlerCalled)
			}

			// Check WWW-Authenticate header for 401 responses
			if rr.Code == http.StatusUnauthorized {
				wwwAuth := rr.Header().Get("WWW-Authenticate")
				if wwwAuth == "" {
					t.Error("Expected WWW-Authenticate header for 401 response")
				}
				if !strings.Contains(wwwAuth, "Basic") {
					t.Errorf("Expected Basic auth challenge, got: %s", wwwAuth)
				}
			}
		})
	}
}

func TestNoOpAuthenticator(t *testing.T) {
	auth := NewNoOpAuthenticator()

	// Test IsEnabled
	if auth.IsEnabled() {
		t.Error("Expected NoOpAuthenticator to be disabled")
	}

	// Test Authenticate (should always return true)
	if !auth.Authenticate("any", "credentials") {
		t.Error("Expected NoOpAuthenticator to always authenticate")
	}

	// Test Middleware (should pass through)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Handler-Called", "true")
		w.WriteHeader(http.StatusOK)
	})

	middleware := auth.Middleware(testHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if rr.Header().Get("X-Handler-Called") != "true" {
		t.Error("Expected handler to be called")
	}
}