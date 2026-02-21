// Package chain provides blockchain-agnostic types used across registry
// and settlement implementations.
package chain

// Address represents a blockchain address in its native string encoding.
// Solana uses base58, EVM chains use hex with 0x prefix.
type Address string

func (a Address) String() string { return string(a) }

// TxHash represents a transaction identifier in its native string encoding.
type TxHash string

func (h TxHash) String() string { return string(h) }

// ID identifies a specific chain and network combination.
type ID string

const (
	SolanaDevnet  ID = "solana-devnet"
	SolanaMainnet ID = "solana-mainnet"
	TempoModerato ID = "tempo-moderato"
	TempoMainnet  ID = "tempo-mainnet"
)

func (id ID) String() string { return string(id) }
