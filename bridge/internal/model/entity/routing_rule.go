package entity

import (
	"time"

	"github.com/issueye/icoo_proxy/common/constants"
)

type RoutingRule struct {
	ID                string             `gorm:"primaryKey"` // 路由规则ID
	Name              string             `gorm:"not null"`   // 路由规则名称
	Priority          int                `gorm:"index"`      // 路由规则优先级
	MatchProtocol     constants.Protocol // 匹配协议
	MatchModelPattern string             // 匹配模型模式
	UpstreamProtocol  constants.Protocol // 上游协议
	TargetProviderID  string             `gorm:"index"` // 目标供应商ID
	TargetModel       string             // 目标模型ID
	Enabled           bool               // 是否启用路由规则
	CreatedAt         time.Time          // 创建时间
	UpdatedAt         time.Time          // 更新时间
}

func (RoutingRule) TableName() string {
	return "routing_rules"
}
