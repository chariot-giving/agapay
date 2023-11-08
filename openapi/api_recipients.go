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

	"github.com/chariot-giving/agapay/pkg/cerr"
	"github.com/chariot-giving/agapay/pkg/core"
	"github.com/gin-gonic/gin"
)

// CreateRecipient - Create a recipient
func (api *openAPIServer) CreateRecipient(c *gin.Context) {
	request := new(CreateRecipientRequest)
	if err := c.ShouldBindJSON(request); err != nil {
		c.Error(cerr.NewBadRequest("invalid request body", err))
		return
	}

	c.Error(cerr.NewHttpError(http.StatusNotImplemented, "not implemented", nil))
}

// GetRecipient - Retrieve a recipient
func (s *openAPIServer) GetRecipient(c *gin.Context) {
	id := c.Param("id")

	recipient, err := s.core.Recipients.Get(c, id)
	if err != nil {
		cErr := new(cerr.HttpError)
		if errors.As(err, &cErr) {
			c.Error(cErr)
			return
		}
		c.Error(cerr.NewInternalServerError("failed to retrieve recipient", err))
		return
	}

	c.JSON(http.StatusOK, &Recipient{
		Id:        recipient.Id.String(),
		Name:      recipient.Organization.LegalName,
		Ein:       recipient.Organization.Ein,
		Primary:   recipient.Primary,
		CreatedAt: recipient.CreatedAt,
		Status:    recipient.BankAddress.Status,
	})
}

// ListRecipients - List recipients
func (s *openAPIServer) ListRecipients(c *gin.Context) {
	limitQuery := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitQuery)
	if err != nil {
		limit = 100
	}

	listParams := core.ListRecipientsRequest{
		Limit: limit,
		Ein:   c.Query("ein"),
	}

	cursor, ok := c.GetQuery("cursor")
	if ok {
		listParams.Cursor = cursor
	}

	response, err := s.core.Recipients.List(c, listParams)
	if err != nil {
		cErr := new(cerr.HttpError)
		if errors.As(err, &cErr) {
			c.Error(cErr)
			return
		}
		c.Error(cerr.NewInternalServerError("error listing accounts", err))
		return
	}

	recipientList := make([]Recipient, len(response.Recipients))
	for i, recipient := range response.Recipients {
		recipientList[i] = Recipient{
			Id:        recipient.Id.String(),
			Name:      recipient.Organization.LegalName,
			Ein:       recipient.Organization.Ein,
			Primary:   recipient.Primary,
			CreatedAt: recipient.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, &RecipientList{
		Data: recipientList,
		Paging: Pagination{
			Total: int32(len(response.Recipients)),
			Cursors: PaginationCursors{
				Before: cursor,
				After:  response.NextCursor,
			},
		},
	})
}
