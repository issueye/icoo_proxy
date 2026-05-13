package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
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
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,x-api-key,x-request-id")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}
}

func AdminAuth(auth AuthVerifier, allowLocalWithoutAuth bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if allowLocalWithoutAuth && isLocalClient(ctx.ClientIP()) {
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
