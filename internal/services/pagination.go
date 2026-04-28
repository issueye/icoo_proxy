package services

import (
	"strings"

	"icoo_proxy/internal/models"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

type AuthKeyPageResult struct {
	Items        []models.AuthKeyRecord `json:"items"`
	Total        int                    `json:"total"`
	Page         int                    `json:"page"`
	PageSize     int                    `json:"page_size"`
	TotalCount   int                    `json:"total_count"`
	EnabledCount int                    `json:"enabled_count"`
}

type EndpointPageResult struct {
	Items        []models.EndpointRecord `json:"items"`
	Total        int                     `json:"total"`
	Page         int                     `json:"page"`
	PageSize     int                     `json:"page_size"`
	TotalCount   int                     `json:"total_count"`
	EnabledCount int                     `json:"enabled_count"`
	CustomCount  int                     `json:"custom_count"`
}

type SupplierPageResult struct {
	Items        []models.SupplierRecord `json:"items"`
	Total        int                     `json:"total"`
	Page         int                     `json:"page"`
	PageSize     int                     `json:"page_size"`
	TotalCount   int                     `json:"total_count"`
	EnabledCount int                     `json:"enabled_count"`
}

func normalizePage(page int) int {
	if page <= 0 {
		return defaultPage
	}
	return page
}

func normalizePageSize(pageSize int) int {
	if pageSize <= 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
}

func clampPage(page int, total int64, pageSize int) int {
	if total <= 0 {
		return defaultPage
	}
	maxPage := int((total + int64(pageSize) - 1) / int64(pageSize))
	if page > maxPage {
		return maxPage
	}
	return page
}

func normalizeKeyword(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
