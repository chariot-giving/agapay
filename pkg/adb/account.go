package adb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/chariot-giving/agapay/pkg/bank"
	"github.com/gin-gonic/gin"
	"github.com/increase/increase-go"
	"github.com/increase/increase-go/option"
	"gorm.io/gorm"
)

func (db *AgapayDB) CreateAccountDag(c *gin.Context) (*Account, error) {
	request := &IdempotentRequest{
		UserId:         c.GetString("user_id"),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Method:         c.Request.Method,
		Path:           c.Request.URL.Path,
		Params:         nil, // TODO: pass through params
	}
	key, err := db.UpsertIdempotencyKey(request)
	if err != nil {
		return nil, err
	}

	var account Account

	// start the loop
	for {
		switch key.RecoveryPoint {
		case RecoveryPointStarted:
			// create the account
			createdAccount, err := db.CreateAccount(c, key, request.Params)
			if err != nil {
				return nil, err
			}
			account = *createdAccount
			key.RecoveryPoint = RecoveryPointAccountCreated
		case RecoveryPointAccountCreated:
			// create the bank account
			createdAccount, err := db.CreateBankAccount(c, key, &account)
			if err != nil {
				return nil, err
			}
			account = *createdAccount
			key.RecoveryPoint = RecoveryPointBankAccountCreated
		case RecoveryPointBankAccountCreated:
			// create the bank account number
			createdAccount, err := db.CreateBankAccountNumber(c, key, &account)
			if err != nil {
				return nil, err
			}
			account = *createdAccount
			key.RecoveryPoint = RecoveryPointFinished
		case RecoveryPointFinished:
			// we're done
		default:
			return nil, fmt.Errorf("bug: unknown recovery point %s", key.RecoveryPoint)
		}

		// if we're done, break out of the loop
		if key.RecoveryPoint == RecoveryPointFinished {
			break
		}
	}

	return &account, nil
}

// CreateAccount creates a new account
func (db *AgapayDB) CreateAccount(ctx context.Context, key *IdempotencyKey, params any) (*Account, error) {
	account := new(Account)
	err := db.AtomicPhase(key, func(tx *gorm.DB) (PhaseAction, error) {
		account = &Account{
			Name:                "", // TODO: pass through in params
			BankAccountId:       nil,
			BankAccountNumberId: nil,
			IdempotencyKeyId:    key.Id,
			UserId:              key.UserId,
		}
		if err := tx.Create(account).Error; err != nil {
			return nil, err
		}

		data, err := json.Marshal(params)
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
			return Response{Status: 503, Data: err}, nil
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
			return Response{Status: 503, Data: err}, nil
		}

		account.BankAccountNumberId = &accountNo.ID
		// update the account
		if err := tx.Updates(account).Error; err != nil {
			return nil, err
		}
		return Response{Status: 201, Data: nil}, nil
	})
	if err != nil {
		return nil, err
	}

	return account, nil
}

type Account struct {
	Id                  *int64  `gorm:"primary_key;auto_increment"`
	Name                string  `gorm:"column:name"`
	BankAccountId       *string `gorm:"column:bank_account_id"`
	BankAccountNumberId *string `gorm:"column:bank_account_number_id"`
	IdempotencyKeyId    int64   `gorm:"column:idempotency_key_id;index"`
	UserId              string  `gorm:"column:user_id"`
}
