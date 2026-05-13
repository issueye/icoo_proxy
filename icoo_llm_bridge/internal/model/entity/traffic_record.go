package entity

import "time"

type TrafficRecord struct {
	ID                   string `gorm:"primaryKey"`
	RequestID            string `gorm:"uniqueIndex;not null"`
	Endpoint             string
	Method               string
	ClientIP             string
	UserAgent            string
	ContentType          string
	DownstreamProtocol   string
	UpstreamProtocol     string
	RequestedModel       string
	Model                string
	RequestBody          string
	RequestBodyBytes     int64
	RequestBodyTruncated bool
	StatusCode           int
	DurationMS           int64
	InputTokens          int
	OutputTokens         int
	TotalTokens          int
	Error                string
	CreatedAt            time.Time `gorm:"index"`
}

func (TrafficRecord) TableName() string {
	return "traffic_records"
}
