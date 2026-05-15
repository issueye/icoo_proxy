package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"icoo_llm_bridge/internal/constants"
	"icoo_llm_bridge/internal/model/entity"
	"icoo_llm_bridge/internal/repository"
	"icoo_llm_bridge/internal/utils/idgen"
)

type AuthService interface {
	Verify(ctx context.Context, secret string, scope string) bool
	ListKeys(ctx context.Context) ([]APIKeyView, error)
	GetKeySecret(ctx context.Context, id string) (APIKeySecretView, error)
	CreateKey(ctx context.Context, input APIKeyCreateInput) (APIKeyView, error)
	DeleteKey(ctx context.Context, id string) error
}

type ProviderService interface {
	List(ctx context.Context) ([]entity.Provider, error)
	Upsert(ctx context.Context, input ProviderUpsertInput) (entity.Provider, error)
	Delete(ctx context.Context, id string) error
}

type ProviderModelService interface {
	ListByProvider(ctx context.Context, providerID string) ([]entity.ProviderModel, error)
	Upsert(ctx context.Context, input ProviderModelUpsertInput) (entity.ProviderModel, error)
	Delete(ctx context.Context, providerID string, id string) error
}

type RoutingRuleService interface {
	List(ctx context.Context) ([]entity.RoutingRule, error)
	Upsert(ctx context.Context, input RoutingRuleUpsertInput) (entity.RoutingRule, error)
	Delete(ctx context.Context, id string) error
}

type TrafficService interface {
	Record(ctx context.Context, item entity.TrafficRecord) error
	List(ctx context.Context, limit int) ([]entity.TrafficRecord, error)
	Clear(ctx context.Context) error
}

type authService struct {
	repo repository.APIKeyRepository
}

func NewAuthService(repo repository.APIKeyRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Verify(ctx context.Context, secret string, scope string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	sum := hashSecret(secret)
	keys, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return false
	}
	for _, key := range keys {
		if !scopeAllowed(key.Scopes, scope) {
			continue
		}
		if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(key.SecretHash), []byte(sum)) == 1 {
			return true
		}
	}
	return false
}

func (s *authService) ListKeys(ctx context.Context) ([]APIKeyView, error) {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]APIKeyView, 0, len(items))
	for _, item := range items {
		views = append(views, toAPIKeyView(item))
	}
	return views, nil
}

func (s *authService) GetKeySecret(ctx context.Context, id string) (APIKeySecretView, error) {
	item, err := s.repo.Find(ctx, strings.TrimSpace(id))
	if err != nil {
		return APIKeySecretView{}, err
	}
	secret := strings.TrimSpace(item.SecretCipher)
	if secret == "" {
		return APIKeySecretView{}, fmt.Errorf("该 Key 创建于旧版本，明文不可恢复，请重新生成后再复制")
	}
	return APIKeySecretView{Secret: secret}, nil
}

func (s *authService) CreateKey(ctx context.Context, input APIKeyCreateInput) (APIKeyView, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return APIKeyView{}, fmt.Errorf("name is required")
	}
	id := strings.TrimSpace(input.ID)
	scopes := strings.TrimSpace(input.Scopes)
	if scopes == "" {
		scopes = "proxy"
	}

	var (
		item entity.APIKey
		err  error
	)
	if id != "" {
		item, err = s.repo.Find(ctx, id)
		if err != nil {
			return APIKeyView{}, err
		}
	} else {
		now := time.Now()
		item = entity.APIKey{
			ID:        idgen.New("key"),
			CreatedAt: now,
		}
	}

	secret := strings.TrimSpace(input.Secret)
	generated := false
	if secret == "" && id == "" {
		secret = idgen.New("sk")
		generated = true
	}
	if secret != "" {
		item.SecretHash = hashSecret(secret)
		item.SecretPreview = previewSecret(secret)
		item.SecretCipher = secret
	} else if item.SecretHash == "" {
		return APIKeyView{}, fmt.Errorf("secret is required")
	}

	item.Name = name
	item.Scopes = scopes
	item.Enabled = input.Enabled
	item.UpdatedAt = time.Now()

	if err := s.repo.Save(ctx, &item); err != nil {
		return APIKeyView{}, err
	}
	view := toAPIKeyView(item)
	if generated || secret != "" {
		view.SecretPreview = secret
	}
	return view, nil
}

func (s *authService) DeleteKey(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

type providerService struct {
	repo repository.ProviderRepository
}

func NewProviderService(repo repository.ProviderRepository) ProviderService {
	return &providerService{repo: repo}
}

func (s *providerService) List(ctx context.Context) ([]entity.Provider, error) {
	return s.repo.List(ctx)
}

func (s *providerService) Upsert(ctx context.Context, input ProviderUpsertInput) (entity.Provider, error) {
	if strings.TrimSpace(input.Name) == "" {
		return entity.Provider{}, fmt.Errorf("name is required")
	}
	if _, ok := constants.ParseProtocol(input.Protocol.String()); !ok {
		return entity.Provider{}, fmt.Errorf("protocol is invalid")
	}
	now := time.Now()
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = idgen.New("provider")
	}
	item := entity.Provider{
		ID:           id,
		Name:         strings.TrimSpace(input.Name),
		Protocol:     input.Protocol,
		Vendor:       input.Vendor,
		BaseURL:      strings.TrimSpace(input.BaseURL),
		APIKeyCipher: strings.TrimSpace(input.APIKey),
		OnlyStream:   input.OnlyStream,
		UserAgent:    strings.TrimSpace(input.UserAgent),
		Enabled:      input.Enabled,
		Description:  strings.TrimSpace(input.Description),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Save(ctx, &item); err != nil {
		return entity.Provider{}, err
	}
	return item, nil
}

func (s *providerService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, strings.TrimSpace(id))
}

