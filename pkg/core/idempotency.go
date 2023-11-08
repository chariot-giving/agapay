package core

import (
	"context"

	"github.com/chariot-giving/agapay/pkg/atomic"
)

type idempotencyHandler struct {
	adb *atomic.AtomicDatabaseHandle
}

func newIdempotencyHandler(adb *atomic.AtomicDatabaseHandle) *idempotencyHandler {
	return &idempotencyHandler{
		adb: adb,
	}
}

func (h *idempotencyHandler) UpsertIdempotencyKey(ctx context.Context, request *atomic.IdempotentRequest) (*atomic.IdempotencyKey, error) {
	return h.adb.UpsertIdempotencyKey(request)
}
