package adb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/chariot-giving/agapay/pkg/bank"
	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/increase/increase-go"
	"github.com/increase/increase-go/option"
	"gorm.io/gorm"
)

// CreateAccount creates a new account
func (db *AgapayDB) CreateAccount(ctx context.Context, key *IdempotencyKey, account *Account) (*Account, error) {
	err := db.AtomicPhase(key, func(tx *gorm.DB) (PhaseAction, error) {
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
			ResourceType: "account",
			ResourceId:   strconv.FormatInt(*account.Id, 10),
		}
		if err := tx.Create(&audit).Error; err != nil {
			return nil, err
		}
		return RecoveryPointAction{RecoveryPointAccountCreated}, nil
	})
	if err != nil {
		return nil, err
	}
	return account, nil
}

// CreateBankAccount creates a new bank account
func (db *AgapayDB) CreateBankAccount(ctx context.Context, key *IdempotencyKey, account *Account) (*Account, error) {
	err := db.AtomicPhase(key, func(tx *gorm.DB) (PhaseAction, error) {
		if account == nil {
			// retrieve account if necessary (we're recovering from a recovery point)
			if err := tx.Where("idempotency_key_id = ?", key.Id).First(account).Error; err != nil {
				return nil, err
			}
		}

		if account == nil {
			return nil, fmt.Errorf("bug: should have an account for key %s at %s", key.Key, RecoveryPointAccountCreated)
		}

		if ctx.Value("simulate_failure") != nil {
			return nil, fmt.Errorf("simulated failure")
		}

		bankIdempotencyKey := fmt.Sprintf("bank-account-%d", key.Id)
		bankAccount, err := bank.IncreaseClient.Accounts.New(ctx, increase.AccountNewParams{
			Name: increase.String(account.Name),
		}, option.WithHeader("X-Idempotency-Key", bankIdempotencyKey))
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return Response{Status: apierr.StatusCode, Data: cerr.NewHttpError(apierr.StatusCode, "failed to create bank account", apierr)}, nil
			}
			return Response{Status: 503, Data: cerr.NewHttpError(503, "failed to create bank account", err)}, nil
		}

		account.BankAccountId = &bankAccount.ID
		// update the account
		if err := tx.Updates(account).Error; err != nil {
			return nil, err
		}
		return RecoveryPointAction{RecoveryPointBankAccountCreated}, nil
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

// CreateBankAccount creates a new bank account
func (db *AgapayDB) CreateBankAccountNumber(ctx context.Context, key *IdempotencyKey, account *Account) (*Account, error) {
	err := db.AtomicPhase(key, func(tx *gorm.DB) (PhaseAction, error) {
		if account == nil {
			// retrieve account if necessary (we're recovering from a recovery point)
			if err := tx.Where("idempotency_key_id = ?", key.Id).First(account).Error; err != nil {
				return nil, err
			}
		}

		if account == nil {
			return nil, fmt.Errorf("bug: should have an account for key %s at %s", key.Key, RecoveryPointBankAccountCreated)
		}

		if ctx.Value("simulate_failure") != nil {
			return nil, fmt.Errorf("simulated failure")
		}

		bankIdempotencyKey := fmt.Sprintf("bank-account-number-%d", key.Id)
		accountNo, err := bank.IncreaseClient.AccountNumbers.New(ctx, increase.AccountNumberNewParams{
			AccountID: increase.String(*account.BankAccountId),
			Name:      increase.String(account.Name),
			InboundACH: increase.F[increase.AccountNumberNewParamsInboundACH](increase.AccountNumberNewParamsInboundACH{
				DebitStatus: increase.F[increase.AccountNumberNewParamsInboundACHDebitStatus](increase.AccountNumberNewParamsInboundACHDebitStatusBlocked),
			}),
		}, option.WithHeader("X-Idempotency-Key", bankIdempotencyKey))
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return Response{Status: apierr.StatusCode, Data: cerr.NewHttpError(apierr.StatusCode, "failed to create bank account numbers", apierr)}, nil
			}
			return Response{Status: 503, Data: cerr.NewHttpError(503, "failed to create bank account numbers", err)}, nil
		}

		account.BankAccountNumberId = &accountNo.ID
		// update the account
		if err := tx.Updates(account).Error; err != nil {
			return nil, err
		}
		return Response{Status: 201, Data: account}, nil
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (db *AgapayDB) GetAccount(ctx context.Context, id string) (*Account, error) {
	account := &Account{}
	if err := db.DB.Where("id = ?", id).First(account).Error; err != nil {
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
