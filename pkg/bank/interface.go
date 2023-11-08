package bank

import (
	"context"
	"os"
	"time"

	"github.com/increase/increase-go"
	"github.com/increase/increase-go/option"
)

// Bank is an interface for interacting with a bank
type Bank interface {
	// Create an account
	CreateAccount(context.Context, CreateAccountRequest) (*CreateAccountResponse, error)

	// Create an account number
	CreateAccountNumber(context.Context, CreateAccountNumberRequest) (*CreateAccountNumberResponse, error)

	// Get account numbers
	GetAccountDetails(context.Context, GetAccountDetailsRequest) (*GetAccountDetailsResponse, error)

	// Get account balance
	GetAccountBalance(context.Context, GetAccountBalanceRequest) (*GetAccountBalanceResponse, error)

	// Transfer funds
	TransferFunds(context.Context, TransferFundsRequest) (*TransferFundsResponse, error)

	// Get transfer details
	GetTransfer(context.Context, GetTransferRequest) (*GetTransferResponse, error)

	// Create a payment
	CreatePayment(context.Context, CreatePaymentRequest) (*CreatePaymentResponse, error)

	// Get payment details
	GetPayment(context.Context, GetPaymentRequest) (*GetPaymentResponse, error)

	// Get transaction details
	GetTransaction(context.Context, GetTransactionRequest) (*Transaction, error)

	// List transactions
	ListTransactions(context.Context, ListTransactionsRequest) (*ListTransactionsResponse, error)
}

func NewBank() Bank {
	// INCREASE
	_, ok := os.LookupEnv("INCREASE_SANDBOX")
	if ok {
		client := increase.NewClient(
			// defaults to os.LookupEnv("INCREASE_API_KEY")
			option.WithEnvironmentSandbox(), // defaults to option.WithEnvironmentProduction()
		)
		return NewIncreaseBank(client)
	} else {
		// defaults to os.LookupEnv("INCREASE_API_KEY")
		client := increase.NewClient()
		return NewIncreaseBank(client)
	}
}

type CreateAccountRequest struct {
	Name           string
	IdempotencyKey string
}

type CreateAccountResponse struct {
	ID string
}

type CreateAccountNumberRequest struct {
	AccountID      string
	Name           string
	IdempotencyKey string
}

type CreateAccountNumberResponse struct {
	ID string
}

type GetAccountDetailsRequest struct {
	AccountID       string
	AccountNumberID string
}

type GetAccountDetailsResponse struct {
	Status  AccountStatus
	Numbers []AccountNumber
}

type AccountNumber struct {
	Status        AccountNumberStatus
	AccountNumber string
	RoutingNumber string
}

type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusInactive AccountStatus = "inactive"
)

type AccountNumberStatus string

const (
	AccountNumberStatusActive   AccountNumberStatus = "active"
	AccountNumberStatusDisabled AccountNumberStatus = "disabled"
	AccountNumberCancelled      AccountNumberStatus = "cancelled"
)

type GetAccountBalanceRequest struct {
	AccountID string
}

type GetAccountBalanceResponse struct {
	CurrentBalance   int64
	AvailableBalance int64
}

type TransferFundsRequest struct {
	AccountID      string
	Amount         int64
	Description    string
	AccountNumber  string
	RoutingNumber  string
	IdempotencyKey string
}

type TransferFundsResponse struct {
	ID            string
	TransactionID string
	Status        string
}

type GetTransferRequest struct {
	ID string
}

type GetTransferResponse struct {
	ID            string
	AccountID     string
	Description   string
	Amount        int64
	AccountNumber string
	RoutingNumber string
	TransactionID string
	Funding       TransferFunding
	Status        string
}

type TransferFunding string

const (
	TransferFundingChecking TransferFunding = "checking"
	TransferFundingSavings  TransferFunding = "savings"
)

type CreatePaymentRequest struct {
	AccountID       string
	AccountNumberID string
	Amount          int64
	Description     string
	Creditor        string
	PaymentRail     PaymentRail
	PaymentMethod   PaymentMethod
	IdempotencyKey  string
}

type PaymentMethod struct {
	Ach *AchPaymentMethod
	Rtp *RtpPaymentMethod
}

type AchPaymentMethod struct {
	AccountNumber string
	RoutingNumber string
}

type RtpPaymentMethod struct {
	AccountNumber string
	RoutingNumber string
}

type CreatePaymentResponse struct {
	ID            string
	PaymentRail   PaymentRail
	TransactionID string
	Status        string
}

type PaymentRail string

const (
	AchPaymentRail PaymentRail = "ach"
	RtpPaymentRail PaymentRail = "rtp"
)

type GetPaymentRequest struct {
	ID string
	PaymentRail
}

type GetPaymentResponse struct {
	ID            string
	AccountID     string
	Amount        int64
	Description   string
	TransactionID string
	Status        string
}

type Transaction struct {
	ID          string
	AccountID   string
	Amount      int64
	Description string
	CreatedAt   time.Time
}

type GetTransactionRequest struct {
	ID string
}

type ListTransactionsRequest struct {
	AccountID string
	Limit     int64
	Cursor    string
}

type ListTransactionsResponse struct {
	Transactions []Transaction
	NextCursor   string
}
