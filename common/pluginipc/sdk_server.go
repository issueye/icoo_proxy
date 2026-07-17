package pluginipc

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// PluginMeta describes the plugin identity returned at handshake.
type PluginMeta struct {
	ID               string
	Version          string
	UpstreamKind     string
	Capabilities     []string
	SupportedIngress []string
	AdminBaseURL     string
	// AdminToken is host-only; injected by bridge UI reverse-proxy.
	AdminToken string
	UIPages    []UIPage
}

// PluginEnv is the host-provided process environment for a plugin.
type PluginEnv struct {
	Endpoint string
	DataDir  string
	PluginID string
	Token    string
}

// ParsePluginFlags parses standard plugin CLI flags and env.
//
//	--endpoint   (default ICOO_PLUGIN_ENDPOINT)
//	--data-dir
//	--plugin-id
//
// Token always comes from ICOO_PLUGIN_TOKEN.
//
// Callers that need extra flags should register them on flag.CommandLine
// before invoking ParsePluginFlags. If flags are already parsed, values are
// read from the existing FlagSet.
func ParsePluginFlags() (PluginEnv, error) {
	ensureStringFlag("endpoint", os.Getenv("ICOO_PLUGIN_ENDPOINT"), "IPC endpoint")
	ensureStringFlag("data-dir", "", "plugin data directory")
	ensureStringFlag("plugin-id", "", "plugin id")
	if !flag.Parsed() {
		flag.Parse()
	}
	env := PluginEnv{
		Endpoint: flag.Lookup("endpoint").Value.String(),
		DataDir:  flag.Lookup("data-dir").Value.String(),
		PluginID: flag.Lookup("plugin-id").Value.String(),
		Token:    os.Getenv("ICOO_PLUGIN_TOKEN"),
	}
	if env.Endpoint == "" || env.Token == "" {
		return env, fmt.Errorf("endpoint and ICOO_PLUGIN_TOKEN are required")
	}
	return env, nil
}

func ensureStringFlag(name, value, usage string) {
	if flag.Lookup(name) == nil {
		flag.String(name, value, usage)
	}
}

// PluginHooks customizes RunPlugin lifecycle.
type PluginHooks struct {
	// Env overrides ParsePluginFlags when non-nil.
	Env *PluginEnv
	// AfterListen runs after Listen succeeds and before Accept.
	// Use for admin UI / light init; do not block longer than host dial timeout.
	AfterListen func(ctx context.Context, env PluginEnv) error
	// PrepareHandshake runs after AfterListen (and Accept) and before ServeConn.
	// Use it to inject AdminBaseURL / UIPages discovered during AfterListen.
	// When nil, HandshakeFrom(meta) uses the call-time PluginMeta snapshot.
	PrepareHandshake func(env PluginEnv, meta PluginMeta) (PluginMeta, error)
	// OnShutdown is invoked on plugin.shutdown (in addition to connection close).
	OnShutdown func()
	// Health overrides the default healthy handler when non-nil.
	Health func(ctx context.Context) (*HealthResult, error)
	// MaxFrameBytes / InlineBodyLimit forwarded to ServerOptions.
	MaxFrameBytes   int
	InlineBodyLimit int
	// NoSignal disables signal.NotifyContext (useful in tests).
	NoSignal bool
	// Context overrides the root context (tests). When nil, Background (+signals) is used.
	Context context.Context
}

// StreamHandler is the unified prepare+run stream callback.
// Prefer returning a closure that captures prepare state (eliminates pendingRuns maps).
type StreamHandler func(ctx context.Context, req ProxyRequest) (
	open *StreamOpenResult,
	errResp *ProxyResponse,
	run func(ctx context.Context, w *StreamWriter),
	err error,
)

// Handlers are business callbacks registered by RunPlugin / ServeConn.
type Handlers struct {
	Complete   func(ctx context.Context, req ProxyRequest) (*ProxyResponse, error)
	Stream     StreamHandler
	ModelsList func(ctx context.Context) (*ModelsListResult, error)
}

// ServeConn wires handlers onto an already-accepted connection and returns the Server.
// Handshake/ping/shutdown defaults are installed by NewServer.
func ServeConn(raw net.Conn, opts ServerOptions, handlers Handlers, health func(ctx context.Context) (*HealthResult, error)) *Server {
	srv := NewServer(raw, opts)
	if handlers.Complete != nil {
		srv.RegisterComplete(handlers.Complete)
	}
	if handlers.Stream != nil {
		srv.RegisterProxyStreamEx(handlers.Stream)
	}
	if handlers.ModelsList != nil {
		srv.RegisterModelsList(handlers.ModelsList)
	}
	if health != nil {
		srv.RegisterHealth(health)
	}
	return srv
}

// RunPlugin is the high-level plugin bootstrap:
//
//	parse flags/env → Listen (immediately) → AfterListen → Accept → Serve → wait
//
// Listen happens before AfterListen so the host can dial while the plugin finishes
// light initialization (admin UI, credential store, etc.).
func RunPlugin(meta PluginMeta, handlers Handlers, hooks PluginHooks) error {
	var env PluginEnv
	var err error
	if hooks.Env != nil {
		env = *hooks.Env
	} else {
		env, err = ParsePluginFlags()
		if err != nil {
			return err
		}
	}
	if env.Endpoint == "" || env.Token == "" {
		return fmt.Errorf("endpoint and ICOO_PLUGIN_TOKEN are required")
	}
	if env.PluginID != "" {
		// Host-supplied plugin-id wins for handshake identity.
		meta.ID = env.PluginID
	}

	ctx := hooks.Context
	if ctx == nil {
		ctx = context.Background()
	}
	stop := func() {}
	if !hooks.NoSignal {
		ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
	}

	ln, err := Listen(ctx, ListenConfig{Endpoint: env.Endpoint})
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer ln.Close()

	if hooks.AfterListen != nil {
		if err := hooks.AfterListen(ctx, env); err != nil {
			return err
		}
	}

	type acceptResult struct {
		conn net.Conn
		err  error
	}
	ch := make(chan acceptResult, 1)
	go func() {
		c, e := ln.Accept()
		ch <- acceptResult{c, e}
	}()

	var conn net.Conn
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ar := <-ch:
		if ar.err != nil {
			return fmt.Errorf("accept: %w", ar.err)
		}
		conn = ar.conn
	}

	onShutdown := func() {
		stop()
		if hooks.OnShutdown != nil {
			hooks.OnShutdown()
		}
		_ = conn.Close()
	}

	if hooks.PrepareHandshake != nil {
		meta, err = hooks.PrepareHandshake(env, meta)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("prepare handshake: %w", err)
		}
	}

	opts := ServerOptions{
		MaxFrameBytes:   hooks.MaxFrameBytes,
		InlineBodyLimit: hooks.InlineBodyLimit,
		HostToken:       env.Token,
		Handshake:       HandshakeFrom(meta),
		OnShutdown:      onShutdown,
	}
	srv := ServeConn(conn, opts, handlers, hooks.Health)

	select {
	case <-ctx.Done():
	case <-srv.Conn().Done():
	}
	_ = srv.Close()
	return nil
}
