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

type PaymentsService struct {
	db   *gorm.DB
	bank bank.Bank
}

func (s *PaymentsService) Get(ctx context.Context, id string) (*GetPaymentResponse, error) {
	payment := new(Payment)
	if err := s.db.Where("id = ?", id).First(payment).Error; err != nil {
		return nil, err
	}

	if payment.BankTransferId == nil {
		return &GetPaymentResponse{
			Payment: payment,
		}, nil
	}

	bankPayment, err := s.bank.GetPayment(ctx, bank.GetPaymentRequest{
		ID:          *payment.BankTransferId,
		PaymentRail: bank.PaymentRail(payment.PaymentRail),
	})
	if err != nil {
		return nil, err
	}

	return &GetPaymentResponse{
		Payment:     payment,
		BankPayment: bankPayment,
	}, nil
}

func (s *PaymentsService) List(ctx context.Context, req ListPaymentsRequest) (*ListPaymentsResponse, error) {
	payments := make([]Payment, 0)
	query := s.db
	if req.UserID != 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.AccountID != 0 {
		query = query.Where("account_id = ?", req.AccountID)
	}
	if req.AccountID != 0 {
		query = query.Where("recipient_id = ?", req.AccountID)
	}
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Cursor != "" {
		query = query.Where("id > ?", req.Cursor)
	}
	if err := query.Find(&payments).Error; err != nil {
		return nil, err
	}

	nextCursor := ""
	if len(payments) > 0 {
		nextCursor = strconv.FormatUint(payments[len(payments)-1].Id, 10)
	}

	return &ListPaymentsResponse{
		Payments:   payments,
		NextCursor: nextCursor,
	}, nil
}

