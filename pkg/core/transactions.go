package core

import (
	"context"

	"github.com/chariot-giving/agapay/pkg/bank"
	"github.com/chariot-giving/agapay/pkg/cerr"
	"gorm.io/gorm"
)

type TransactionsService struct {
	db   *gorm.DB
	bank bank.Bank
}

func newTransactionService(db *gorm.DB, bank bank.Bank) *TransactionsService {
	return &TransactionsService{
		db:   db,
		bank: bank,
	}
}

func (s *TransactionsService) Get(ctx context.Context, id string) (*bank.Transaction, error) {
	return s.bank.GetTransaction(ctx, bank.GetTransactionRequest{
		ID: id,
	})
}

func (s *TransactionsService) List(ctx context.Context, listParams ListTransactionsRequest) (*ListTransactionsResponse, error) {
	// get the account
	account := new(Account)
	if err := s.db.Where("id = ?", listParams.AccountID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, cerr.NewNotFoundError("account not found", nil)
		}
		return nil, cerr.NewDatabaseError("failed to read account", err)
	}

	if account.BankAccountId == nil {
		return nil, cerr.NewNotFoundError("bank account not found", nil)
	}

	response, err := s.bank.ListTransactions(ctx, bank.ListTransactionsRequest{
		AccountID: *account.BankAccountId,
	})
	if err != nil {
		return nil, err
	}

	return &ListTransactionsResponse{
		Transactions: response.Transactions,
		NextCursor:   response.NextCursor,
	}, nil
}

type ListTransactionsRequest struct {
	AccountID uint64
	Limit     int64
	Cursor    string
}

type ListTransactionsResponse struct {
	Transactions []bank.Transaction
	NextCursor   string
}
