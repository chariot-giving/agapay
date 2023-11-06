package adb

import "gorm.io/datatypes"

type AuditRecordAction string

const (
	AuditRecordActionCreateAccount AuditRecordAction = "create_account"
)

type AuditRecord struct {
	Id           *int64         `gorm:"primary_key;auto_increment"`
	Action       string         `gorm:"column:action"`
	Data         datatypes.JSON `gorm:"column:data"`
	OriginIp     string         `gorm:"column:origin_ip"`
	ResourceType string         `gorm:"column:resource_type"`
	ResourceId   string         `gorm:"column:resource_id"`
	UserId       string         `gorm:"column:user_id"`
}