func (s *PaymentsService) Create(ctx context.Context, req *atomic.IdempotentRequest, input *CreatePaymentRequest) (*atomic.IdempotencyKey, error) {
	logger := ctx.Value("logger").(*zap.Logger)
	adh := atomic.NewAtomicDatabaseHandle(s.db, logger)
	idempotencyHandler := newIdempotencyHandler(adh)
	key, err := idempotencyHandler.UpsertIdempotencyKey(ctx, req)
	if err != nil {
		return nil, err
	}

	logger.Info("idempotency key upserted", zap.String("idempotency_key", key.Key))

	payment := new(Payment)

	// start the idempotent state machine
	for {
		switch key.RecoveryPoint {
		case atomic.RecoveryPointStarted:
			// create the transfer
			payment = &Payment{
				IdempotencyKeyId: key.Id,
				UserId:           key.UserId,
				Amount:           input.Amount,
				Description:      input.Description,
				AccountId:        input.AccountID,
				RecipientId:      input.RecipientID,
			}
			_, err := s.createPayment(ctx, adh, key, payment)
			if err != nil {
				cErr := new(cerr.HttpError)
				if errors.As(err, &cErr) {
					return nil, cErr
				}
				return nil, cerr.NewInternalServerError("failed to create payment", err)
			}
		case atomic.RecoveryPointPaymentCreated:
			// create the bank transfer
			_, err := s.createBankPayment(ctx, adh, key, payment)
			if err != nil {
				cErr := new(cerr.HttpError)
				if errors.As(err, &cErr) {
					return nil, cErr
				}
				return nil, cerr.NewInternalServerError("failed to create payment", err)
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

// createPayment creates a new payment
func (s *PaymentsService) createPayment(ctx context.Context, handle *atomic.AtomicDatabaseHandle, key *atomic.IdempotencyKey, payment *Payment) (*Payment, error) {
	err := handle.AtomicPhase(key, func(tx *gorm.DB) (atomic.PhaseAction, error) {
		if err := tx.Create(payment).Error; err != nil {
			return nil, err
		}

		data, err := json.Marshal(payment)
		if err != nil {
			return nil, err
		}

		// insert an audit record
		audit := AuditRecord{
			Action:       string(AuditRecordActionCreatePayment),
			Data:         data,
			OriginIp:     ctx.Value("origin_ip").(string),
			UserId:       key.UserId,
			ResourceType: AuditRecordResourceTypePayment,
			ResourceId:   strconv.FormatUint(payment.Id, 10),
		}
		if err := tx.Create(&audit).Error; err != nil {
			return nil, err
		}
		return atomic.RecoveryPointAction{Name: atomic.RecoveryPointPaymentCreated}, nil
	})
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// createBankPayment creates a new bank payment
func (s *PaymentsService) createBankPayment(ctx context.Context, handle *atomic.AtomicDatabaseHandle, key *atomic.IdempotencyKey, payment *Payment) (*Payment, error) {
	err := handle.AtomicPhase(key, func(tx *gorm.DB) (atomic.PhaseAction, error) {
		if payment == nil {
			// retrieve payment if necessary (we're recovering from a recovery point)
			if err := tx.Where("idempotency_key_id = ?", key.Id).First(payment).Error; err != nil {
				return nil, err
			}
		}

		if payment == nil {
			return nil, fmt.Errorf("bug: should have an payment for key %s at %s", key.Key, atomic.RecoveryPointPaymentCreated)
		}

		if ctx.Value("simulate_failure") != nil {
			return nil, fmt.Errorf("simulated failure")
		}

		account := new(Account)
		if err := s.db.Where("id = ?", payment.AccountId).First(account).Error; err != nil {
			return nil, err
		}

		recipient := new(Recipient)
		if err := s.db.Preload("Organization").Preload("BankAddress").Where("id = ?", payment.RecipientId).First(recipient).Error; err != nil {
			return nil, err
		}

		if recipient.BankAddress == nil {
			return nil, cerr.NewBadRequest("recipient does not have a bank address", nil)
		}

		response, err := s.bank.CreatePayment(ctx, bank.CreatePaymentRequest{
			AccountID:       *account.BankAccountId,
			AccountNumberID: *account.BankAccountNumberId,
			Amount:          payment.Amount,
			Description:     payment.Description,
			Creditor:        fmt.Sprintf("%s %s", recipient.Organization.LegalName, recipient.Name),
			PaymentRail:     bank.PaymentRail(payment.PaymentRail),
			PaymentMethod: bank.PaymentMethod{
				Ach: &bank.AchPaymentMethod{
					AccountNumber: recipient.BankAddress.AccountNumber,
					RoutingNumber: recipient.BankAddress.RoutingNumber,
				},
				Rtp: &bank.RtpPaymentMethod{
					AccountNumber: recipient.BankAddress.AccountNumber,
					RoutingNumber: recipient.BankAddress.RoutingNumber,
				},
			},
			IdempotencyKey: key.Key,
		})
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return atomic.Response{Status: apierr.StatusCode, Data: cerr.NewHttpError(apierr.StatusCode, "failed to create bank payment", apierr)}, nil
			}
			return atomic.Response{Status: 503, Data: cerr.NewHttpError(503, "failed to create bank payment", err)}, nil
		}

		payment.BankTransferId = &response.ID

		// update the transfer
		if err := tx.Updates(payment).Error; err != nil {
			return nil, err
		}
		return atomic.Response{Status: 201, Data: payment}, nil
	})
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func newPaymentsService(db *gorm.DB, bank bank.Bank) *PaymentsService {
	return &PaymentsService{
		db:   db,
		bank: bank,
	}
}

type CreatePaymentRequest struct {
	AccountID   uint64
	Description string
	Amount      int64
	RecipientID uint64
}

type ListPaymentsRequest struct {
	UserID      uint64
	AccountID   uint64
	RecipientID uint64
	Limit       int
	Cursor      string
}

type ListPaymentsResponse struct {
	Payments   []Payment
	NextCursor string
}

type GetPaymentResponse struct {
	Payment     *Payment
	BankPayment *bank.GetPaymentResponse
}

type Payment struct {
	Id               uint64 `gorm:"primaryKey;autoIncrement"`
	Amount           int64
	Description      string
	ChariotId        *string
	RecipientId      uint64
	Recipient        *Recipient `gorm:"foreignKey:RecipientId"`
	PaymentRail      PaymentRail
	BankTransferId   *string
	AccountId        uint64   `gorm:"column:account_id;index"`
	Account          *Account `gorm:"foreignKey:AccountId"`
	IdempotencyKeyId uint64   `gorm:"column:idempotency_key_id;index"`
	UserId           uint64   `gorm:"column:user_id"`
	CreatedAt        *time.Time
}

type PaymentRail string

const (
	AchPaymentRail PaymentRail = "ach"
	RtpPaymentRail PaymentRail = "rtp"
)

func (p *Payment) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id            uint64      `json:"id"`
		Amount        int64       `json:"amount"`
		Description   string      `json:"description"`
		ChariotId     *string     `json:"chariot_id"`
		RecipientId   uint64      `json:"recipient_id"`
		PaymentRail   PaymentRail `json:"payment_rail"`
		BankPaymentId *string     `json:"bank_transfer_id"`
		AccountId     uint64      `json:"account_id"`
		CreatedAt     string      `json:"created_at"`
	}{
		Id:            p.Id,
		Amount:        p.Amount,
		Description:   p.Description,
		ChariotId:     p.ChariotId,
		RecipientId:   p.RecipientId,
		PaymentRail:   p.PaymentRail,
		BankPaymentId: p.BankTransferId,
		AccountId:     p.AccountId,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
	})
}