type providerModelService struct {
	repo repository.ProviderModelRepository
}

func NewProviderModelService(repo repository.ProviderModelRepository) ProviderModelService {
	return &providerModelService{repo: repo}
}

func (s *providerModelService) ListByProvider(ctx context.Context, providerID string) ([]entity.ProviderModel, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, fmt.Errorf("provider_id is required")
	}
	return s.repo.ListByProvider(ctx, providerID)
}

func (s *providerModelService) Upsert(ctx context.Context, input ProviderModelUpsertInput) (entity.ProviderModel, error) {
	providerID := strings.TrimSpace(input.ProviderID)
	if providerID == "" {
		return entity.ProviderModel{}, fmt.Errorf("provider_id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return entity.ProviderModel{}, fmt.Errorf("name is required")
	}
	now := time.Now()
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = idgen.New("model")
	}
	item := entity.ProviderModel{
		ID:         id,
		ProviderID: providerID,
		Name:       name,
		MaxTokens:  input.MaxTokens,
		Enabled:    input.Enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.Save(ctx, &item); err != nil {
		return entity.ProviderModel{}, err
	}
	return item, nil
}

func (s *providerModelService) Delete(ctx context.Context, providerID string, id string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return fmt.Errorf("provider_id is required")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	return s.repo.Delete(ctx, providerID, id)
}

type routingRuleService struct {
	repo    repository.RoutingRuleRepository
	tracker RequestTracker
}

func NewRoutingRuleService(repo repository.RoutingRuleRepository, tracker RequestTracker) RoutingRuleService {
	return &routingRuleService{repo: repo, tracker: tracker}
}

func (s *routingRuleService) List(ctx context.Context) ([]entity.RoutingRule, error) {
	return s.repo.List(ctx)
}

func (s *routingRuleService) Upsert(ctx context.Context, input RoutingRuleUpsertInput) (entity.RoutingRule, error) {
	if strings.TrimSpace(input.Name) == "" {
		return entity.RoutingRule{}, fmt.Errorf("name is required")
	}
	id := strings.TrimSpace(input.ID)
	if id != "" && s.tracker != nil && s.tracker.ActiveCount(id) > 0 {
		return entity.RoutingRule{}, fmt.Errorf("routing rule is currently handling active requests, cannot modify")
	}
	now := time.Now()
	if id == "" {
		id = idgen.New("rule")
	}
	item := entity.RoutingRule{
		ID:                id,
		Name:              strings.TrimSpace(input.Name),
		Priority:          input.Priority,
		MatchProtocol:     input.MatchProtocol,
		MatchModelPattern: strings.TrimSpace(input.MatchModelPattern),
		UpstreamProtocol:  input.UpstreamProtocol,
		TargetProviderID:  strings.TrimSpace(input.TargetProviderID),
		TargetModel:       strings.TrimSpace(input.TargetModel),
		Enabled:           input.Enabled,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.repo.Save(ctx, &item); err != nil {
		return entity.RoutingRule{}, err
	}
	return item, nil
}

func (s *routingRuleService) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if s.tracker != nil && s.tracker.ActiveCount(id) > 0 {
		return fmt.Errorf("routing rule is currently handling active requests, cannot delete")
	}
	return s.repo.Delete(ctx, id)
}

type trafficService struct {
	repo repository.TrafficRepository
}

func NewTrafficService(repo repository.TrafficRepository) TrafficService {
	return &trafficService{repo: repo}
}

func (s *trafficService) List(ctx context.Context, limit int) ([]entity.TrafficRecord, error) {
	return s.repo.List(ctx, limit)
}

func (s *trafficService) Record(ctx context.Context, item entity.TrafficRecord) error {
	return s.repo.Record(ctx, &item)
}

func (s *trafficService) Clear(ctx context.Context) error {
	return s.repo.Clear(ctx)
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func previewSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}

func scopeAllowed(scopes string, scope string) bool {
	scope = strings.TrimSpace(scope)
	for _, part := range strings.Split(scopes, ",") {
		value := strings.TrimSpace(part)
		if value == "*" || value == scope {
			return true
		}
	}
	return false
}

func toAPIKeyView(item entity.APIKey) APIKeyView {
	return APIKeyView{
		ID:            item.ID,
		Name:          item.Name,
		SecretPreview: item.SecretPreview,
		CanReveal:     strings.TrimSpace(item.SecretCipher) != "",
		Scopes:        item.Scopes,
		Enabled:       item.Enabled,
		CreatedAt:     item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     item.UpdatedAt.Format(time.RFC3339),
	}
}
