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

// Recipient - A recipient is a verified nonprofit organization account that can receive payments.
type Recipient struct {

	// The unique identifier for the recipient
	Id string `json:"id,omitempty"`

	// The name of the recipient
	Name string `json:"name,omitempty"`

	// The Employer Identification Number (EIN) for the recipient.
	Ein string `json:"ein,omitempty"`

	// Indicates whether the recipient is the primary recipient for the EIN. Only one recipient can be the primary recipient for an EIN. 
	Primary bool `json:"primary,omitempty"`

	// The status of the recipient
	Status string `json:"status,omitempty"`

	// The date and time the recipient was created
	CreatedAt time.Time `json:"created_at,omitempty"`
}