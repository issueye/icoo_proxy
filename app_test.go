package main

import (
	"strings"
	"testing"

	appcore "icoo_proxy/internal/app"
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

func TestDeleteSupplierRejectsEnabledPolicyReference(t *testing.T) {
	root := t.TempDir()
	core, err := appcore.NewApp(root)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	record, err := core.Services().Supplier().Upsert(models.SupplierUpsertInput{
		Name:     "Policy Vendor",
		Protocol: consts.ProtocolOpenAIResponses.ToString(),
		BaseURL:  "https://example.com",
		Enabled:  true,
		Models: []models.SupplierModelItem{{
			Name:      "gpt-4.1-mini",
			MaxTokens: 32768,
		}},
		DefaultModel: "gpt-4.1-mini",
		UserAgent:    "PolicyVendor/1.0",
	})
	if err != nil {
		t.Fatalf("upsert supplier: %v", err)
	}

	if _, err := core.Services().RoutePolicy().Upsert(models.UpsertInput{
		DownstreamProtocol: consts.ProtocolOpenAIChat,
		SupplierID:         record.ID,
		Enabled:            true,
	}); err != nil {
		t.Fatalf("upsert policy: %v", err)
	}

	app := &App{
		root: root,
		app:  core,
	}

	_, err = app.DeleteSupplier(record.ID)
	if err == nil {
		t.Fatalf("expected delete supplier to be blocked")
	}
	if !strings.Contains(err.Error(), "supplier is used by enabled route policy \"openai-chat\"") {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if _, ok := core.Services().Supplier().Resolve(record.ID); !ok {
		t.Fatalf("expected blocked supplier deletion to keep supplier record")
	}
}
