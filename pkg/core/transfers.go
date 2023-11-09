package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/chariot-giving/agapay/pkg/atomic"
	"github.com/chariot-giving/agapay/pkg/bank"
	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/increase/increase-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TransfersService struct {
	db   *gorm.DB
	bank bank.Bank
}

func (s *TransfersService) Get(ctx context.Context, id string) (*GetTransferResponse, error) {
	transfer := new(Transfer)
	if err := s.db.Where("id = ?", id).First(transfer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, cerr.NewHttpError(http.StatusNotFound, "transfer not found", err)
		}
		return nil, err
	}

	response, err := s.bank.GetTransfer(ctx, bank.GetTransferRequest{
		ID: *transfer.AchTransferId,
	})
	if err != nil {
		return nil, err
	}

	return &GetTransferResponse{
		Transfer:     transfer,
		BankTransfer: response,
	}, nil
}

func (s *TransfersService) List(ctx context.Context, req ListTransfersRequest) (*ListTransfersResponse, error) {
	transfers := make([]Transfer, 0)
	query := s.db
	if req.UserID != 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.AccountID != 0 {
		query = query.Where("account_id = ?", req.AccountID)
	}
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Cursor != "" {
		query = query.Where("id > ?", req.Cursor)
	}
	if err := query.Find(&transfers).Error; err != nil {
		return nil, err
	}

	nextCursor := ""
	if len(transfers) > 0 {
		nextCursor = strconv.FormatUint(transfers[len(transfers)-1].Id, 10)
	}

	return &ListTransfersResponse{
		Transfers:  transfers,
		NextCursor: nextCursor,
	}, nil
}

