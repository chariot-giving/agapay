package adb

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AgapayDB struct {
	*gorm.DB
	logger *zap.Logger
}

type IdempotentRequest struct {
	UserId         string
	IdempotencyKey string
	Method         string
	Path           string
	Params         any
}

type ArgumentErr struct {
	msg string
}

func (e ArgumentErr) Error() string {
	return fmt.Sprintf("invalid argument: %s", e.msg)
}

type IdempotencyKeyConflictErr struct {
	msg string
}

func (e IdempotencyKeyConflictErr) Error() string {
	return fmt.Sprintf("idempotency key conflict: %s", e.msg)
}
