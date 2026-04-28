package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"icoo_proxy/internal/models"
)

type AuthKeyService struct {
	db *gorm.DB
}

func NewAuthKeyService(db *gorm.DB) (*AuthKeyService, error) {
	return &AuthKeyService{db: db}, nil
}

func (s *AuthKeyService) List() []models.AuthKeyRecord {
	var rows []models.AuthKeyModel
	if err := s.db.Order("lower(name) asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]models.AuthKeyRecord, 0, len(rows))
	for _, item := range rows {
		items = append(items, toAuthKeyRecord(item))
	}
	return items
}

func (s *AuthKeyService) QueryPage(page int, pageSize int, keyword string, status string) AuthKeyPageResult {
	page = normalizePage(page)
	pageSize = normalizePageSize(pageSize)
	keyword = normalizeKeyword(keyword)
	status = normalizeAuthKeyStatus(status)

	result := AuthKeyPageResult{
		Items:    make([]models.AuthKeyRecord, 0, pageSize),
		Page:     page,
		PageSize: pageSize,
	}

	if s == nil || s.db == nil {
		return result
	}

	var totalCount int64
	_ = s.db.Model(&models.AuthKeyModel{}).Count(&totalCount).Error
	result.TotalCount = int(totalCount)

	var enabledCount int64
	_ = s.db.Model(&models.AuthKeyModel{}).Where("enabled = ?", true).Count(&enabledCount).Error
	result.EnabledCount = int(enabledCount)

	query := s.db.Model(&models.AuthKeyModel{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("lower(name) LIKE ? OR lower(description) LIKE ?", like, like)
	}

	switch status {
	case "enabled":
		query = query.Where("enabled = ?", true)
	case "disabled":
		query = query.Where("enabled = ?", false)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return result
	}

	result.Total = int(total)
	result.Page = clampPage(page, total, pageSize)

	var rows []models.AuthKeyModel
	if err := query.
		Order("lower(name) asc").
		Offset((result.Page - 1) * result.PageSize).
		Limit(result.PageSize).
		Find(&rows).Error; err != nil {
		return result
	}

	for _, item := range rows {
		result.Items = append(result.Items, toAuthKeyRecord(item))
	}
	return result
}

func (s *AuthKeyService) EnabledSecrets() []string {
	var rows []models.AuthKeyModel
	if err := s.db.Where("enabled = ?", true).Order("lower(name) asc").Find(&rows).Error; err != nil {
		return nil
	}
	items := make([]string, 0, len(rows))
	for _, item := range rows {
		if secret := strings.TrimSpace(item.Secret); secret != "" {
			items = append(items, secret)
		}
	}
	return items
}

func (s *AuthKeyService) Upsert(input models.AuthKeyUpsertInput) (models.AuthKeyRecord, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return models.AuthKeyRecord{}, fmt.Errorf("auth key name is required")
	}
	secret := strings.TrimSpace(input.Secret)

	id := strings.TrimSpace(input.ID)
	var existing models.AuthKeyModel
	found := false
	if id != "" {
		found = s.db.Limit(1).Find(&existing, "id = ?", id).RowsAffected > 0
	}
	if !found && secret != "" {
		found = s.db.Limit(1).Find(&existing, "secret = ?", secret).RowsAffected > 0
	}
	if !found && secret == "" {
		secret = generateSecret()
	}
	if found && secret == "" {
		secret = existing.Secret
	}
	if secret == "" {
		return models.AuthKeyRecord{}, fmt.Errorf("auth key secret is required")
	}

	now := time.Now()
	current := models.AuthKeyModel{
		ID:          generateID(name),
		Name:        name,
		Secret:      secret,
		Enabled:     input.Enabled,
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if found {
		current.ID = existing.ID
		current.CreatedAt = existing.CreatedAt
	} else if id != "" {
		current.ID = id
	}
	if err := s.db.Save(&current).Error; err != nil {
		return models.AuthKeyRecord{}, err
	}
	return toAuthKeyRecord(current), nil
}

func (s *AuthKeyService) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("auth key id is required")
	}
	result := s.db.Delete(&models.AuthKeyModel{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("auth key not found")
	}
	return nil
}

func (s *AuthKeyService) GetSecret(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("auth key id is required")
	}
	var item models.AuthKeyModel
	if err := s.db.First(&item, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("auth key not found")
		}
		return "", err
	}
	return item.Secret, nil
}

func toAuthKeyRecord(item models.AuthKeyModel) models.AuthKeyRecord {
	return models.AuthKeyRecord{
		ID:           item.ID,
		Name:         item.Name,
		SecretMasked: maskSecret(item.Secret),
		Enabled:      item.Enabled,
		Description:  item.Description,
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
	}
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 10 {
		return strings.Repeat("*", len(value))
	}
	return value[:6] + strings.Repeat("*", 6) + value[len(value)-4:]
}

func generateID(name string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == ' ':
			b.WriteRune('-')
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "auth-key"
	}
	return fmt.Sprintf("%s-%d", id, time.Now().UnixNano())
}

func generateSecret() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("icoo_%d", time.Now().UnixNano())
	}
	return "icoo_" + hex.EncodeToString(buf)
}

func normalizeAuthKeyStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "enabled", "disabled":
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return "all"
	}
}

func MergeSecrets(groups ...[]string) []string {
	values := make([]string, 0)
	for _, group := range groups {
		for _, item := range group {
			for _, part := range strings.Split(item, ",") {
				value := strings.TrimSpace(part)
				if value != "" && !slices.Contains(values, value) {
					values = append(values, value)
				}
			}
		}
	}
	return values
}
