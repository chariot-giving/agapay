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

// Pagination - The paging information
type Pagination struct {
	Cursors PaginationCursors `json:"cursors"`

	// The total number of objects
	Total int32 `json:"total"`
}
