package entity

import (
	"time"

	"github.com/issueye/icoo_proxy/common/constants"
)

// IngressEndpoint 入口端点
type IngressEndpoint struct {
	ID                 string             `gorm:"primaryKey;column:id;comment:入口端点ID"`
	Path               string             `gorm:"uniqueIndex;not null;comment:路径模式"`
	DownstreamProtocol constants.Protocol `gorm:"column:downstream_protocol;comment:下游协议"`
	Enabled            bool               `gorm:"column:enabled;comment:是否启用"`
	Protected          bool               `gorm:"column:protected;comment:是否受保护"`
	BuiltIn            bool               `gorm:"column:built_in;comment:是否内建"`
	Description        string             `gorm:"column:description;comment:描述"`
	CreatedAt          time.Time          `gorm:"column:created_at;comment:创建时间"`
	UpdatedAt          time.Time          `gorm:"column:updated_at;comment:更新时间"`
}

func (IngressEndpoint) TableName() string {
	return "ingress_endpoints"
}
