// Package registry provides a client for reading and writing to the Agapay
// on-chain nonprofit registry on Solana.
package registry

import "github.com/gagliardetto/solana-go"

// IssuerAccount mirrors the on-chain IssuerAccount struct.
type IssuerAccount struct {
	Authority solana.PublicKey
	DID       string
	Active    bool
	CreatedAt int64
}

// OrganizationAccount mirrors the on-chain OrganizationAccount struct.
type OrganizationAccount struct {
	EIN         string
	Name        string
	Domain      string
	DIDURI      string
	Issuer      solana.PublicKey
	VCHash      [32]byte
	USDCAddress solana.PublicKey
	Active      bool
	CreatedAt   int64
	UpdatedAt   int64
}