func (s *TransfersService) Create(ctx context.Context, req *atomic.IdempotentRequest, input *CreateTransferRequest) (*atomic.IdempotencyKey, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	adh := atomic.NewAtomicDatabaseHandle(s.db, logger)
	idempotencyHandler := newIdempotencyHandler(adh)
	key, err := idempotencyHandler.UpsertIdempotencyKey(ctx, req)
	if err != nil {
		return nil, err
	}

	logger.Info("idempotency key upserted", zap.String("idempotency_key", key.Key))

	transfer := new(Transfer)

	// start the idempotent state machine
	for {
		switch key.RecoveryPoint {
		case atomic.RecoveryPointStarted:
			// create the transfer
			transfer = &Transfer{
				IdempotencyKeyId: key.Id,
				UserId:           key.UserId,
				Amount:           input.Amount,
				Description:      input.Description,
				AccountId:        input.AccountID,
				AccountNumber:    input.AccountNumber,
				RoutingNumber:    input.RoutingNumber,
			}
			_, err := s.createTransfer(ctx, adh, key, transfer)
			if err != nil {
				cErr := new(cerr.HttpError)
				if errors.As(err, &cErr) {
					return nil, cErr
				}
				return nil, cerr.NewInternalServerError("failed to create transfer", err)
			}
		case atomic.RecoveryPointTransferCreated:
			// create the bank transfer
			_, err := s.createBankTransfer(ctx, adh, key, transfer)
			if err != nil {
				cErr := new(cerr.HttpError)
				if errors.As(err, &cErr) {
					return nil, cErr
				}
				return nil, cerr.NewInternalServerError("failed to create transfer", err)
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

	return key, err
}

// createTransfer creates a new transfer
func (s *TransfersService) createTransfer(ctx context.Context, handle *atomic.AtomicDatabaseHandle, key *atomic.IdempotencyKey, transfer *Transfer) (*Transfer, error) {
	err := handle.AtomicPhase(key, func(tx *gorm.DB) (atomic.PhaseAction, error) {
		if err := tx.Create(transfer).Error; err != nil {
			return nil, err
		}

		data, err := json.Marshal(transfer)
		if err != nil {
			return nil, err
		}

		// insert an audit record
		audit := AuditRecord{
			Action:       string(AuditRecordActionCreateTransfer),
			Data:         data,
			OriginIp:     ctx.Value("origin_ip").(string),
			UserId:       key.UserId,
			ResourceType: AuditRecordResourceTypeTransfer,
			ResourceId:   strconv.FormatUint(transfer.Id, 10),
		}
		if err := tx.Create(&audit).Error; err != nil {
			return nil, err
		}
		return atomic.RecoveryPointAction{Name: atomic.RecoveryPointTransferCreated}, nil
	})
	if err != nil {
		return nil, err
	}
	return transfer, nil
}

// createBankTransfer creates a new bank transfer
func (s *TransfersService) createBankTransfer(ctx context.Context, handle *atomic.AtomicDatabaseHandle, key *atomic.IdempotencyKey, transfer *Transfer) (*Transfer, error) {
	err := handle.AtomicPhase(key, func(tx *gorm.DB) (atomic.PhaseAction, error) {
		if transfer == nil {
			// retrieve transfer if necessary (we're recovering from a recovery point)
			if err := tx.Where("idempotency_key_id = ?", key.Id).First(transfer).Error; err != nil {
				return nil, err
			}
		}

		if transfer == nil {
			return nil, fmt.Errorf("bug: should have an transfer for key %s at %s", key.Key, atomic.RecoveryPointTransferCreated)
		}

		if ctx.Value("simulate_failure") != nil {
			return nil, fmt.Errorf("simulated failure")
		}

		account := new(Account)
		if err := s.db.Where("id = ?", transfer.AccountId).First(account).Error; err != nil {
			return nil, err
		}

		response, err := s.bank.TransferFunds(ctx, bank.TransferFundsRequest{
			AccountID:      *account.BankAccountId,
			Amount:         transfer.Amount,
			Description:    transfer.Description,
			AccountNumber:  transfer.AccountNumber,
			RoutingNumber:  transfer.RoutingNumber,
			IdempotencyKey: key.Key,
		})
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return atomic.Response{Status: apierr.StatusCode, Data: cerr.NewHttpError(apierr.StatusCode, "failed to create bank transfer", apierr)}, nil
			}
			return atomic.Response{Status: 503, Data: cerr.NewHttpError(503, "failed to create bank transfer", err)}, nil
		}

		transfer.AchTransferId = &response.ID

		// update the transfer
		if err := tx.Updates(transfer).Error; err != nil {
			return nil, err
		}
		return atomic.Response{Status: 201, Data: transfer}, nil
	})
	if err != nil {
		return nil, err
	}

	return transfer, nil
}

func newTransfersService(db *gorm.DB, bank bank.Bank) *TransfersService {
	return &TransfersService{
		db:   db,
		bank: bank,
	}
}

type CreateTransferRequest struct {
	AccountID     uint64
	Description   string
	Amount        int64
	AccountNumber string
	RoutingNumber string
	Funding       string
}

type ListTransfersRequest struct {
	UserID    uint64
	AccountID uint64
	Limit     int
	Cursor    string
}

type ListTransfersResponse struct {
	Transfers  []Transfer
	NextCursor string
}

type GetTransferResponse struct {
	Transfer     *Transfer
	BankTransfer *bank.GetTransferResponse
}

type Transfer struct {
	Id               uint64 `gorm:"primaryKey;autoIncrement"`
	Amount           int64
	Description      string
	AccountNumber    string
	RoutingNumber    string
	AchTransferId    *string
	AccountId        uint64   `gorm:"column:account_id;index"`
	Account          *Account `gorm:"foreignKey:AccountId"`
	IdempotencyKeyId uint64   `gorm:"column:idempotency_key_id;index"`
	UserId           uint64   `gorm:"column:user_id"`
	CreatedAt        *time.Time
}

func (t *Transfer) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id            uint64  `json:"id"`
		Amount        int64   `json:"amount"`
		Description   string  `json:"description"`
		AchTransferId *string `json:"ach_transfer_id"`
		AccountId     uint64  `json:"account_id"`
		CreatedAt     string  `json:"created_at"`
	}{
		Id:            t.Id,
		Amount:        t.Amount,
		Description:   t.Description,
		AchTransferId: t.AchTransferId,
		AccountId:     t.AccountId,
		CreatedAt:     t.CreatedAt.Format(time.RFC3339),
	})
}
