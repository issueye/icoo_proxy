// mockplugin is a process plugin used for pluginhost integration tests.
// It demonstrates the pluginipc Server SDK (RunPlugin + helpers).
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/issueye/icoo_proxy/common/pluginipc"
)

func main() {
	err := pluginipc.RunPlugin(
		pluginipc.PluginMeta{
			ID:           "mock",
			Version:      "0.0.1-mock",
			UpstreamKind: "mock",
			Capabilities: []string{
				pluginipc.CapProxyComplete,
				pluginipc.CapProxyStream,
				pluginipc.CapModelsList,
				pluginipc.CapHealth,
			},
			SupportedIngress: []string{"anthropic", "openai-chat", "openai-responses"},
		},
		pluginipc.Handlers{
			Complete: func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
				body := append([]byte(`{"echo":true,"bytes":`), append([]byte(fmt.Sprintf("%d", len(req.Body))), '}')...)
				return pluginipc.OKJSON(body, nil), nil
			},
			Stream: func(ctx context.Context, req pluginipc.ProxyRequest) (
				*pluginipc.StreamOpenResult,
				*pluginipc.ProxyResponse,
				func(context.Context, *pluginipc.StreamWriter),
				error,
			) {
				open := pluginipc.SSEOpen(pluginipc.NewStreamID("mock"))
				run := func(ctx context.Context, w *pluginipc.StreamWriter) {
					_ = w.WriteChunk([]byte("data: {\"type\":\"message\"}\n\n"))
					_ = w.End(&pluginipc.Usage{OutputTokens: 1})
				}
				return open, nil, run, nil
			},
			ModelsList: func(ctx context.Context) (*pluginipc.ModelsListResult, error) {
				return &pluginipc.ModelsListResult{
					Models: []pluginipc.ModelInfo{{ID: "mock-model"}},
				}, nil
			},
		},
		pluginipc.PluginHooks{},
	)
	if err != nil {
		log.Fatal(err)
	}
}
