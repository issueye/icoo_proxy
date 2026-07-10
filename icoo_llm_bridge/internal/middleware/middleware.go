package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuthVerifier interface {
	Verify(ctx context.Context, secret string, scope string) bool
}

type Middlewares struct {
	RequestID gin.HandlerFunc
	CORS      gin.HandlerFunc
	Recovery  gin.HandlerFunc
	AdminAuth gin.HandlerFunc
}

func NewMiddlewares(auth AuthVerifier, allowLocalWithoutAuth bool) Middlewares {
	return Middlewares{
		RequestID: RequestID(),
		CORS:      CORS(),
		Recovery:  gin.Recovery(),
		AdminAuth: AdminAuth(auth, allowLocalWithoutAuth),
	}
}

func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		requestID := strings.TrimSpace(ctx.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		ctx.Set("request_id", requestID)
		ctx.Header("X-ICOO-Request-ID", requestID)
		ctx.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := strings.TrimSpace(ctx.GetHeader("Origin"))
		if origin != "" {
			if !isAllowedBrowserOrigin(origin) {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": gin.H{"code": "ORIGIN_FORBIDDEN", "message": "browser origin is not allowed"},
				})
				return
			}
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Vary", "Origin")
		}
		ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,x-api-key,x-request-id")
		ctx.Header("Access-Control-Max-Age", "600")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

func AdminAuth(auth AuthVerifier, allowLocalWithoutAuth bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if allowLocalWithoutAuth && isLocalClient(ctx.Request.RemoteAddr) {
			ctx.Next()
			return
		}
		key := extractAPIKey(ctx)
		if key == "" || auth == nil || !auth.Verify(ctx.Request.Context(), key, "admin") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid admin api key"},
			})
			return
		}
		ctx.Next()
	}
}

func isAllowedBrowserOrigin(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	switch strings.ToLower(parsed.Scheme) {
	case "wails":
		return host == "wails"
	case "http", "https":
		if host == "localhost" || host == "wails.localhost" {
			return true
		}
		ip := net.ParseIP(host)
		return ip != nil && ip.IsLoopback()
	default:
		return false
	}
}

func extractAPIKey(ctx *gin.Context) string {
	if key := strings.TrimSpace(ctx.GetHeader("x-api-key")); key != "" {
		return key
	}
	auth := strings.TrimSpace(ctx.GetHeader("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func isLocalClient(raw string) bool {
	host, _, err := net.SplitHostPort(raw)
	if err == nil {
		raw = host
	}
	ip := net.ParseIP(strings.TrimSpace(raw))
	return ip != nil && ip.IsLoopback()
}

func newRequestID() string {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "req-fallback"
	}
	return "req-" + hex.EncodeToString(data[:])
}
