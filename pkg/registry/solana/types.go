// Package solana implements the Agapay Registry interface for Solana.
package solana

import solanago "github.com/gagliardetto/solana-go"

// IssuerAccount mirrors the on-chain Anchor IssuerAccount struct.
type IssuerAccount struct {
	Authority solanago.PublicKey
	DID       string
	Active    bool
	CreatedAt int64
}

// OrganizationAccount mirrors the on-chain Anchor OrganizationAccount struct.
type OrganizationAccount struct {
	EIN         string
	Name        string
	Domain      string
	DIDURI      string
	Issuer      solanago.PublicKey
	VCHash      [32]byte
	USDCAddress solanago.PublicKey
	Active      bool
	CreatedAt   int64
	UpdatedAt   int64
}
