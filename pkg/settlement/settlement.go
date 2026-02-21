// Package settlement defines the chain-agnostic interface and types for
// stablecoin payment settlement. Concrete implementations for specific
// blockchains live in sub-packages (e.g., settlement/solana, settlement/tempo).
package settlement

import (
	"context"

	"github.com/chariot-giving/agapay/pkg/chain"
)

// PaymentParams contains the chain-agnostic parameters for sending a stablecoin payment.
type PaymentParams struct {
	SignerKey []byte        // private key bytes of the payer
	Recipient chain.Address // recipient payment address (ATA on Solana, EVM address on Tempo)
	Amount    uint64        // amount in stablecoin minor units (6 decimals)
	MemoData  []byte        // public header or reference data to attach to the payment
}

// PaymentResult contains the chain-agnostic result of a stablecoin payment.
type PaymentResult struct {
	TxHash  chain.TxHash
	Amount  uint64
	ChainID chain.ID
}

// Settler defines the interface for sending stablecoin payments,
// independent of the underlying blockchain.
type Settler interface {
	SendPayment(ctx context.Context, params *PaymentParams) (*PaymentResult, error)
	ChainID() chain.ID
}

// CentsToUSDCUnits converts an amount in USD cents to USDC minor units (6 decimals).
// Example: 50000 cents ($500.00) -> 500_000_000 USDC units
func CentsToUSDCUnits(cents int64) uint64 {
	return uint64(cents) * 10_000
}

// USDCUnitsToCents converts USDC minor units to USD cents.
func USDCUnitsToCents(units uint64) int64 {
	return int64(units / 10_000)
}
