package bank

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/increase/increase-go"
	"github.com/increase/increase-go/option"
)

// increaseBank is a Bank implementation for Increase, Inc.
type increaseBank struct {
	client *increase.Client
}

// CreateAccount implements Bank.
func (bank *increaseBank) CreateAccount(ctx context.Context, request CreateAccountRequest) (*CreateAccountResponse, error) {
	bankIdempotencyKey := fmt.Sprintf("bank-account-%s", request.IdempotencyKey)
	bankAccount, err := bank.client.Accounts.New(ctx, increase.AccountNewParams{
		Name: increase.String(request.Name),
	}, option.WithHeader("Idempotency-Key", bankIdempotencyKey))
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to create bank account", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to create bank account", err)
	}

	return &CreateAccountResponse{
		ID: bankAccount.ID,
	}, nil
}

// CreateAccountNumber implements Bank.
func (bank *increaseBank) CreateAccountNumber(ctx context.Context, request CreateAccountNumberRequest) (*CreateAccountNumberResponse, error) {
	bankIdempotencyKey := fmt.Sprintf("bank-account-number-%s", request.IdempotencyKey)
	accountNo, err := bank.client.AccountNumbers.New(ctx, increase.AccountNumberNewParams{
		AccountID: increase.String(request.AccountID),
		Name:      increase.String(request.Name),
		InboundACH: increase.F[increase.AccountNumberNewParamsInboundACH](increase.AccountNumberNewParamsInboundACH{
			DebitStatus: increase.F[increase.AccountNumberNewParamsInboundACHDebitStatus](increase.AccountNumberNewParamsInboundACHDebitStatusBlocked),
		}),
	}, option.WithHeader("Idempotency-Key", bankIdempotencyKey))
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to create bank account numbers", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to create bank account numbers", err)
	}

	return &CreateAccountNumberResponse{
		ID: accountNo.ID,
	}, nil
}

// CreatePayment implements Bank.
func (bank *increaseBank) CreatePayment(ctx context.Context, request CreatePaymentRequest) (*CreatePaymentResponse, error) {
	bankIdempotencyKey := fmt.Sprintf("bank-payment-%s", request.IdempotencyKey)
	if request.PaymentRail == AchPaymentRail {
		ach := request.PaymentMethod.Ach
		if ach == nil {
			return nil, cerr.NewInternalServerError("ACH payment method required", nil)
		}
		// use ACH
		transfer, err := bank.client.ACHTransfers.New(ctx, increase.ACHTransferNewParams{
			AccountID:           increase.String(request.AccountID),
			Amount:              increase.Int(request.Amount),
			StatementDescriptor: increase.String(request.Description),
			IndividualName:      increase.String(request.Creditor),
			AccountNumber:       increase.String(ach.AccountNumber),
			RoutingNumber:       increase.String(ach.RoutingNumber),
			RequireApproval:     increase.Bool(false),
		}, option.WithHeader("Idempotency-Key", bankIdempotencyKey))
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return nil, cerr.NewHttpError(apierr.StatusCode, "failed to create ACH transfer payment", apierr)
			}
			return nil, cerr.NewHttpError(503, "failed to create ACH transfer payment", err)
		}

		return &CreatePaymentResponse{
			ID:            transfer.ID,
			PaymentRail:   AchPaymentRail,
			TransactionID: transfer.TransactionID,
			Status:        string(transfer.Status),
		}, nil
	} else if request.PaymentRail == RtpPaymentRail {
		rtp := request.PaymentMethod.Rtp
		if rtp == nil {
			return nil, cerr.NewInternalServerError("RTP payment method required", nil)
		}
		// use RTP
		transfer, err := bank.client.RealTimePaymentsTransfers.New(ctx, increase.RealTimePaymentsTransferNewParams{
			CreditorName:             increase.String(request.Creditor),
			Amount:                   increase.Int(request.Amount),
			RemittanceInformation:    increase.String(request.Description),
			SourceAccountNumberID:    increase.String(request.AccountNumberID),
			DestinationAccountNumber: increase.String(rtp.AccountNumber),
			DestinationRoutingNumber: increase.String(rtp.RoutingNumber),
			RequireApproval:          increase.Bool(false),
		})
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return nil, cerr.NewHttpError(apierr.StatusCode, "failed to create RTP transfer payment", apierr)
			}
			return nil, cerr.NewHttpError(503, "failed to create RTP transfer payment", err)
		}

		return &CreatePaymentResponse{
			ID:            transfer.ID,
			PaymentRail:   RtpPaymentRail,
			TransactionID: transfer.TransactionID,
			Status:        string(transfer.Status),
		}, nil
	} else {
		return nil, cerr.NewInternalServerError("invalid payment rail", nil)
	}
}

