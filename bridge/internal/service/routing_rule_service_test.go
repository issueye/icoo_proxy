package service

import (
	"context"
	"strings"
	"testing"

	"github.com/issueye/icoo_proxy/common/constants"
	"github.com/issueye/icoo_proxy/bridge/internal/model/entity"
)

type memoryRoutingRuleRepo struct {
	items map[string]entity.RoutingRule
	saved *entity.RoutingRule
}

func newMemoryRoutingRuleRepo() *memoryRoutingRuleRepo {
	return &memoryRoutingRuleRepo{items: make(map[string]entity.RoutingRule)}
}

func (r *memoryRoutingRuleRepo) List(context.Context) ([]entity.RoutingRule, error) {
	items := make([]entity.RoutingRule, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return items, nil
}

func (r *memoryRoutingRuleRepo) ListEnabled(context.Context) ([]entity.RoutingRule, error) {
	items := make([]entity.RoutingRule, 0, len(r.items))
	for _, item := range r.items {
		if item.Enabled {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *memoryRoutingRuleRepo) Find(_ context.Context, id string) (entity.RoutingRule, error) {
	return r.items[id], nil
}

func (r *memoryRoutingRuleRepo) Save(_ context.Context, item *entity.RoutingRule) error {
	copied := *item
	r.items[item.ID] = copied
	r.saved = &copied
	return nil
}

func (r *memoryRoutingRuleRepo) Delete(_ context.Context, id string) error {
	delete(r.items, id)
	return nil
}

func TestRoutingRuleUpsertRequiresForceForActiveRule(t *testing.T) {
	repo := newMemoryRoutingRuleRepo()
	tracker := NewRequestTracker()
	tracker.Acquire("rule-active")
	service := NewRoutingRuleService(repo, tracker)

	_, err := service.Upsert(context.Background(), RoutingRuleUpsertInput{
		ID:                "rule-active",
		Name:              "default",
		Priority:          100,
		MatchProtocol:     constants.ProtocolAnthropic,
		MatchModelPattern: "*",
		UpstreamProtocol:  constants.ProtocolOpenAIResponses,
		TargetProviderID:  "provider-next",
		Enabled:           true,
	})
	if err == nil || !strings.Contains(err.Error(), "active requests") {
		t.Fatalf("expected active request error, got %v", err)
	}
	if repo.saved != nil {
		t.Fatalf("rule was saved without force: %+v", repo.saved)
	}

	item, err := service.Upsert(context.Background(), RoutingRuleUpsertInput{
		ID:                "rule-active",
		Name:              "default",
		Priority:          100,
		MatchProtocol:     constants.ProtocolAnthropic,
		MatchModelPattern: "*",
		UpstreamProtocol:  constants.ProtocolOpenAIResponses,
		TargetProviderID:  "provider-next",
		Enabled:           true,
		Force:             true,
	})
	if err != nil {
		t.Fatalf("force upsert failed: %v", err)
	}
	if item.TargetProviderID != "provider-next" || repo.saved == nil {
		t.Fatalf("force upsert did not save rule: item=%+v saved=%+v", item, repo.saved)
	}
}
