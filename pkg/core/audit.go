package core

import "gorm.io/datatypes"

type AuditRecordResourceType string

const (
	AuditRecordResourceTypeAccount  AuditRecordResourceType = "account"
	AuditRecordResourceTypeTransfer AuditRecordResourceType = "transfer"
	AuditRecordResourceTypePayment  AuditRecordResourceType = "payment"
)

type AuditRecordAction string

const (
	AuditRecordActionCreateAccount  AuditRecordAction = "create_account"
	AuditRecordActionCreateTransfer AuditRecordAction = "create_transfer"
	AuditRecordActionCreatePayment  AuditRecordAction = "create_payment"
)

type AuditRecord struct {
	Id           *int64                  `gorm:"primary_key;auto_increment"`
	Action       string                  `gorm:"column:action"`
	Data         datatypes.JSON          `gorm:"column:data"`
	OriginIp     string                  `gorm:"column:origin_ip"`
	ResourceType AuditRecordResourceType `gorm:"column:resource_type"`
	ResourceId   string                  `gorm:"column:resource_id"`
	UserId       uint64                  `gorm:"column:user_id"`
}
