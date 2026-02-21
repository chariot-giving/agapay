package main

import (
	"fmt"

	"github.com/chariot-giving/agapay/pkg/chain"
	"github.com/chariot-giving/agapay/pkg/registry"
	solanaregistry "github.com/chariot-giving/agapay/pkg/registry/solana"
	temporegistry "github.com/chariot-giving/agapay/pkg/registry/tempo"
	"github.com/chariot-giving/agapay/pkg/settlement"
	solanasettlement "github.com/chariot-giving/agapay/pkg/settlement/solana"
	temposettlement "github.com/chariot-giving/agapay/pkg/settlement/tempo"
	"github.com/ethereum/go-ethereum/common"
	solanago "github.com/gagliardetto/solana-go"
)

// newRegistry creates a Registry implementation based on the config's chain.
func newRegistry(cfg *liveConfig) (registry.Registry, error) {
	switch cfg.Chain {
	case "solana", "":
		if cfg.ProgramID == "" {
			return nil, fmt.Errorf("program_id is required for Solana")
		}
		progPubkey, err := solanago.PublicKeyFromBase58(cfg.ProgramID)
		if err != nil {
			return nil, fmt.Errorf("invalid program ID: %w", err)
		}
		return solanaregistry.NewClient(cfg.RPCURL, cfg.WSURL, progPubkey, chain.SolanaDevnet), nil

	case "tempo":
		if cfg.RegistryAddress == "" {
			return nil, fmt.Errorf("registry_address is required for Tempo")
		}
		registryAddr := common.HexToAddress(cfg.RegistryAddress)
		return temporegistry.NewClient(cfg.RPCURL, registryAddr, cfg.EVMChainID, chain.TempoModerato), nil

	default:
		return nil, fmt.Errorf("unsupported chain: %q (use 'solana' or 'tempo')", cfg.Chain)
	}
}

// newSettler creates a Settler implementation based on the config's chain.
func newSettler(cfg *liveConfig) (settlement.Settler, error) {
	switch cfg.Chain {
	case "solana", "":
		if cfg.MintAddress == "" {
			return nil, fmt.Errorf("mint_address is required for Solana settlement")
		}
		mintPubkey, err := solanago.PublicKeyFromBase58(cfg.MintAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid mint address: %w", err)
		}
		return solanasettlement.NewPaymentClient(cfg.RPCURL, cfg.WSURL, mintPubkey, chain.SolanaDevnet), nil

	case "tempo":
		if cfg.RouterAddress == "" || cfg.TokenAddress == "" {
			return nil, fmt.Errorf("router_address and token_address are required for Tempo settlement")
		}
		routerAddr := common.HexToAddress(cfg.RouterAddress)
		tokenAddr := common.HexToAddress(cfg.TokenAddress)
		return temposettlement.NewPaymentClient(cfg.RPCURL, routerAddr, tokenAddr, cfg.EVMChainID, chain.TempoModerato), nil

	default:
		return nil, fmt.Errorf("unsupported chain: %q", cfg.Chain)
	}
}

// chainName returns a display name for the chain config.
func chainName(cfg *liveConfig) string {
	switch cfg.Chain {
	case "tempo":
		return "Tempo"
	default:
		return "Solana"
	}
}
