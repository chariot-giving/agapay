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
	"errors"
	"net/http"
	"strconv"

	"github.com/chariot-giving/agapay/pkg/bank"
	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/gin-gonic/gin"
)

// GetTransaction - Retrieve a transaction
func (api *openAPIServer) GetTransaction(c *gin.Context) {
	id := c.Param("id")

	tx, err := api.core.Transactions.Get(c, id)
	if err != nil {
		cErr := new(cerr.HttpError)
		if errors.As(err, &cErr) {
			c.Error(cErr)
			return
		}
		c.Error(cerr.NewInternalServerError("failed to retrieve transaction", err))
		return
	}

	transaction := Transaction{
		Id:          tx.ID,
		AccountId:   tx.AccountID,
		Amount:      tx.Amount,
		Description: tx.Description,
		CreatedAt:   tx.CreatedAt,
	}

	c.JSON(http.StatusOK, &transaction)
}

// ListTransactions - List transactions
func (api *openAPIServer) ListTransactions(c *gin.Context) {
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

	listParams := bank.ListTransactionsRequest{
		AccountID: accountId,
		Limit:     limit,
	}

	cursor, ok := c.GetQuery("cursor")
	if ok {
		listParams.Cursor = cursor
	}

	response, err := api.core.Transactions.List(c, listParams)
	if err != nil {
		cErr := new(cerr.HttpError)
		if errors.As(err, &cErr) {
			c.Error(cErr)
			return
		}
		c.Error(cerr.NewInternalServerError("failed to list transaction", err))
		return
	}

	transactions := make([]Transaction, len(response.Transactions))
	for i, tx := range response.Transactions {
		transactions[i] = Transaction{
			Id:          tx.ID,
			AccountId:   tx.AccountID,
			Amount:      tx.Amount,
			Description: tx.Description,
			CreatedAt:   tx.CreatedAt,
		}
	}

	transactionList := TransactionList{
		Data: transactions,
		Paging: Pagination{
			Total: int32(len(response.Transactions)),
			Cursors: PaginationCursors{
				Before: cursor,
				After:  response.NextCursor,
			},
		},
	}

	c.JSON(http.StatusOK, &transactionList)
}
