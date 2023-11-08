package core

import (
	"context"

	"github.com/chariot-giving/agapay/pkg/bank"
)

type TransactionsService struct {
	bank bank.Bank
}

func newTransactionService(bank bank.Bank) *TransactionsService {
	return &TransactionsService{
		bank: bank,
	}
}

func (s *TransactionsService) Get(ctx context.Context, id string) (*bank.Transaction, error) {
	return s.bank.GetTransaction(ctx, bank.GetTransactionRequest{
		ID: id,
	})
}

func (s *TransactionsService) List(ctx context.Context, listParams bank.ListTransactionsRequest) (*bank.ListTransactionsResponse, error) {
	return s.bank.ListTransactions(ctx, listParams)
}
