package atomic

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AtomicDatabaseHandle struct {
	*gorm.DB
	logger *zap.Logger
}

func NewAtomicDatabaseHandle(db *gorm.DB, logger *zap.Logger) *AtomicDatabaseHandle {
	return &AtomicDatabaseHandle{
		DB:     db,
		logger: logger,
	}
}

type IdempotentRequest struct {
	UserId         uint64
	IdempotencyKey string
	Method         string
	Path           string
	Params         map[string]string
	Body           any
}
