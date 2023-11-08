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

type AccountsService struct {
	db   *gorm.DB
	bank bank.Bank
}

func newAccountsService(db *gorm.DB, bank bank.Bank) *AccountsService {
	return &AccountsService{
		db:   db,
		bank: bank,
	}
}

func (s *AccountsService) Create(ctx context.Context, req *atomic.IdempotentRequest, input *CreateAccountRequest) (*atomic.IdempotencyKey, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	key, err := newIdempotencyHandler(atomic.NewAtomicDatabaseHandle(s.db, logger)).UpsertIdempotencyKey(ctx, req)
	if err != nil {
		return nil, err
	}

	logger.Info("idempotency key upserted", zap.String("idempotency_key", key.Key))

	account := new(Account)

	// start the idempotent state machine
	for {
		switch key.RecoveryPoint {
		case atomic.RecoveryPointStarted:
			// create the account
			account = &Account{
				IdempotencyKeyId: key.Id,
				UserId:           key.UserId,
				Name:             input.Name,
			}
			_, err := s.createAccount(ctx, key, account)
			if err != nil {
				cErr := new(cerr.HttpError)
				if errors.As(err, &cErr) {
					return nil, cErr
				}
				return nil, cerr.NewInternalServerError("failed to create account", err)
			}
		case atomic.RecoveryPointAccountCreated:
			// create the bank account
			_, err := s.createBankAccount(ctx, key, account)
			if err != nil {
				cErr := new(cerr.HttpError)
				if errors.As(err, &cErr) {
					return nil, cErr
				}
				return nil, cerr.NewInternalServerError("failed to create bank account", err)
			}
		case atomic.RecoveryPointBankAccountCreated:
			// create the bank account number
			_, err := s.createBankAccountNumber(ctx, key, account)
			if err != nil {
				cErr := new(cerr.HttpError)
				if errors.As(err, &cErr) {
					return nil, cErr
				}
				return nil, cerr.NewInternalServerError("failed to create bank account number", err)
			}
		case atomic.RecoveryPointFinished:
			// we're done
		default:
			return nil, cerr.NewInternalServerError("bug: unknown recovery point", nil)
		}

		// if we're done, break out of the loop
		if key.RecoveryPoint == atomic.RecoveryPointFinished {
			break
		}
	}

	return key, nil
}

type CreateAccountRequest struct {
	Name string
}

// CreateAccount creates a new account
func (s *AccountsService) createAccount(ctx context.Context, key *atomic.IdempotencyKey, account *Account) (*Account, error) {
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
			ResourceId:   strconv.FormatInt(account.Id, 10),
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
func (s *AccountsService) createBankAccount(ctx context.Context, key *atomic.IdempotencyKey, account *Account) (*Account, error) {
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
func (s *AccountsService) createBankAccountNumber(ctx context.Context, key *atomic.IdempotencyKey, account *Account) (*Account, error) {
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

func (s *AccountsService) Get(ctx context.Context, id string) (*Account, error) {
	account := new(Account)
	if err := s.db.Where("id = ?", id).First(account).Error; err != nil {
		return nil, err
	}
	return account, nil
}

func (s *AccountsService) List(ctx context.Context, req ListAccountsRequest) (*ListAccountsResponse, error) {
	accounts := make([]Account, 0)
	query := s.db
	if req.UserID != 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Cursor != "" {
		query = query.Where("id > ?", req.Cursor)
	}
	if err := query.Find(&accounts).Error; err != nil {
		return nil, err
	}

	nextCursor := ""
	if len(accounts) > 0 {
		nextCursor = strconv.FormatInt(accounts[len(accounts)-1].Id, 10)
	}

	return &ListAccountsResponse{
		Accounts:   accounts,
		NextCursor: nextCursor,
	}, nil
}

type ListAccountsRequest struct {
	UserID uint64
	Limit  int
	Cursor string
}

type ListAccountsResponse struct {
	Accounts   []Account
	NextCursor string
}

func (s *AccountsService) GetDetails(ctx context.Context, id string) (*AccountDetails, error) {
	details, err := s.bank.GetAccountDetails(ctx, bank.GetAccountDetailsRequest{
		AccountID: id,
	})
	if err != nil {
		return nil, err
	}

	return &AccountDetails{
		Status:  string(details.Status),
		Numbers: details.Numbers,
	}, nil
}

func (s *AccountsService) GetBalance(ctx context.Context, id string) (*AccountBalance, error) {
	balanceLookup, err := s.bank.GetAccountBalance(ctx, bank.GetAccountBalanceRequest{
		AccountID: id,
	})
	if err != nil {
		return nil, err
	}

	return &AccountBalance{
		CurrentBalance:   balanceLookup.CurrentBalance,
		AvailableBalance: balanceLookup.AvailableBalance,
	}, nil
}

type AccountBalance struct {
	CurrentBalance   int64
	AvailableBalance int64
}

type AccountDetails struct {
	Status  string
	Numbers []bank.AccountNumber
}

type Account struct {
	Id                  int64      `gorm:"primary_key;auto_increment"`
	Name                string     `gorm:"column:name"`
	BankAccountId       *string    `gorm:"column:bank_account_id"`
	BankAccountNumberId *string    `gorm:"column:bank_account_number_id"`
	IdempotencyKeyId    uint64     `gorm:"column:idempotency_key_id;index"`
	UserId              uint64     `gorm:"column:user_id"`
	CreatedAt           *time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (a *Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id                  int64   `json:"id"`
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
