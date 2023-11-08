package core

import (
	"github.com/chariot-giving/agapay/pkg/bank"
	"gorm.io/gorm"
)

type AgapayServer struct {
	Accounts     *AccountsService
	Recipients   *RecipientsService
	Transactions *TransactionsService
	Transfers    *TransfersService
	Payments     *PaymentsService
	db           *gorm.DB
}

func NewAgapayServer(db *gorm.DB, bank bank.Bank) *AgapayServer {
	return &AgapayServer{
		Accounts:     newAccountsService(db, bank),
		Recipients:   newRecipientsService(db),
		Transactions: newTransactionService(bank),
		Transfers:    newTransfersService(db, bank),
		Payments:     newPaymentsService(db, bank),
		db:           db,
	}
}
