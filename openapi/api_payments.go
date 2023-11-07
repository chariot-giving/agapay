/*
 * Chariot Payments API
 *
 * The Chariot Payments REST API.
 *
 * API version: v1
 * Contact: developers@givechariot.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"net/http"
	"strconv"

	"github.com/chariot-giving/agapay/pkg/bank"
	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/chariot-giving/agapay/pkg/network"
	"github.com/gin-gonic/gin"
	"github.com/increase/increase-go"
)

// CreatePayment - Create a payment
func CreatePayment(c *gin.Context) {
	payment := new(Payment)
	if err := c.ShouldBindJSON(payment); err != nil {
		c.JSON(http.StatusBadRequest, cerr.NewBadRequest("invalid request body", err))
		return
	}

	// get payee
	electronicAccount, err := network.PayeeDB.GetPayeeElectronicAccount(payment.RecipientId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, cerr.NewInternalServerError("error retrieving payee", err))
		return
	}

	// find out how to pay this address based on routing numbers
	routingNumberResponse, err := bank.IncreaseClient.RoutingNumbers.List(c, increase.RoutingNumberListParams{
		RoutingNumber: increase.String(electronicAccount.RoutingNumber),
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, cerr.NewBadGatewayError("error retrieving routing number", err))
		return
	}

	if len(routingNumberResponse.Data) == 0 {
		c.JSON(http.StatusNotFound, cerr.NewNotFoundError("routing number not found", nil))
		return
	}

	destinationBank := routingNumberResponse.Data[0]
	if destinationBank.RealTimePaymentsTransfers == increase.RoutingNumberRealTimePaymentsTransfersSupported {
		// resolve the originating account number
		accountNumbers, err := bank.IncreaseClient.AccountNumbers.List(c, increase.AccountNumberListParams{
			AccountID: increase.String(payment.AccountId),
			Status:    increase.F[increase.AccountNumberListParamsStatus](increase.AccountNumberListParamsStatusActive),
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, cerr.NewBadRequest("error retrieving account numbers", err))
			return
		}

		if len(accountNumbers.Data) == 0 {
			c.JSON(http.StatusNotFound, cerr.NewBadRequest("no account numbers found", nil))
			return
		}
		accountNumberId := accountNumbers.Data[0].ID

		// use RTP
		transfer, err := bank.IncreaseClient.RealTimePaymentsTransfers.New(c, increase.RealTimePaymentsTransferNewParams{
			CreditorName:             increase.String(electronicAccount.Name),
			Amount:                   increase.Int(payment.Amount),
			RemittanceInformation:    increase.String(payment.Description),
			SourceAccountNumberID:    increase.String(accountNumberId),
			DestinationAccountNumber: increase.String(electronicAccount.AccountNumber),
			DestinationRoutingNumber: increase.String(electronicAccount.RoutingNumber),
			RequireApproval:          increase.Bool(false),
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, cerr.NewBadGatewayError("error creating RTP transfer", err))
			return
		}

		payment.Id = transfer.ID
		payment.TransactionId = transfer.TransactionID
		payment.Status = string(transfer.Status)
	} else if destinationBank.ACHTransfers == increase.RoutingNumberACHTransfersSupported {
		// use ACH
		transfer, err := bank.IncreaseClient.ACHTransfers.New(c, increase.ACHTransferNewParams{
			AccountID:           increase.String(payment.AccountId),
			Amount:              increase.Int(payment.Amount),
			StatementDescriptor: increase.String(payment.Description),
			AccountNumber:       increase.String(electronicAccount.AccountNumber),
			RoutingNumber:       increase.String(electronicAccount.RoutingNumber),
			RequireApproval:     increase.Bool(false),
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, cerr.NewBadGatewayError("error creating ACH transfer", err))
			return
		}

		payment.Id = transfer.ID
		payment.TransactionId = transfer.TransactionID
		payment.Status = string(transfer.Status)
	} else {
		// TODO: fallback to sending a check or fail the request??
		c.JSON(http.StatusBadRequest, cerr.NewBadRequest("destination account does not support RTP or ACH transfers", nil))
		return
	}

	c.Header("Location", "/payments/"+payment.Id)
	c.JSON(http.StatusOK, payment)
}

// GetPayment - Retrieve a payment
func GetPayment(c *gin.Context) {
	id := c.Param("id")

	// TODO: do we want to check RTP and then fallback to ACH or should we be explicit about the payment type?
	rtpTransfer, err := bank.IncreaseClient.RealTimePaymentsTransfers.Get(c, id)
	if err != nil {
		c.JSON(http.StatusBadGateway, cerr.NewBadGatewayError("error retrieving RTP transfer", err))
		return
	}

	if rtpTransfer == nil {
		achTransfer, err := bank.IncreaseClient.ACHTransfers.Get(c, id)
		if err != nil {
			c.JSON(http.StatusBadGateway, cerr.NewBadGatewayError("error retrieving ACH transfer", err))
			return
		}

		if achTransfer == nil {
			c.JSON(http.StatusNotFound, cerr.NewNotFoundError("ACH transfer not found", nil))
			return
		}

		// TODO: source recipient + chariot ID from database
		payment := Payment{
			Id:            achTransfer.ID,
			AccountId:     achTransfer.AccountID,
			Amount:        achTransfer.Amount,
			Description:   achTransfer.StatementDescriptor,
			TransactionId: achTransfer.TransactionID,
			Status:        string(achTransfer.Status),
			RecipientId:   "",
			ChariotId:     "",
		}

		c.JSON(http.StatusOK, &payment)
	}

	// TODO: source recipient + chariot ID from database
	payment := Payment{
		Id:            rtpTransfer.ID,
		AccountId:     rtpTransfer.AccountID,
		Amount:        rtpTransfer.Amount,
		Description:   rtpTransfer.RemittanceInformation,
		TransactionId: rtpTransfer.TransactionID,
		Status:        string(rtpTransfer.Status),
		RecipientId:   "",
		ChariotId:     "",
	}

	c.JSON(http.StatusOK, &payment)
}

// ListPayments - List payments
// TODO: reconcile different payment types and merging them into a single list - not ideal
func ListPayments(c *gin.Context) {
	limitQuery := c.DefaultQuery("limit", "100")
	limit, err := strconv.ParseInt(limitQuery, 10, 64)
	if err != nil {
		limit = 100
	}

	accountId, ok := c.GetQuery("account_id")
	if !ok {
		c.JSON(http.StatusBadRequest, cerr.NewBadRequest("account_id is required", nil))
		return
	}

	listParams := increase.RealTimePaymentsTransferListParams{
		AccountID: increase.String(accountId),
		Limit:     increase.Int(limit),
	}

	cursor, ok := c.GetQuery("cursor")
	if ok {
		listParams.Cursor = increase.String(cursor)
	} else {
		listParams.Cursor = increase.Null[string]()
	}

	// TODO: filter by recipient ID in our database first
	// recipientId := c.DefaultQuery("recipient_id", "")
	response, err := bank.IncreaseClient.RealTimePaymentsTransfers.List(c, listParams)
	if err != nil {
		c.JSON(http.StatusBadGateway, cerr.NewBadGatewayError("error retrieving RTP transfers", err))
		return
	}

	payments := make([]Payment, len(response.Data))
	for i, transfer := range response.Data {
		payments[i] = Payment{
			Id:            transfer.ID,
			AccountId:     transfer.AccountID,
			Amount:        transfer.Amount,
			Description:   transfer.RemittanceInformation,
			TransactionId: transfer.TransactionID,
			Status:        string(transfer.Status),
			RecipientId:   "",
			ChariotId:     "",
		}
	}

	paymentList := PaymentList{
		Data: payments,
		Paging: Pagination{
			Total: int32(len(response.Data)),
			Cursors: PaginationCursors{
				Before: cursor,
				After:  response.NextCursor,
			},
		},
	}

	c.JSON(http.StatusOK, &paymentList)
}