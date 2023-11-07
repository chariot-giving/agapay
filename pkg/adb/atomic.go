package adb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AtomicPhase is a function that executes a function in a long-running transaction
// Please note that this function should only be used where transactions and idempotence are absolutely necessary
// which is most likely the case whenever we need to make a foreign state mutation (e.g. a payment).
// An atomic phase is a set of local state mutations that occur in transactions between foreign state mutations.
// So if you're not mutating foreign state, you probably don't need this.
func (db *AgapayDB) AtomicPhase(key *IdempotencyKey, fn TransactionFunc) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		result, err := fn(tx)
		if err != nil {
			return err
		}
		phaseKey := PhaseKey{key: key, tx: tx}
		return result.Exec(phaseKey)
	}, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		// handle serialization error (code 40001 from Postgres)
		// retry transaction if it's a serialization error
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "40001" {
			db.logger.Warn("retrying transaction due to serialization error", zap.Error(err))
			return db.AtomicPhase(key, fn)
		}

		// if we're leaving under an error condition, try to unlock the idempotency key
		// so that another request can try again.
		if key != nil {
			tx := db.Model(key).Update("locked_at", nil)
			if tx.Error != nil {
				// We're already inside an error condition so swallow additional errors
				// and just log them.
				db.logger.Error("failed to unlock idempotency key", zap.Error(tx.Error))
			}
		}
		return err
	}
	return nil
}

// TransactionFunc is a function executed within an atomic phase
type TransactionFunc func(tx *gorm.DB) (PhaseAction, error)

type PhaseKey struct {
	key *IdempotencyKey
	tx  *gorm.DB
}

// PhaseAction is the result of a function executed within an atomic phase
type PhaseAction interface {
	Exec(key PhaseKey) error
}

// Noop indicates that program flow should continue,
// but that neither a recovery point nor response should be set.
type Noop struct{}

func (r Noop) Exec(key PhaseKey) error {
	return nil
}

// Represents an action to set a new recovery point. One possible option for a
// return from an AtomicPhase transaction function.
type RecoveryPointAction struct {
	Name RecoveryPoint
}

func (r RecoveryPointAction) Exec(key PhaseKey) error {
	if key.key == nil {
		return cerr.NewBadRequest("idempotency key must be provided to use a recovery point", nil)
	}
	return key.tx.Model(key.key).Update("recovery_point", string(r.Name)).Error
}

// Represents an action to set a new API response (which will be stored onto an
// idempotency key). One  possible option for a return from an #atomic_phase block.
type Response struct {
	Status int
	Data   json.Marshaler
}

func (r Response) Exec(key PhaseKey) error {
	if key.key == nil {
		return cerr.NewBadRequest("idempotency key must be provided to record a response", nil)
	}
	if r.Status == 0 {
		return cerr.NewInternalServerError("response status must be provided", nil)
	}

	body, err := r.Data.MarshalJSON()
	if err != nil {
		return err
	}

	return key.tx.Model(key.key).Updates(IdempotencyKey{
		LockedAt:      nil,
		RecoveryPoint: RecoveryPointFinished,
		ResponseCode:  r.Status,
		ResponseBody:  body,
	}).Error
}

type IdempotencyKey struct {
	Id            uint64         `gorm:"primaryKey;autoIncrement"`
	Key           string         `gorm:"column:key;uniqueIndex"`
	UserId        uint64         `gorm:"column:user_id"`
	LastRunAt     time.Time      `gorm:"column:last_run_at"`
	LockedAt      *time.Time     `gorm:"column:locked_at"`
	RecoveryPoint RecoveryPoint  `gorm:"column:recovery_point"`
	RequestMethod string         `gorm:"column:request_method"`
	RequestPath   string         `gorm:"column:request_path"`
	RequestParams datatypes.JSON `gorm:"column:request_params"`
	RequestBody   datatypes.JSON `gorm:"column:request_body"`
	ResponseCode  int            `gorm:"column:response_code"`
	ResponseBody  datatypes.JSON `gorm:"column:response_body"`
}