// GetAccountNumbers implements Bank.
func (bank *increaseBank) GetAccountDetails(ctx context.Context, request GetAccountDetailsRequest) (*GetAccountDetailsResponse, error) {
	bankAccount, err := bank.client.Accounts.Get(ctx, request.AccountID)
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to get account", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to get account", err)
	}

	accountNumbers, err := bank.client.AccountNumbers.List(ctx, increase.AccountNumberListParams{
		AccountID: increase.String(request.AccountID),
		Status:    increase.F[increase.AccountNumberListParamsStatus](increase.AccountNumberListParamsStatusActive),
	})
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to get account numbers", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to get account numbers", err)
	}

	if len(accountNumbers.Data) == 0 {
		return nil, cerr.NewNotFoundError("account number not found", nil)
	}

	numbers := make([]AccountNumber, len(accountNumbers.Data))
	for i, accountNo := range accountNumbers.Data {
		numbers[i] = AccountNumber{
			Status:        AccountNumberStatus(accountNo.Status),
			AccountNumber: accountNo.AccountNumber,
			RoutingNumber: accountNo.RoutingNumber,
		}
	}

	return &GetAccountDetailsResponse{
		Status:  AccountStatus(bankAccount.Status),
		Numbers: numbers,
	}, nil
}

// GetAccountBalance implements Bank.
func (bank *increaseBank) GetAccountBalance(ctx context.Context, request GetAccountBalanceRequest) (*GetAccountBalanceResponse, error) {
	balanceLookup, err := bank.client.Accounts.Balance(ctx, request.AccountID, increase.AccountBalanceParams{
		AtTime: increase.Null[time.Time](),
	})
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to get account balance", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to get account balance", err)
	}

	return &GetAccountBalanceResponse{
		CurrentBalance:   balanceLookup.CurrentBalance,
		AvailableBalance: balanceLookup.AvailableBalance,
	}, nil
}

// GetPayment implements Bank.
func (bank *increaseBank) GetPayment(ctx context.Context, request GetPaymentRequest) (*GetPaymentResponse, error) {
	if request.PaymentRail == AchPaymentRail {
		achTransfer, err := bank.client.ACHTransfers.Get(ctx, request.ID)
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return nil, cerr.NewHttpError(apierr.StatusCode, "failed to get ACH transfer", apierr)
			}
			return nil, cerr.NewHttpError(503, "failed to get ACH transfer", err)
		}

		if achTransfer == nil {
			return nil, cerr.NewNotFoundError("ACH transfer not found", nil)
		}

		return &GetPaymentResponse{
			ID:            achTransfer.ID,
			AccountID:     achTransfer.AccountID,
			Amount:        achTransfer.Amount,
			Description:   achTransfer.StatementDescriptor,
			TransactionID: achTransfer.TransactionID,
			Status:        string(achTransfer.Status),
		}, nil
	} else if request.PaymentRail == RtpPaymentRail {
		rtpTransfer, err := bank.client.RealTimePaymentsTransfers.Get(ctx, request.ID)
		if err != nil {
			var apierr *increase.Error
			if errors.As(err, &apierr) {
				return nil, cerr.NewHttpError(apierr.StatusCode, "failed to get RTP transfer", apierr)
			}
			return nil, cerr.NewHttpError(503, "failed to get RTP transfer", err)
		}

		if rtpTransfer == nil {
			return nil, cerr.NewNotFoundError("RTP transfer not found", nil)
		}

		return &GetPaymentResponse{
			ID:            rtpTransfer.ID,
			AccountID:     rtpTransfer.AccountID,
			Amount:        rtpTransfer.Amount,
			Description:   rtpTransfer.RemittanceInformation,
			TransactionID: rtpTransfer.TransactionID,
			Status:        string(rtpTransfer.Status),
		}, nil
	} else {
		return nil, cerr.NewInternalServerError("invalid payment rail", nil)
	}
}

