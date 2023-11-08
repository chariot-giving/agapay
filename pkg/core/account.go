package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/chariot-giving/agapay/pkg/atomic"
	"github.com/chariot-giving/agapay/pkg/bank"
	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/increase/increase-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CreateAccount creates a new account
func (s *AgapayServer) CreateAccount(ctx context.Context, key *atomic.IdempotencyKey, account *Account) (*Account, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	adb := atomic.NewAtomicDatabaseHandle(s.db, logger)
	err := adb.AtomicPhase(key, func(tx *gorm.DB) (atomic.PhaseAction, error) {
		if err := tx.Create(account).Error; err != nil {
			return nil, err
		}

		data, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		// insert an audit record
		audit := AuditRecord{
			Action:       string(AuditRecordActionCreateAccount),
			Data:         data,
			OriginIp:     ctx.Value("origin_ip").(string),
			UserId:       key.UserId,
			ResourceType: AuditRecordResourceTypeAccount,
			ResourceId:   strconv.FormatInt(*account.Id, 10),
		}
		if err := tx.Create(&audit).Error; err != nil {
			return nil, err
		}
		return atomic.RecoveryPointAction{Name: atomic.RecoveryPointAccountCreated}, nil
	})
	if err != nil {
		return nil, err
	}
	return account, nil
}

// CreateBankAccount creates a new bank account
func (s *AgapayServer) CreateBankAccount(ctx context.Context, key *atomic.IdempotencyKey, account *Account) (*Account, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	adb := atomic.NewAtomicDatabaseHandle(s.db, logger)
	err := adb.AtomicPhase(key, func(tx *gorm.DB) (atomic.PhaseAction, error) {
		if account == nil {
			// retrieve account if necessary (we're recovering from a recovery point)
			if err := tx.Where("idempotency_key_id = ?", key.Id).First(account).Error; err != nil {
				return nil, err
			}
		}

		if account == nil {
			return nil, fmt.Errorf("bug: should have an account for key %s at %s", key.Key, atomic.RecoveryPointAccountCreated)
		}

		if ctx.Value("simulate_failure") != nil {
			return nil, fmt.Errorf("simulated failure")
		}

		bankAccount, err := s.bank.CreateAccount(ctx, bank.CreateAccountRequest{
			Name:           account.Name,
			IdempotencyKey: key.Key,
		})
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return atomic.Response{Status: apierr.StatusCode, Data: cerr.NewHttpError(apierr.StatusCode, "failed to create bank account", apierr)}, nil
			}
			return atomic.Response{Status: 503, Data: cerr.NewHttpError(503, "failed to create bank account", err)}, nil
		}

		account.BankAccountId = &bankAccount.ID
		// update the account
		if err := tx.Updates(account).Error; err != nil {
			return nil, err
		}
		return atomic.RecoveryPointAction{Name: atomic.RecoveryPointBankAccountCreated}, nil
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

// CreateBankAccount creates a new bank account
func (s *AgapayServer) CreateBankAccountNumber(ctx context.Context, key *atomic.IdempotencyKey, account *Account) (*Account, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	adb := atomic.NewAtomicDatabaseHandle(s.db, logger)
	err := adb.AtomicPhase(key, func(tx *gorm.DB) (atomic.PhaseAction, error) {
		if account == nil {
			// retrieve account if necessary (we're recovering from a recovery point)
			if err := tx.Where("idempotency_key_id = ?", key.Id).First(account).Error; err != nil {
				return nil, err
			}
		}

		if account == nil {
			return nil, fmt.Errorf("bug: should have an account for key %s at %s", key.Key, atomic.RecoveryPointBankAccountCreated)
		}

		if ctx.Value("simulate_failure") != nil {
			return nil, fmt.Errorf("simulated failure")
		}

		accountNo, err := s.bank.CreateAccountNumber(ctx, bank.CreateAccountNumberRequest{
			AccountID:      *account.BankAccountId,
			Name:           account.Name,
			IdempotencyKey: key.Key,
		})
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return atomic.Response{Status: apierr.StatusCode, Data: cerr.NewHttpError(apierr.StatusCode, "failed to create bank account numbers", apierr)}, nil
			}
			return atomic.Response{Status: 503, Data: cerr.NewHttpError(503, "failed to create bank account numbers", err)}, nil
		}

		account.BankAccountNumberId = &accountNo.ID
		// update the account
		if err := tx.Updates(account).Error; err != nil {
			return nil, err
		}
		return atomic.Response{Status: 201, Data: account}, nil
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *AgapayServer) GetAccount(ctx context.Context, id string) (*Account, error) {
	account := new(Account)
	if err := s.db.Where("id = ?", id).First(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

type Account struct {
	Id                  *int64     `gorm:"primary_key;auto_increment"`
	Name                string     `gorm:"column:name"`
	BankAccountId       *string    `gorm:"column:bank_account_id"`
	BankAccountNumberId *string    `gorm:"column:bank_account_number_id"`
	IdempotencyKeyId    uint64     `gorm:"column:idempotency_key_id;index"`
	UserId              string     `gorm:"column:user_id"`
	CreatedAt           *time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (a *Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id                  *int64  `json:"id"`
		Name                string  `json:"name"`
		BankAccountId       *string `json:"bank_account_id"`
		BankAccountNumberId *string `json:"bank_account_number_id"`
		CreatedAt           string  `json:"created_at"`
	}{
		Id:                  a.Id,
		Name:                a.Name,
		BankAccountId:       a.BankAccountId,
		BankAccountNumberId: a.BankAccountNumberId,
		CreatedAt:           a.CreatedAt.Format(time.RFC3339),
	})
}
