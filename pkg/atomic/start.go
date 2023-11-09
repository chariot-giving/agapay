package atomic

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/chariot-giving/agapay/pkg/cerr"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// DefaultLockDuration is the default duration for which an idempotency key is locked
	DefaultLockDuration = 5 * time.Minute
)

// UpsertIdempotencyKey creates a new idempotency key if one does not already exist
// This should always be the first call whenever an idempotent request is received.
func (db *AtomicDatabaseHandle) UpsertIdempotencyKey(request *IdempotentRequest) (*IdempotencyKey, error) {
	key := new(IdempotencyKey)
	err := db.AtomicPhase(key, func(tx *gorm.DB) (PhaseAction, error) {
		err := tx.Where("user_id = ? AND key = ?", request.UserId, request.IdempotencyKey).First(key).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				params, err := json.Marshal(request.Params)
				if err != nil {
					return nil, err
				}
				body, err := json.Marshal(request.Body)
				if err != nil {
					return nil, err
				}
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
					RequestParams: params,
					RequestBody:   body,
					ResponseBody:  body,
				}
				if err := tx.Create(key).Error; err != nil {
					return nil, err
				}
				return Noop{}, nil
			}
			return nil, err
		}

		// programs sending multiple requests with diff parameters but the same idempotency key is a bug
		keyParams := make(map[string]string)
		if err := json.Unmarshal(key.RequestParams, &keyParams); err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(keyParams, request.Params) {
			return nil, cerr.NewHttpError(http.StatusUnprocessableEntity, "request parameters do not match", nil)
		}

		requestType := reflect.TypeOf(request.Body)
		if requestType.Kind() == reflect.Ptr {
			requestType = requestType.Elem()
		}
		keyBody := reflect.New(requestType).Interface()
		err = json.Unmarshal(key.RequestBody, keyBody)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(keyBody, request.Body) {
			db.logger.Error("request body does not match idempotent request", zap.Any("request_body", request.Body), zap.Any("idempotent_request_body", keyBody))
			return nil, cerr.NewHttpError(http.StatusUnprocessableEntity, "request body does not match", nil)
		}

		// only acquire a lock if the key is unlocked or it's lock has expired
		if key.LockedAt != nil && key.LockedAt.Add(DefaultLockDuration).Before(time.Now()) {
			return nil, cerr.NewConflictError("request is already in progress", nil)
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