// GetTransfer implements Bank.
func (bank *increaseBank) GetTransfer(ctx context.Context, request GetTransferRequest) (*GetTransferResponse, error) {
	achTransfer, err := bank.client.ACHTransfers.Get(ctx, request.ID)
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to get ACH transfer", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to get ACH transfer", err)
	}

	if achTransfer == nil {
		return nil, cerr.NewNotFoundError("ACH transfer not found", nil)
	}

	return &GetTransferResponse{
		ID:            achTransfer.ID,
		AccountID:     achTransfer.AccountID,
		Amount:        achTransfer.Amount,
		Description:   achTransfer.StatementDescriptor,
		AccountNumber: achTransfer.AccountNumber,
		RoutingNumber: achTransfer.RoutingNumber,
		TransactionID: achTransfer.TransactionID,
		Funding:       TransferFunding(achTransfer.Funding),
		Status:        string(achTransfer.Status),
	}, nil
}

// GetTransaction implements Bank.
func (bank *increaseBank) GetTransaction(ctx context.Context, request GetTransactionRequest) (*Transaction, error) {
	transaction, err := bank.client.Transactions.Get(ctx, request.ID)
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to get transaction", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to get transaction", err)
	}

	if transaction == nil {
		return nil, cerr.NewNotFoundError("transaction not found", nil)
	}

	return &Transaction{
		ID:          transaction.ID,
		AccountID:   transaction.AccountID,
		Amount:      transaction.Amount,
		Description: transaction.Description,
		CreatedAt:   transaction.CreatedAt,
	}, nil
}

// ListTransactions implements Bank.
func (bank *increaseBank) ListTransactions(ctx context.Context, request ListTransactionsRequest) (*ListTransactionsResponse, error) {
	limit := request.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	if request.AccountID == "" {
		return nil, cerr.NewBadRequest("account_id is required to list transactions", nil)
	}

	listParams := increase.TransactionListParams{
		AccountID: increase.String(request.AccountID),
		Limit:     increase.Int(limit),
	}

	if request.Cursor != "" {
		listParams.Cursor = increase.String(request.Cursor)
	} else {
		listParams.Cursor = increase.Null[string]()
	}

	response, err := bank.client.Transactions.List(ctx, listParams)
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to list account transactions", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to list account transactions", err)
	}

	transactions := make([]Transaction, len(response.Data))
	for i, tx := range response.Data {
		transactions[i] = Transaction{
			ID:          tx.ID,
			AccountID:   tx.AccountID,
			Amount:      tx.Amount,
			Description: tx.Description,
			CreatedAt:   tx.CreatedAt,
		}
	}

	return &ListTransactionsResponse{
		Transactions: transactions,
		NextCursor:   response.NextCursor,
	}, nil
}

// TransferFunds implements Bank.
func (bank *increaseBank) TransferFunds(ctx context.Context, request TransferFundsRequest) (*TransferFundsResponse, error) {
	bankIdempotencyKey := fmt.Sprintf("bank-transfer-%s", request.IdempotencyKey)
	// use ACH
	transfer, err := bank.client.ACHTransfers.New(ctx, increase.ACHTransferNewParams{
		AccountID:           increase.String(request.AccountID),
		Amount:              increase.Int(request.Amount),
		StatementDescriptor: increase.String(request.Description),
		AccountNumber:       increase.String(request.AccountNumber),
		RoutingNumber:       increase.String(request.RoutingNumber),
		RequireApproval:     increase.Bool(false),
	}, option.WithHeader("Idempotency-Key", bankIdempotencyKey))
	if err != nil {
		var apierr *increase.Error
		if errors.As(err, &apierr) {
			return nil, cerr.NewHttpError(apierr.StatusCode, "failed to create ACH transfer payment", apierr)
		}
		return nil, cerr.NewHttpError(503, "failed to create ACH transfer payment", err)
	}

	return &TransferFundsResponse{
		ID:            transfer.ID,
		TransactionID: transfer.TransactionID,
		Status:        string(transfer.Status),
	}, nil
}

// NewIncreaseBank returns a new IncreaseBank
func NewIncreaseBank(client *increase.Client) Bank {
	return &increaseBank{
		client: client,
	}
}
