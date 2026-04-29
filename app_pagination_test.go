package main

import (
	"testing"

	appcore "icoo_proxy/internal/app"
	"icoo_proxy/internal/consts"
	"icoo_proxy/internal/models"
)

func TestGetAuthKeysPageSupportsPaginationAndFilters(t *testing.T) {
	root := t.TempDir()
	core, err := appcore.NewApp(root)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	inputs := []models.AuthKeyUpsertInput{
		{Name: "Alpha Key", Secret: "alpha-secret-123456", Enabled: true, Description: "alpha access"},
		{Name: "Beta Key", Secret: "beta-secret-123456", Enabled: false, Description: "beta access"},
		{Name: "Gamma Key", Secret: "gamma-secret-123456", Enabled: true, Description: "gamma access"},
	}
	for _, input := range inputs {
		if _, err := core.Services().AuthKey().Upsert(input); err != nil {
			t.Fatalf("upsert auth key %q: %v", input.Name, err)
		}
	}

	app := &App{root: root, app: core}
	result := app.GetAuthKeysPage(1, 2, "alpha", "enabled")

	if result.TotalCount != 3 {
		t.Fatalf("expected total count 3, got %d", result.TotalCount)
	}
	if result.EnabledCount != 2 {
		t.Fatalf("expected enabled count 2, got %d", result.EnabledCount)
	}
	if result.Total != 1 {
		t.Fatalf("expected filtered total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "Alpha Key" {
		t.Fatalf("expected Alpha Key page result, got %#v", result.Items)
	}
}

func TestGetEndpointsPageSupportsPaginationAndFilters(t *testing.T) {
	root := t.TempDir()
	core, err := appcore.NewApp(root)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	if _, err := core.Services().Endpoint().Upsert(models.EndpointUpsertInput{
		Path:        "/custom/v1/chat/completions",
		Protocol:    consts.ProtocolOpenAIChat.ToString(),
		Description: "custom chat endpoint",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("upsert endpoint: %v", err)
	}

	app := &App{root: root, app: core}
	result := app.GetEndpointsPage(1, 5, "custom", consts.ProtocolOpenAIChat.ToString())

	if result.TotalCount != 7 {
		t.Fatalf("expected total count 7, got %d", result.TotalCount)
	}
	if result.CustomCount != 1 {
		t.Fatalf("expected custom count 1, got %d", result.CustomCount)
	}
	if result.Total != 1 {
		t.Fatalf("expected filtered total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Path != "/custom/v1/chat/completions" {
		t.Fatalf("expected custom endpoint page result, got %#v", result.Items)
	}
}

func TestGetSuppliersPageSupportsPaginationAndFilters(t *testing.T) {
	root := t.TempDir()
	core, err := appcore.NewApp(root)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	t.Cleanup(func() { _ = core.Close() })

	inputs := []models.SupplierUpsertInput{
		{
			Name:     "Alpha Supplier",
			Protocol: consts.ProtocolOpenAIChat.ToString(),
			BaseURL:  "https://alpha.example.com",
			Enabled:  true,
			Models: []models.SupplierModelItem{{
				Name:      "gpt-4.1-mini",
				MaxTokens: 32768,
			}},
			DefaultModel: "gpt-4.1-mini",
		},
		{
			Name:     "Beta Supplier",
			Protocol: consts.ProtocolAnthropic.ToString(),
			BaseURL:  "https://beta.example.com",
			Enabled:  false,
			Models: []models.SupplierModelItem{{
				Name:      "claude-3-5-sonnet",
				MaxTokens: 32768,
			}},
			DefaultModel: "claude-3-5-sonnet",
		},
	}
	for _, input := range inputs {
		if _, err := core.Services().Supplier().Upsert(input); err != nil {
			t.Fatalf("upsert supplier %q: %v", input.Name, err)
		}
	}

	app := &App{root: root, app: core}
	result := app.GetSuppliersPage(1, 1, "alpha", consts.ProtocolOpenAIChat.ToString())

	if result.TotalCount != 2 {
		t.Fatalf("expected total count 2, got %d", result.TotalCount)
	}
	if result.EnabledCount != 1 {
		t.Fatalf("expected enabled count 1, got %d", result.EnabledCount)
	}
	if result.Total != 1 {
		t.Fatalf("expected filtered total 1, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "Alpha Supplier" {
		t.Fatalf("expected Alpha Supplier page result, got %#v", result.Items)
	}
}
