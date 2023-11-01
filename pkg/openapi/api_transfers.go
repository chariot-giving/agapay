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
	"github.com/gin-gonic/gin"
	"github.com/increase/increase-go"
)

// GetTransfer - Retrieve a transfer
func GetTransfer(c *gin.Context) {
	id := c.Param("id")

	achTransfer, err := bank.IncreaseClient.ACHTransfers.Get(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	transfer := Transfer{
		Id:            achTransfer.ID,
		AccountId:     achTransfer.AccountID,
		Description:   achTransfer.StatementDescriptor,
		Amount:        achTransfer.Amount,
		AccountNumber: achTransfer.AccountNumber,
		RoutingNumber: achTransfer.RoutingNumber,
		Funding:       string(achTransfer.Funding),
		TransactionId: achTransfer.TransactionID,
		Status:        string(achTransfer.Status),
		CreatedAt:     achTransfer.CreatedAt,
	}

	c.JSON(http.StatusOK, &transfer)
}

// ListTransfers - List transfers
func ListTransfers(c *gin.Context) {
	limitQuery := c.DefaultQuery("limit", "100")
	limit, err := strconv.ParseInt(limitQuery, 10, 64)
	if err != nil {
		limit = 100
	}

	accountId, ok := c.GetQuery("account_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required"})
		return
	}
	listParams := increase.ACHTransferListParams{
		AccountID: increase.String(accountId),
		Limit:     increase.Int(limit),
	}

	cursor, ok := c.GetQuery("cursor")
	if ok {
		listParams.Cursor = increase.String(cursor)
	} else {
		listParams.Cursor = increase.Null[string]()
	}

	response, err := bank.IncreaseClient.ACHTransfers.List(c, listParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	transfers := make([]Transfer, len(response.Data))
	for i, ach := range response.Data {
		transfers[i] = Transfer{
			Id:            ach.ID,
			AccountId:     ach.AccountID,
			Description:   ach.StatementDescriptor,
			Amount:        ach.Amount,
			AccountNumber: ach.AccountNumber,
			RoutingNumber: ach.RoutingNumber,
			Funding:       string(ach.Funding),
			TransactionId: ach.TransactionID,
			Status:        string(ach.Status),
			CreatedAt:     ach.CreatedAt,
		}
	}

	transferList := TransferList{
		Data: transfers,
		Paging: Pagination{
			Total: int32(len(response.Data)),
			Cursors: PaginationCursors{
				Before: cursor,
				After:  response.NextCursor,
			},
		},
	}

	c.JSON(http.StatusOK, &transferList)
}

// TransferFunds - Transfer funds
func TransferFunds(c *gin.Context) {
	transfer := new(Transfer)
	if err := c.ShouldBindJSON(transfer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	achTransfer, err := bank.IncreaseClient.ACHTransfers.New(c, increase.ACHTransferNewParams{
		AccountID:           increase.String(transfer.AccountId),
		StatementDescriptor: increase.String(transfer.Description),
		Amount:              increase.Int(transfer.Amount),
		AccountNumber:       increase.String(transfer.AccountNumber),
		RoutingNumber:       increase.String(transfer.RoutingNumber),
		Funding:             increase.F[increase.ACHTransferNewParamsFunding](increase.ACHTransferNewParamsFunding(transfer.Funding)),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	transfer.Id = achTransfer.ID
	transfer.TransactionId = achTransfer.TransactionID
	transfer.Status = string(achTransfer.Status)
	transfer.CreatedAt = achTransfer.CreatedAt

	c.JSON(http.StatusOK, transfer)
}
