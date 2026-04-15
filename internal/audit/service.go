package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"icoo_proxy/internal/appdb"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type RequestLog struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt       time.Time `json:"createdAt"`
	Method          string    `json:"method" gorm:"not null"`
	Path            string    `json:"path" gorm:"not null"`
	Model           string    `json:"model"`
	TargetModel     string    `json:"targetModel"`
	ProviderID      string    `json:"providerId"`
	ProviderName    string    `json:"providerName"`
	ProviderType    string    `json:"providerType"`
	EndpointMode    string    `json:"endpointMode"`
	UpstreamBase    string    `json:"upstreamBase"`
	UpstreamPath    string    `json:"upstreamPath"`
	Streaming       bool      `json:"streaming" gorm:"not null"`
	StatusCode      int       `json:"statusCode" gorm:"not null"`
	DurationMs      int64     `json:"durationMs" gorm:"not null"`
	ErrorMessage    string    `json:"errorMessage"`
	ClientIP        string    `json:"clientIp"`
	UserAgent       string    `json:"userAgent"`
	RequestPayload  string    `json:"requestPayload,omitempty"`
	ResponseHeaders string    `json:"responseHeaders,omitempty"`
	ResponsePayload string    `json:"responsePayload,omitempty"`
}

func (RequestLog) TableName() string { return "request_logs" }

type RequestLogInput struct {
	Method          string
	Path            string
	Model           string
	TargetModel     string
	ProviderID      string
	ProviderName    string
	ProviderType    string
	EndpointMode    string
	UpstreamBase    string
	UpstreamPath    string
	Streaming       bool
	StatusCode      int
	DurationMs      int64
	ErrorMessage    string
	ClientIP        string
	UserAgent       string
	RequestPayload  string
	ResponseHeaders string
	ResponsePayload string
}

type Service struct {
	db *gorm.DB
}

var (
	instance *Service
	once     sync.Once
)

func GetService() *Service {
	once.Do(func() {
		instance = &Service{}
	})
	return instance
}

func (s *Service) Init() error {
	if s.db != nil {
		return nil
	}
	if err := os.MkdirAll(appdb.WorkingDir(), 0755); err != nil {
		return err
	}
	db, err := gorm.Open(sqlite.Open(appdb.DBPath()), &gorm.Config{})
	if err != nil {
		return err
	}
	if err := db.AutoMigrate(&RequestLog{}); err != nil {
		return fmt.Errorf("failed to initialize request_logs schema: %w", err)
	}
	s.db = db
	return nil
}

func (s *Service) Close() error {
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		s.db = nil
		return err
	}
	err = sqlDB.Close()
	s.db = nil
	return err
}

func (s *Service) Add(entry RequestLogInput) error {
	if err := s.Init(); err != nil {
		return err
	}
	record := RequestLog{
		Method:          entry.Method,
		Path:            entry.Path,
		Model:           entry.Model,
		TargetModel:     entry.TargetModel,
		ProviderID:      entry.ProviderID,
		ProviderName:    entry.ProviderName,
		ProviderType:    entry.ProviderType,
		EndpointMode:    entry.EndpointMode,
		UpstreamBase:    entry.UpstreamBase,
		UpstreamPath:    entry.UpstreamPath,
		Streaming:       entry.Streaming,
		StatusCode:      entry.StatusCode,
		DurationMs:      entry.DurationMs,
		ErrorMessage:    entry.ErrorMessage,
		ClientIP:        entry.ClientIP,
		UserAgent:       entry.UserAgent,
		RequestPayload:  entry.RequestPayload,
		ResponseHeaders: entry.ResponseHeaders,
		ResponsePayload: entry.ResponsePayload,
	}
	return s.db.Create(&record).Error
}

func (s *Service) List(limit int) ([]RequestLog, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	var records []RequestLog
	err := s.db.Order("created_at DESC").Limit(limit).Find(&records).Error
	return records, err
}

func (s *Service) ListJSON(limit int) string {
	records, err := s.List(limit)
	if err != nil {
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(payload)
	}
	payload, _ := json.Marshal(records)
	return string(payload)
}
