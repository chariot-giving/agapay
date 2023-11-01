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
	"time"
)

type Account struct {

	// The unique identifier for the account
	Id string `json:"id,omitempty"`

	// The name of the account
	Name string `json:"name,omitempty"`

	// The account number
	AccountNumber string `json:"account_number,omitempty"`

	// The American Bankers' Association (ABA) Routing Transit Number (RTN).
	RoutingNumber string `json:"routing_number,omitempty"`

	// The status of the account
	Status string `json:"status,omitempty"`

	// The date and time the account was created
	CreatedAt time.Time `json:"created_at,omitempty"`
}
