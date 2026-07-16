package entity

import "time"

// TrafficRecord 流量记录
type TrafficRecord struct {
	ID                   string    `gorm:"primaryKey;column:id;comment:流量记录ID"`
	RequestID            string    `gorm:"uniqueIndex;not null;comment:请求ID"`
	Endpoint             string    `gorm:"column:endpoint;comment:入口端点"`
	Method               string    `gorm:"column:method;comment:请求方法"`
	ClientIP             string    `gorm:"column:client_ip;comment:客户端IP"`
	UserAgent            string    `gorm:"column:user_agent;comment:用户代理"`
	ContentType          string    `gorm:"column:content_type;comment:内容类型"`
	UpstreamProtocol     string    `gorm:"column:upstream_protocol;comment:上游协议"`
	DownstreamProtocol   string    `gorm:"column:downstream_protocol;comment:下游协议"`
	RouteName            string    `gorm:"column:route_name;comment:路由名称"`
	RouteSource          string    `gorm:"column:route_source;comment:路由来源"`
	MatchedRuleID        string    `gorm:"column:matched_rule_id;comment:匹配规则ID"`
	MatchedRuleName      string    `gorm:"column:matched_rule_name;comment:匹配规则名称"`
	RequestedModel       string    `gorm:"column:request_model;comment:请求模型"`
	Model                string    `gorm:"column:model;comment:模型"`
	RequestBody          string    `gorm:"column:request_body;comment:请求体"`
	RequestBodyBytes     int64     `gorm:"column:request_body_bytes;comment:请求体体"`
	RequestBodyTruncated bool      `gorm:"column:request_body_truncated;comment:是否截断请求体"`
	StatusCode           int       `gorm:"column:status_code;comment:状态码"`
	DurationMS           int64     `gorm:"column:duration_ms;comment:持续时间（毫秒）"`
	InputTokens          int       `gorm:"column:input_tokens;comment:输入令牌数"`
	OutputTokens         int       `gorm:"column:output_tokens;comment:输出令牌数"`
	TotalTokens          int       `gorm:"column:total_tokens;comment:总令牌数"`
	Error                string    `gorm:"column:error;comment:错误信息"`
	CreatedAt            time.Time `gorm:"column:created_at;comment:创建时间"`
}

func (TrafficRecord) TableName() string {
	return "traffic_records"
}
