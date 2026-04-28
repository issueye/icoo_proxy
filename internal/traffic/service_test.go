package traffic

import (
	"testing"
	"time"

	"icoo_proxy/internal/api"
)

func TestServiceRecordsAndListsRecentRequestsNewestFirst(t *testing.T) {
	root := t.TempDir()
	service, err := NewService(root)
	if err != nil {
		t.Fatalf("new traffic service: %v", err)
	}
	defer func() { _ = service.Close() }()

	first := api.RequestView{
		RequestID:  "req-first",
		Downstream: "openai-chat",
		Upstream:   "openai-responses",
		Model:      "gpt-4.1-mini",
		StatusCode: 200,
		DurationMS: 42,
		CreatedAt:  time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	second := api.RequestView{
		RequestID:  "req-second",
		Downstream: "anthropic",
		Upstream:   "openai-responses",
		Model:      "claude-sonnet",
		StatusCode: 502,
		DurationMS: 110,
		Error:      "upstream failed",
		CreatedAt:  time.Date(2026, 4, 27, 10, 1, 0, 0, time.UTC).Format(time.RFC3339),
	}

	if err := service.RecordRequest(first); err != nil {
		t.Fatalf("record first: %v", err)
	}
	if err := service.RecordRequest(second); err != nil {
		t.Fatalf("record second: %v", err)
	}

	got := service.ListRecent(10)
	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if got[0].RequestID != "req-second" || got[1].RequestID != "req-first" {
		t.Fatalf("expected newest first, got %#v", got)
	}
	if got[0].Error != "upstream failed" {
		t.Fatalf("expected error field preserved, got %#v", got[0])
	}

	if err := service.Close(); err != nil {
		t.Fatalf("close traffic service: %v", err)
	}
	reopened, err := NewService(root)
	if err != nil {
		t.Fatalf("reopen traffic service: %v", err)
	}
	defer func() { _ = reopened.Close() }()

	got = reopened.ListRecent(10)
	if len(got) != 2 {
		t.Fatalf("expected 2 persisted records, got %d", len(got))
	}
	if got[0].RequestID != "req-second" || got[1].RequestID != "req-first" {
		t.Fatalf("expected persisted records newest first, got %#v", got)
	}
}

func TestServiceTokenStatsAndPersistence(t *testing.T) {
	root := t.TempDir()
	service, err := NewService(root)
	if err != nil {
		t.Fatalf("new traffic service: %v", err)
	}
	defer func() { _ = service.Close() }()

	items := []api.RequestView{
		{
			RequestID:    "req-usage-1",
			Downstream:   "openai-chat",
			Upstream:     "openai-responses",
			Model:        "gpt-4.1-mini",
			StatusCode:   200,
			DurationMS:   21,
			InputTokens:  11,
			OutputTokens: 7,
			TotalTokens:  18,
			CreatedAt:    time.Date(2026, 4, 27, 10, 2, 0, 0, time.UTC).Format(time.RFC3339),
		},
		{
			RequestID:    "req-usage-2",
			Downstream:   "anthropic",
			Upstream:     "anthropic",
			Model:        "claude-sonnet",
			StatusCode:   200,
			DurationMS:   33,
			InputTokens:  5,
			OutputTokens: 13,
			TotalTokens:  18,
			CreatedAt:    time.Date(2026, 4, 27, 10, 3, 0, 0, time.UTC).Format(time.RFC3339),
		},
	}

	for _, item := range items {
		if err := service.RecordRequest(item); err != nil {
			t.Fatalf("record request %s: %v", item.RequestID, err)
		}
	}

	got := service.ListRecent(10)
	if len(got) != 2 {
		t.Fatalf("expected 2 records, got %d", len(got))
	}
	if got[0].RequestID != "req-usage-2" {
		t.Fatalf("expected newest request first, got %#v", got[0])
	}
	if got[0].InputTokens != 5 || got[0].OutputTokens != 13 || got[0].TotalTokens != 18 {
		t.Fatalf("expected persisted token fields, got %#v", got[0])
	}

	stats := service.TokenStats()
	if stats.InputTokens != 16 || stats.OutputTokens != 20 || stats.TotalTokens != 36 {
		t.Fatalf("expected stats 16/20/36, got %#v", stats)
	}
}

func TestServiceTokenStatsIgnoresLegacyRecordsWithoutUsage(t *testing.T) {
	root := t.TempDir()
	service, err := NewService(root)
	if err != nil {
		t.Fatalf("new traffic service: %v", err)
	}
	defer func() { _ = service.Close() }()

	legacy := api.RequestView{
		RequestID:  "req-legacy",
		Downstream: "openai-chat",
		Upstream:   "openai-chat",
		Model:      "gpt-4.1-mini",
		StatusCode: 200,
		DurationMS: 9,
		CreatedAt:  time.Date(2026, 4, 27, 10, 4, 0, 0, time.UTC).Format(time.RFC3339),
	}
	if err := service.RecordRequest(legacy); err != nil {
		t.Fatalf("record legacy request: %v", err)
	}

	stats := service.TokenStats()
	if stats.InputTokens != 0 || stats.OutputTokens != 0 || stats.TotalTokens != 0 {
		t.Fatalf("expected zero stats for legacy records, got %#v", stats)
	}

	got := service.ListRecent(10)
	if len(got) != 1 {
		t.Fatalf("expected 1 record, got %d", len(got))
	}
	if got[0].InputTokens != 0 || got[0].OutputTokens != 0 || got[0].TotalTokens != 0 {
		t.Fatalf("expected zero-value token fields for legacy record, got %#v", got[0])
	}
}

func TestServiceQueryPageSupportsFilterAndPagination(t *testing.T) {
	root := t.TempDir()
	service, err := NewService(root)
	if err != nil {
		t.Fatalf("new traffic service: %v", err)
	}
	defer func() { _ = service.Close() }()

	items := []api.RequestView{
		{
			RequestID:    "req-1",
			Downstream:   "openai-chat",
			Upstream:     "openai-responses",
			Model:        "gpt-4.1-mini",
			StatusCode:   200,
			DurationMS:   10,
			InputTokens:  3,
			OutputTokens: 7,
			TotalTokens:  10,
			CreatedAt:    time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		},
		{
			RequestID:    "req-2",
			Downstream:   "anthropic",
			Upstream:     "openai-responses",
			Model:        "claude-sonnet",
			StatusCode:   500,
			DurationMS:   20,
			InputTokens:  5,
			OutputTokens: 5,
			TotalTokens:  10,
			CreatedAt:    time.Date(2026, 4, 27, 10, 1, 0, 0, time.UTC).Format(time.RFC3339),
		},
		{
			RequestID:    "req-3",
			Downstream:   "openai-chat",
			Upstream:     "openai-chat",
			Model:        "gpt-4.1",
			StatusCode:   200,
			DurationMS:   30,
			InputTokens:  2,
			OutputTokens: 8,
			TotalTokens:  10,
			CreatedAt:    time.Date(2026, 4, 27, 10, 2, 0, 0, time.UTC).Format(time.RFC3339),
		},
	}

	for _, item := range items {
		if err := service.RecordRequest(item); err != nil {
			t.Fatalf("record request %s: %v", item.RequestID, err)
		}
	}

	result := service.QueryPage("openai-chat", 1, 1)
	if result.Total != 2 {
		t.Fatalf("expected 2 filtered records, got %d", result.Total)
	}
	if len(result.Items) != 1 || result.Items[0].RequestID != "req-3" {
		t.Fatalf("expected first page to contain newest openai-chat record, got %#v", result.Items)
	}
	if result.TotalRequests != 3 {
		t.Fatalf("expected total request summary 3, got %d", result.TotalRequests)
	}
	if result.SuccessCount != 2 || result.ErrorCount != 1 {
		t.Fatalf("expected success/error 2/1, got %d/%d", result.SuccessCount, result.ErrorCount)
	}
	if result.AverageLatency != 20 {
		t.Fatalf("expected average latency 20, got %d", result.AverageLatency)
	}
	if result.TokenStats.TotalTokens != 30 {
		t.Fatalf("expected total tokens 30, got %#v", result.TokenStats)
	}
	if len(result.ProtocolOptions) < 3 {
		t.Fatalf("expected protocol options collected, got %#v", result.ProtocolOptions)
	}

	pageTwo := service.QueryPage("openai-chat", 2, 1)
	if len(pageTwo.Items) != 1 || pageTwo.Items[0].RequestID != "req-1" {
		t.Fatalf("expected second page to contain older openai-chat record, got %#v", pageTwo.Items)
	}
}
