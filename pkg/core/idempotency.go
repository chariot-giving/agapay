package core

import (
	"context"

	"github.com/chariot-giving/agapay/pkg/atomic"
	"go.uber.org/zap"
)

func (s *AgapayServer) UpsertIdempotencyKey(ctx context.Context, request *atomic.IdempotentRequest) (*atomic.IdempotencyKey, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	adb := atomic.NewAtomicDatabaseHandle(s.db, logger)
	return adb.UpsertIdempotencyKey(request)
}
