package controller

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/issueye/icoo_proxy/bridge/internal/service"
	"github.com/issueye/icoo_proxy/common/pluginipc"
)

// ProxyUI reverse-proxies /api/v1/plugins/:id/ui/* to the plugin admin HTTP base.
// This allows the desktop shell to embed plugin-provided SuperGrok / admin pages
// without opening a separate browser origin for each plugin port.
func (c *PluginController) ProxyUI(ctx *gin.Context) {
	id := pathID(ctx)
	base, adminToken, err := c.service.AdminProxyTarget(id)
	if err != nil {
		if errors.Is(err, service.ErrPluginUIDisabled) {
			ctx.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "PLUGIN_UI_DISABLED", "message": err.Error()},
			})
			return
		}
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"code": "PLUGIN_UI_UNAVAILABLE", "message": err.Error()},
		})
		return
	}
	target, err := url.Parse(base)
	if err != nil || target.Scheme == "" || target.Host == "" {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"code": "PLUGIN_UI_BAD_BASE", "message": "invalid plugin admin base url"},
		})
		return
	}
	// Only allow loopback targets (plugin-owned local admin).
	host := target.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		ctx.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"code": "PLUGIN_UI_FORBIDDEN", "message": "plugin admin ui must be loopback"},
		})
		return
	}

	// Strip /api/v1/plugins/:id/ui prefix.
	// Gin may deliver:
	//   /api/v1/plugins/grokbuild/ui
	//   /api/v1/plugins/grokbuild/ui/
	//   /api/v1/plugins/grokbuild/ui/api/credentials
	//   /api/v1/plugins/grokbuild/ui/*filepath param
	prefix := "/api/v1/plugins/" + id + "/ui"
	rel := strings.TrimPrefix(ctx.Request.URL.Path, prefix)
	if fp := strings.TrimSpace(ctx.Param("filepath")); fp != "" {
		// Prefer explicit catch-all param when present.
		rel = fp
		if !strings.HasPrefix(rel, "/") {
			rel = "/" + rel
		}
	}
	if rel == "" || rel == "/" {
		rel = "/"
	}
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = singleJoiningSlash(target.Path, rel)
		req.URL.RawQuery = ctx.Request.URL.RawQuery
		req.Host = target.Host
		// Avoid leaking host admin credentials to the plugin process.
		req.Header.Del("Authorization")
		req.Header.Del("X-Api-Key")
		// Inject host-held plugin admin token (never sent to the browser).
		if adminToken != "" {
			req.Header.Set(pluginipc.HeaderPluginAdminToken, adminToken)
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"code":"PLUGIN_UI_PROXY","message":"plugin ui proxy failed"}}`))
	}
	proxy.ServeHTTP(ctx.Writer, ctx.Request)
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		if a == "" {
			return b
		}
		return a + "/" + b
	}
	return a + b
}

// ListUIPages returns extension pages from all running plugins.
func (c *PluginController) ListUIPages(ctx *gin.Context) {
	items, err := c.service.ListUIPages(ctx.Request.Context())
	writeResult(ctx, items, err)
}

