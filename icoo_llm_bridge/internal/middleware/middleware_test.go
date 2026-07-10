package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type testAuthVerifier struct {
	scopesBySecret map[string]string
}

func (v testAuthVerifier) Verify(_ context.Context, secret string, scope string) bool {
	return v.scopesBySecret[secret] == scope || v.scopesBySecret[secret] == "*"
}

func TestAdminAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                  string
		allowLocalWithoutAuth bool
		remoteAddr            string
		apiKey                string
		forwardedFor          string
		wantStatus            int
		wantCalled            bool
	}{
		{
			name:                  "local without auth is allowed",
			allowLocalWithoutAuth: true,
			remoteAddr:            "127.0.0.1:12345",
			wantStatus:            http.StatusOK,
			wantCalled:            true,
		},
		{
			name:       "missing key is unauthorized",
			remoteAddr: "203.0.113.10:12345",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:                  "forwarded loopback does not bypass auth",
			allowLocalWithoutAuth: true,
			remoteAddr:            "203.0.113.10:12345",
			forwardedFor:          "127.0.0.1",
			wantStatus:            http.StatusUnauthorized,
		},
		{
			name:       "valid admin scoped key is allowed",
			remoteAddr: "203.0.113.10:12345",
			apiKey:     "admin-secret",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "non admin scoped key is rejected",
			remoteAddr: "203.0.113.10:12345",
			apiKey:     "proxy-secret",
			wantStatus: http.StatusUnauthorized,
		},
	}

	auth := testAuthVerifier{scopesBySecret: map[string]string{
		"admin-secret": "admin",
		"proxy-secret": "proxy",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			engine := gin.New()
			engine.Use(AdminAuth(auth, tt.allowLocalWithoutAuth))
			engine.GET("/admin", func(ctx *gin.Context) {
				called = true
				ctx.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("x-api-key", tt.apiKey)
			}
			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}
			recorder := httptest.NewRecorder()

			engine.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Fatalf("handler called = %v, want %v", called, tt.wantCalled)
			}
		})
	}
}

func TestCORSRejectsUntrustedBrowserOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		origin      string
		method      string
		wantStatus  int
		wantOrigin  string
		wantHandled bool
	}{
		{
			name:        "request without origin is allowed",
			method:      http.MethodGet,
			wantStatus:  http.StatusOK,
			wantHandled: true,
		},
		{
			name:        "wails origin is allowed",
			origin:      "wails://wails",
			method:      http.MethodGet,
			wantStatus:  http.StatusOK,
			wantOrigin:  "wails://wails",
			wantHandled: true,
		},
		{
			name:        "wails localhost origin is allowed",
			origin:      "http://wails.localhost",
			method:      http.MethodGet,
			wantStatus:  http.StatusOK,
			wantOrigin:  "http://wails.localhost",
			wantHandled: true,
		},
		{
			name:        "loopback development origin is allowed",
			origin:      "http://127.0.0.1:5173",
			method:      http.MethodOptions,
			wantStatus:  http.StatusNoContent,
			wantOrigin:  "http://127.0.0.1:5173",
			wantHandled: false,
		},
		{
			name:        "hostile origin is rejected",
			origin:      "https://attacker.example",
			method:      http.MethodPost,
			wantStatus:  http.StatusForbidden,
			wantHandled: false,
		},
		{
			name:        "null origin is rejected",
			origin:      "null",
			method:      http.MethodPost,
			wantStatus:  http.StatusForbidden,
			wantHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handled := false
			engine := gin.New()
			engine.Use(CORS())
			engine.Any("/resource", func(ctx *gin.Context) {
				handled = true
				ctx.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, "/resource", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if handled != tt.wantHandled {
				t.Fatalf("handler called = %v, want %v", handled, tt.wantHandled)
			}
			if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Fatalf("allow origin = %q, want %q", got, tt.wantOrigin)
			}
		})
	}
}
