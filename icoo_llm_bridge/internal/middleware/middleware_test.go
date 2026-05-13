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
