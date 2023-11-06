package adb

import (
	"time"

	"gorm.io/gorm"
)

const (
	// DefaultLockDuration is the default duration for which an idempotency key is locked
	DefaultLockDuration = 5 * time.Minute
)

// UpsertIdempotencyKey creates a new idempotency key if one does not already exist
// This should always be the first call whenever an idempotent request is received.
func (db *AgapayDB) UpsertIdempotencyKey(request *IdempotentRequest) (*IdempotencyKey, error) {
	key := new(IdempotencyKey)
	err := db.AtomicPhase(nil, func(tx *gorm.DB) (PhaseAction, error) {
		err := tx.Where("user_id = ? AND idempotency_key = ?", request.UserId, request.IdempotencyKey).First(key).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				now := time.Now()
				// create a new key
				key = &IdempotencyKey{
					UserId:        request.UserId,
					Key:           request.IdempotencyKey,
					LockedAt:      &now,
					LastRunAt:     now,
					RecoveryPoint: RecoveryPointStarted,
					RequestMethod: request.Method,
					RequestPath:   request.Path,
					RequestParams: request.Params,
				}
				if err := tx.Create(key).Error; err != nil {
					return nil, err
				}
				return Noop{}, nil
			} else {
				return nil, err
			}
		}

		// programs sending multiple requestw with diff parameters but the same idempotency key is a bug
		if key.RequestParams != request.Params {
			return nil, IdempotencyKeyConflictErr{"request parameters do not match"}
		}

		// only acquire a lock if the key is unlocked or it's lock has expired
		if key.LockedAt == nil || key.LockedAt.Add(DefaultLockDuration).Before(time.Now()) {
			return nil, IdempotencyKeyConflictErr{"request is already in progress"}
		}

		// lock the key and update latest run time if request is not already finished
		if key.RecoveryPoint != RecoveryPointFinished {
			now := time.Now()
			key.LockedAt = &now
			key.LastRunAt = now
			if err := tx.Updates(key).Error; err != nil {
				return nil, err
			}
		}

		// no response and no need to set a recovery point
		return Noop{}, nil
	})
	if err != nil {
		return nil, err
	}
	return key, nil
}
