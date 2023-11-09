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

	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/gin-gonic/gin"
)

// GetAccountBalances - Retrieve account balances
func (api *openAPIServer) GetAccountBalances(c *gin.Context) {
	accountId := c.Param("id")

	balanceLookup, err := api.core.Accounts.GetBalance(c, accountId)
	if err != nil {
		cErr := new(cerr.HttpError)
		if errors.As(err, &cErr) {
			c.Error(cErr)
			return
		}
		c.Error(cerr.NewInternalServerError("failed to retrieve account balance", err))
		return
	}

	balance := AccountBalance{
		AccountId:        accountId,
		CurrentBalance:   balanceLookup.CurrentBalance,
		AvailableBalance: balanceLookup.AvailableBalance,
	}

	c.JSON(http.StatusOK, &balance)
}
