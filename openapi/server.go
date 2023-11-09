package openapi

import (
	"github.com/chariot-giving/agapay/pkg/core"
	"github.com/gin-gonic/gin"
)

type openAPIServer struct {
	core *core.AgapayServer
}

// Get the GIN handler for the given operation name.
func (api *openAPIServer) getHandler(name string) gin.HandlerFunc {
	switch name {
	case "Index":
		return Index
	case "CreateAccount":
		return api.CreateAccount
	case "GetAccount":
		return api.GetAccount
	case "ListAccounts":
		return api.ListAccounts
	case "GetAccountDetails":
		return api.GetAccountDetails
	case "GetAccountBalances":
		return api.GetAccountBalances
	case "CreateRecipient":
		return api.CreateRecipient
	case "GetRecipient":
		return api.GetRecipient
	case "ListRecipients":
		return api.ListRecipients
	case "GetTransaction":
		return api.GetTransaction
	case "ListTransactions":
		return api.ListTransactions
	case "GetTransfer":
		return api.GetTransfer
	case "ListTransfers":
		return api.ListTransfers
	case "TransferFunds":
		return api.TransferFunds
	case "GetPayment":
		return api.GetPayment
	case "ListPayments":
		return api.ListPayments
	case "CreatePayment":
		return api.CreatePayment
	default:
		return nil
	}
}
