// Package registry defines the chain-agnostic interface and types for the
// Agapay on-chain nonprofit registry. Concrete implementations for specific
// blockchains live in sub-packages (e.g., registry/solana, registry/tempo).
package registry

import (
	"context"
	"crypto/sha256"

	"github.com/chariot-giving/agapay/pkg/chain"
)

// Organization is the chain-agnostic representation of a registered nonprofit.
type Organization struct {
	EIN            string
	Name           string
	Domain         string
	DIDURI         string
	IssuerID       chain.Address
	VCHash         [32]byte
	PaymentAddress chain.Address
	Active         bool
	CreatedAt      int64
	UpdatedAt      int64
}

// Issuer is the chain-agnostic representation of a registered issuer (service provider).
type Issuer struct {
	Authority chain.Address
	DID       string
	Active    bool
	CreatedAt int64
}

// RegisterIssuerParams contains the parameters for registering an issuer.
type RegisterIssuerParams struct {
	AuthoritySigner []byte        // private key bytes of the program authority
	IssuerAuthority chain.Address // public address of the issuer being registered
	DID             string
}

// RegisterOrgParams contains the parameters for registering an organization.
type RegisterOrgParams struct {
	IssuerSigner   []byte // private key bytes of the issuer
	EIN            string
	Name           string
	Domain         string
	DIDURI         string
	VCHash         [32]byte
	PaymentAddress chain.Address
}

// DeactivateOrgParams contains the parameters for deactivating an organization.
type DeactivateOrgParams struct {
	IssuerSigner []byte // private key bytes of the issuer
	EIN          string
}

// Registry defines the interface for interacting with the Agapay on-chain
// nonprofit registry, independent of the underlying blockchain.
type Registry interface {
	GetOrganization(ctx context.Context, ein string) (*Organization, error)
	GetIssuer(ctx context.Context, authority chain.Address) (*Issuer, error)
	RegisterIssuer(ctx context.Context, params RegisterIssuerParams) (chain.TxHash, error)
	RegisterOrganization(ctx context.Context, params RegisterOrgParams) (chain.TxHash, error)
	DeactivateOrganization(ctx context.Context, params DeactivateOrgParams) (chain.TxHash, error)
	ChainID() chain.ID
}

// ComputeVCHash computes a SHA-256 hash of one or more VC JWT tokens.
func ComputeVCHash(vcTokens ...string) [32]byte {
	h := sha256.New()
	for _, token := range vcTokens {
		h.Write([]byte(token))
	}
	var hash [32]byte
	copy(hash[:], h.Sum(nil))
	return hash
}
