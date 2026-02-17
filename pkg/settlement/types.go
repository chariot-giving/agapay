// Package settlement handles USDC stablecoin transfers on Solana with
// SPL Memo containing the Agapay public header for payment data.
package settlement

import "github.com/gagliardetto/solana-go"

// PaymentParams contains the parameters for sending a USDC payment.
type PaymentParams struct {
	// Payer's Solana wallet (signer)
	PayerWallet solana.PrivateKey
	// Payer's USDC Associated Token Account
	PayerTokenAccount solana.PublicKey
	// Recipient's USDC Associated Token Account (from registry)
	RecipientTokenAccount solana.PublicKey
	// Amount in USDC minor units (6 decimals). $500.00 = 500_000_000
	Amount uint64
	// Public header JSON to attach as SPL Memo
	MemoData []byte
}

// PaymentResult contains the result of a USDC payment.
type PaymentResult struct {
	// Solana transaction signature
	Signature string
	// The amount transferred in USDC minor units
	Amount uint64
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
