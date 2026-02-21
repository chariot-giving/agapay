package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chariot-giving/agapay/pkg/chain"
	"github.com/chariot-giving/agapay/pkg/credential"
	"github.com/chariot-giving/agapay/pkg/did"
	"github.com/chariot-giving/agapay/pkg/ipfs"
	"github.com/chariot-giving/agapay/pkg/message"
	"github.com/chariot-giving/agapay/pkg/registry"
	solanaregistry "github.com/chariot-giving/agapay/pkg/registry/solana"
	"github.com/chariot-giving/agapay/pkg/settlement"
	solanasettlement "github.com/chariot-giving/agapay/pkg/settlement/solana"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	solanago "github.com/gagliardetto/solana-go"
	"github.com/spf13/cobra"
)

// liveConfig holds the saved configuration from setup.
type liveConfig struct {
	Chain  string `json:"chain"` // "solana" (default) or "tempo"
	RPCURL string `json:"rpc_url"`
	IPFSURL string `json:"ipfs_url"`

	// Solana-specific
	ProgramID        string `json:"program_id,omitempty"`
	WSURL            string `json:"ws_url,omitempty"`
	MintAddress      string `json:"mint_address,omitempty"`
	AuthorityKeypath string `json:"authority_keypath,omitempty"`

	// Tempo-specific
	RegistryAddress string `json:"registry_address,omitempty"`
	RouterAddress   string `json:"router_address,omitempty"`
	TokenAddress    string `json:"token_address,omitempty"`
	EVMChainID      int64  `json:"evm_chain_id,omitempty"`
	PrivateKeyHex   string `json:"private_key_hex,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agapay", "config.json")
}

func loadConfig() (*liveConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("read config: %w (run 'agapay setup' first)", err)
	}
	var cfg liveConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func saveConfig(cfg *liveConfig) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(configPath(), data, 0600)
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func setupCmd() *cobra.Command {
	var chainFlag, programID, rpcURL, wsURL, ipfsURL string
	var registryAddr, routerAddr, tokenAddr, privateKey string
	var evmChainID int64

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Provision testnet/devnet infrastructure for the live demo",
		Long: `Sets up infrastructure for the selected chain:

Solana (default):
  1. Checks prerequisites (IPFS daemon)
  2. Generates Solana keypairs
  3. Airdrops devnet SOL
  4. Creates a test USDC mint
  5. Saves configuration to ~/.agapay/config.json

Tempo:
  1. Checks prerequisites (IPFS daemon)
  2. Saves contract addresses and private key to config
  3. Verifies connectivity to Tempo RPC`,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch chainFlag {
			case "tempo":
				return runSetupTempo(rpcURL, ipfsURL, registryAddr, routerAddr, tokenAddr, privateKey, evmChainID)
			default:
				return runSetup(programID, rpcURL, wsURL, ipfsURL)
			}
		},
	}

	cmd.Flags().StringVar(&chainFlag, "chain", "solana", "Blockchain to use: solana or tempo")
	cmd.Flags().StringVar(&programID, "program-id", "", "Solana: deployed Agapay Registry program ID")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "RPC URL (defaults per chain)")
	cmd.Flags().StringVar(&wsURL, "ws", envOrDefault("SOLANA_WS_URL", "wss://api.devnet.solana.com"), "Solana WebSocket URL")
	cmd.Flags().StringVar(&ipfsURL, "ipfs", envOrDefault("IPFS_API_URL", "http://localhost:5001"), "IPFS API URL")
	cmd.Flags().StringVar(&registryAddr, "registry", "", "Tempo: deployed AgapayRegistry contract address")
	cmd.Flags().StringVar(&routerAddr, "router", "", "Tempo: deployed AgapayPaymentRouter contract address")
	cmd.Flags().StringVar(&tokenAddr, "token", "", "Tempo: TIP-20 token (USDC) contract address")
	cmd.Flags().StringVar(&privateKey, "private-key", "", "Tempo: hex-encoded private key for signing")
	cmd.Flags().Int64Var(&evmChainID, "chain-id", 42431, "Tempo: EVM chain ID (default: 42431 for Moderato)")

	return cmd
}

func runSetupTempo(rpcURL, ipfsURL, registryAddr, routerAddr, tokenAddr, privateKey string, evmChainID int64) error {
	printHeader("AGAPAY TEMPO TESTNET SETUP")

	if rpcURL == "" {
		rpcURL = envOrDefault("TEMPO_RPC_URL", "https://rpc.moderato.tempo.xyz")
	}
	if registryAddr == "" {
		return fmt.Errorf("--registry is required for Tempo setup")
	}
	if routerAddr == "" {
		return fmt.Errorf("--router is required for Tempo setup")
	}
	if tokenAddr == "" {
		return fmt.Errorf("--token is required for Tempo setup")
	}
	if privateKey == "" {
		return fmt.Errorf("--private-key is required for Tempo setup")
	}

	fmt.Printf("  Chain:     Tempo\n")
	fmt.Printf("  RPC URL:   %s\n", rpcURL)
	fmt.Printf("  IPFS URL:  %s\n", ipfsURL)
	fmt.Printf("  Registry:  %s\n", registryAddr)
	fmt.Printf("  Router:    %s\n", routerAddr)
	fmt.Printf("  Token:     %s\n", tokenAddr)
	fmt.Printf("  Chain ID:  %d\n", evmChainID)

	printStep(1, "Checking IPFS daemon")
	resp, err := http.Post(ipfsURL+"/api/v0/id", "", nil)
	if err != nil {
		fmt.Printf("  IPFS daemon not detected at %s (optional, will use mock)\n", ipfsURL)
	} else {
		resp.Body.Close()
		fmt.Printf("  IPFS daemon: OK (%s)\n", ipfsURL)
	}

	printStep(2, "Saving configuration")
	cfg := &liveConfig{
		Chain:           "tempo",
		RPCURL:          rpcURL,
		IPFSURL:         ipfsURL,
		RegistryAddress: registryAddr,
		RouterAddress:   routerAddr,
		TokenAddress:    tokenAddr,
		EVMChainID:      evmChainID,
		PrivateKeyHex:   privateKey,
	}
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("  Config saved to: %s\n", configPath())

	printHeader("SETUP COMPLETE")
	fmt.Printf("  Chain:    Tempo (Moderato testnet)\n")
	fmt.Printf("  Registry: %s\n", registryAddr)
	fmt.Printf("  Router:   %s\n", routerAddr)
	fmt.Printf("  Token:    %s\n", tokenAddr)
	fmt.Printf("  Config:   %s\n\n", configPath())
	fmt.Println(`Next steps:
  1. Ensure 'ipfs daemon' is running
  2. Fund your account via the Tempo Moderato faucet
  3. Run the live demo:
     go run ./cmd/agapay live --chain tempo`)

	return nil
}

func runSetup(programID, rpcURL, wsURL, ipfsURL string) error {
	printHeader("AGAPAY DEVNET SETUP")
	ctx := context.Background()

	if rpcURL == "" {
		rpcURL = envOrDefault("SOLANA_RPC_URL", "https://api.devnet.solana.com")
	}
	if programID == "" {
		return fmt.Errorf("--program-id is required for Solana setup")
	}

	progPubkey, err := solanago.PublicKeyFromBase58(programID)
	if err != nil {
		return fmt.Errorf("invalid program ID: %w", err)
	}
	fmt.Printf("  Program ID: %s\n", progPubkey)
	fmt.Printf("  RPC URL:    %s\n", rpcURL)
	fmt.Printf("  WS URL:     %s\n", wsURL)
	fmt.Printf("  IPFS URL:   %s\n", ipfsURL)

	printStep(1, "Checking prerequisites")
	resp, err := http.Post(ipfsURL+"/api/v0/id", "", nil)
	if err != nil {
		return fmt.Errorf("IPFS daemon not reachable at %s: %w\nRun 'ipfs daemon' first", ipfsURL, err)
	}
	resp.Body.Close()
	fmt.Printf("  IPFS daemon: OK (%s)\n", ipfsURL)

	// Create an RPC client for devnet operations
	regClient := solanaregistry.NewClient(rpcURL, wsURL, progPubkey, chain.SolanaDevnet)
	rpcClient := regClient.GetRPCClient()

	printStep(2, "Loading or generating Solana keypair")
	var authorityKey solanago.PrivateKey
	existingCfg, cfgErr := loadConfig()
	if cfgErr == nil && existingCfg.AuthorityKeypath != "" {
		keyBytes, err := hex.DecodeString(existingCfg.AuthorityKeypath)
		if err == nil && len(keyBytes) == 64 {
			authorityKey = solanago.PrivateKey(keyBytes)
			fmt.Printf("  Reusing existing authority: %s\n", authorityKey.PublicKey())
		}
	}
	if authorityKey == nil {
		ak, err := solanago.NewRandomPrivateKey()
		if err != nil {
			return fmt.Errorf("generate authority key: %w", err)
		}
		authorityKey = solanago.PrivateKey(ak)
		fmt.Printf("  Generated new authority: %s\n", authorityKey.PublicKey())
	}

	cfg := &liveConfig{
		Chain:            "solana",
		ProgramID:        programID,
		RPCURL:           rpcURL,
		WSURL:            wsURL,
		IPFSURL:          ipfsURL,
		AuthorityKeypath: hex.EncodeToString(authorityKey),
	}
	if cfgErr == nil && existingCfg.MintAddress != "" {
		cfg.MintAddress = existingCfg.MintAddress
	}
	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("  Config saved to: %s\n", configPath())

	printStep(3, "Checking SOL balance / airdropping")
	balance, err := solanasettlement.GetSOLBalance(ctx, rpcClient, authorityKey.PublicKey())
	if err != nil {
		fmt.Printf("  Could not check balance: %v\n", err)
		balance = 0
	}
	balanceSOL := float64(balance) / float64(solanago.LAMPORTS_PER_SOL)
	fmt.Printf("  Current balance: %.4f SOL\n", balanceSOL)

	if balance < solanago.LAMPORTS_PER_SOL {
		fmt.Printf("  Requesting airdrop of 2 SOL...\n")
		if err := solanasettlement.AirdropSOL(ctx, rpcClient, authorityKey.PublicKey(), 2*solanago.LAMPORTS_PER_SOL); err != nil {
			fmt.Printf("  Airdrop failed (rate limit): %v\n", err)
			fmt.Printf("\n  Please fund this address manually via https://faucet.solana.com\n")
			fmt.Printf("  Address: %s\n", authorityKey.PublicKey())
			fmt.Printf("  Then re-run: agapay setup --program-id %s\n\n", programID)
			return nil
		}
		fmt.Printf("  Airdrop: OK\n")
	} else {
		fmt.Printf("  Balance sufficient, skipping airdrop\n")
	}

	if cfg.MintAddress != "" {
		printStep(4, "Test USDC mint already exists")
		fmt.Printf("  Mint: %s\n", cfg.MintAddress)
	} else {
		printStep(4, "Creating test USDC mint (6 decimals)")
		mintPubkey, _, err := solanasettlement.CreateTestMint(ctx, rpcClient, authorityKey, 6, wsURL)
		if err != nil {
			return fmt.Errorf("create test mint: %w", err)
		}
		cfg.MintAddress = mintPubkey.String()
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("  Test USDC Mint: %s\n", mintPubkey)
	}

	printHeader("SETUP COMPLETE")
	fmt.Printf("  Authority: %s\n", authorityKey.PublicKey())
	fmt.Printf("  Program:   %s\n", programID)
	fmt.Printf("  Mint:      %s\n", cfg.MintAddress)
	fmt.Printf("  Config:    %s\n\n", configPath())
	fmt.Println(`Next steps:
  1. Ensure 'ipfs daemon' is running
  2. Run the live demo:
     go run ./cmd/agapay live`)

	return nil
}

func liveCmd() *cobra.Command {
	var chainFlag, programID, rpcURL, wsURL, ipfsURL string

	cmd := &cobra.Command{
		Use:   "live",
		Short: "Run the full E2E demo against real infrastructure",
		Long: `Executes the complete Agapay flow against real infrastructure:
1. Issue all 4 Verifiable Credentials
2. Register Chariot as issuer on-chain
3. Register nonprofit organization on-chain
4. Verify full credential chain (Entity + Org + ControlPerson)
5. Build ISO 20022-inspired payment, encrypt PII
6. Pin encrypted data to local IPFS (real CID)
7. Send stablecoin transfer with payment metadata
8. Retrieve and decrypt payment data from IPFS

Supports both Solana devnet and Tempo Moderato testnet.
Requires: 'agapay setup' to have been run first.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if chainFlag == "tempo" {
				return runLiveTempo(rpcURL, ipfsURL)
			}
			return runLive(programID, rpcURL, wsURL, ipfsURL)
		},
	}

	cmd.Flags().StringVar(&chainFlag, "chain", "", "Override chain from config: solana or tempo")
	cmd.Flags().StringVar(&programID, "program-id", "", "Override program ID from config (Solana)")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "Override RPC URL")
	cmd.Flags().StringVar(&wsURL, "ws", "", "Override WebSocket URL (Solana)")
	cmd.Flags().StringVar(&ipfsURL, "ipfs", "", "Override IPFS API URL")

	return cmd
}

func runLiveTempo(rpcOverride, ipfsOverride string) error {
	printHeader("AGAPAY LIVE TEMPO TESTNET DEMO")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("no config found. Run 'agapay setup --chain tempo' first")
	}
	if cfg.Chain != "tempo" {
		return fmt.Errorf("config is for %s, not tempo. Run 'agapay setup --chain tempo' first", chainName(cfg))
	}
	if rpcOverride != "" {
		cfg.RPCURL = rpcOverride
	}
	if ipfsOverride != "" {
		cfg.IPFSURL = ipfsOverride
	}

	fmt.Printf("  Chain:     Tempo\n")
	fmt.Printf("  RPC URL:   %s\n", cfg.RPCURL)
	fmt.Printf("  IPFS URL:  %s\n", cfg.IPFSURL)
	fmt.Printf("  Registry:  %s\n", cfg.RegistryAddress)
	fmt.Printf("  Router:    %s\n", cfg.RouterAddress)
	fmt.Printf("  Token:     %s\n", cfg.TokenAddress)

	reg, err := newRegistry(cfg)
	if err != nil {
		return fmt.Errorf("create registry client: %w", err)
	}
	settler, err := newSettler(cfg)
	if err != nil {
		return fmt.Errorf("create settler client: %w", err)
	}

	var ipfsStore ipfs.IPFSStore
	resp, ipfsErr := http.Post(cfg.IPFSURL+"/api/v0/id", "", nil)
	if ipfsErr != nil {
		fmt.Printf("  IPFS: not available, using mock (encrypted data stored in-memory)\n")
		ipfsStore = ipfs.NewMockClient()
	} else {
		resp.Body.Close()
		fmt.Printf("  IPFS: connected (%s)\n", cfg.IPFSURL)
		ipfsStore = ipfs.NewClient(cfg.IPFSURL)
	}

	signerKey := common.FromHex(cfg.PrivateKeyHex)

	// -----------------------------------------------------------------------
	// Step 1: Generate identity keypairs (chain-agnostic)
	// -----------------------------------------------------------------------
	printStep(1, "Generating identity keypairs")

	chariotKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate chariot keypair: %w", err)
	}
	chariotDID := "did:web:givechariot.com"
	fmt.Printf("  Chariot (VC Issuer) DID: %s\n", chariotDID)

	nonprofitKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate nonprofit keypair: %w", err)
	}
	nonprofitDomain := "agapay.redcross.org"
	nonprofitDID := "did:web:" + nonprofitDomain
	fmt.Printf("  Nonprofit (Holder) DID: %s\n", nonprofitDID)

	personKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate person keypair: %w", err)
	}
	personDID := did.CreateDIDKey(personKP.PublicKey)
	fmt.Printf("  Control Person DID: %s\n", personDID)

	// Derive the EVM address from the signer key for on-chain identity
	evmKey, err := ethcrypto.ToECDSA(signerKey)
	if err != nil {
		return fmt.Errorf("parse signer key: %w", err)
	}
	issuerAddress := ethcrypto.PubkeyToAddress(evmKey.PublicKey)
	paymentAddress := issuerAddress
	fmt.Printf("  Issuer/Payer EVM address: %s\n", issuerAddress.Hex())

	// -----------------------------------------------------------------------
	// Step 2: Issue Verifiable Credentials
	// -----------------------------------------------------------------------
	printStep(2, "Issuing Verifiable Credentials")

	vcIssuer, err := credential.NewIssuer(chariotDID, chariotKP.PrivateKey)
	if err != nil {
		return fmt.Errorf("create issuer: %w", err)
	}

	entityVC, err := vcIssuer.IssueNonprofitEntity(nonprofitDID, &credential.NonprofitEntityClaims{
		EIN:       "530196605",
		LegalName: "American National Red Cross",
		PhysicalAddress: credential.PostalAddress{
			Line1: "430 17th St NW", City: "Washington", State: "DC",
			PostalCode: "20006", Country: "US",
		},
		IRSSubsectionCode: "03",
		IRSPub78:          true,
		Incorporation: credential.Incorporation{
			Date: "1881-05-21", State: "DC", Jurisdiction: "federal",
		},
	})
	if err != nil {
		return fmt.Errorf("issue entity VC: %w", err)
	}
	fmt.Printf("  NonprofitEntityCredential: %s...\n", entityVC[:60])

	personVC, err := vcIssuer.IssueControlPerson(personDID, &credential.ControlPersonClaims{
		FullName:  "Gail McGovern",
		Title:     "President & CEO",
		Email:     "ceo@redcross.org",
		EntityDID: nonprofitDID,
		EntityEIN: "530196605",
		Role:      "officer",
	})
	if err != nil {
		return fmt.Errorf("issue control person VC: %w", err)
	}
	fmt.Printf("  ControlPersonCredential:   %s...\n", personVC[:60])

	orgVC, err := vcIssuer.IssueOrganization(nonprofitDID, &credential.OrganizationClaims{
		OrganizationName: "American Red Cross",
		Domain:           "redcross.org",
		EntityDID:        nonprofitDID,
		EntityEIN:        "530196605",
		Affiliation:      "independent_nonprofit",
		NTEECode:         "P50",
		MissionStatement: "Prevent and alleviate human suffering in the face of emergencies.",
	})
	if err != nil {
		return fmt.Errorf("issue org VC: %w", err)
	}
	fmt.Printf("  OrganizationCredential:    %s...\n", orgVC[:60])

	addrVC, err := vcIssuer.IssueAddress(nonprofitDID, &credential.AddressClaims{
		OrganizationDID:         nonprofitDID,
		OrganizationEIN:         "530196605",
		AddressType:             "evm_wallet",
		SupportedPaymentMethods: []string{"usdc"},
		SolanaWallet:            paymentAddress.Hex(),
	})
	if err != nil {
		return fmt.Errorf("issue address VC: %w", err)
	}
	fmt.Printf("  AddressCredential:         %s...\n", addrVC[:60])

	// -----------------------------------------------------------------------
	// Step 3: DID Document
	// -----------------------------------------------------------------------
	printStep(3, "Setting up CNAME-delegated DID hosting (simulated DNS)")

	hosting := did.NewHostingService()
	hosting.RegisterExistingDID(nonprofitDomain, nonprofitKP,
		did.CreateDIDWebDocument(nonprofitDomain, nonprofitKP, "https://api.givechariot.com/v1/organizations/org_redcross"))
	doc, _ := hosting.GetDocument(nonprofitDomain)
	docJSON, _ := json.MarshalIndent(doc, "  ", "  ")
	fmt.Printf("  DNS CNAME: %s. CNAME dids.givechariot.com.\n", nonprofitDomain)
	fmt.Printf("  DID Document:\n  %s\n", string(docJSON))

	// -----------------------------------------------------------------------
	// Step 4: Register issuer on Tempo (via Registry interface)
	// -----------------------------------------------------------------------
	printStep(4, "Registering Chariot as issuer on Tempo")

	issuerAddr := chain.Address(issuerAddress.Hex())
	issuerAccount, err := reg.GetIssuer(ctx, issuerAddr)
	if err == nil && issuerAccount.Active {
		fmt.Printf("  Issuer already registered on-chain (skipping)\n")
		fmt.Printf("  Issuer DID: %s\n", issuerAccount.DID)
	} else {
		txHash, err := reg.RegisterIssuer(ctx, registry.RegisterIssuerParams{
			AuthoritySigner: signerKey,
			IssuerAuthority: issuerAddr,
			DID:             chariotDID,
		})
		if err != nil {
			return fmt.Errorf("register issuer on-chain: %w", err)
		}
		fmt.Printf("  Transaction: %s\n", txHash)
	}

	// -----------------------------------------------------------------------
	// Step 5: Register organization on Tempo (via Registry interface)
	// -----------------------------------------------------------------------
	printStep(5, "Registering nonprofit organization on Tempo")

	vcHash := registry.ComputeVCHash(entityVC, orgVC, addrVC)

	existingOrg, orgErr := reg.GetOrganization(ctx, "530196605")
	if orgErr == nil && existingOrg.Active {
		fmt.Printf("  Organization already registered on-chain (skipping)\n")
		fmt.Printf("  EIN: %s, Name: %s\n", existingOrg.EIN, existingOrg.Name)
	} else {
		txHash, err := reg.RegisterOrganization(ctx, registry.RegisterOrgParams{
			IssuerSigner:   signerKey,
			EIN:            "530196605",
			Name:           "American Red Cross",
			Domain:         "redcross.org",
			DIDURI:         nonprofitDID,
			VCHash:         vcHash,
			PaymentAddress: chain.Address(paymentAddress.Hex()),
		})
		if err != nil {
			return fmt.Errorf("register organization on-chain: %w", err)
		}
		fmt.Printf("  Transaction: %s\n", txHash)
	}

	// -----------------------------------------------------------------------
	// Step 6: Read back from registry (via Registry interface)
	// -----------------------------------------------------------------------
	printStep(6, "Reading organization from on-chain registry")

	org, err := reg.GetOrganization(ctx, "530196605")
	if err != nil {
		return fmt.Errorf("read organization: %w", err)
	}
	fmt.Printf("  EIN: %s\n", org.EIN)
	fmt.Printf("  Name: %s\n", org.Name)
	fmt.Printf("  Domain: %s\n", org.Domain)
	fmt.Printf("  DID: %s\n", org.DIDURI)
	fmt.Printf("  Payment Address: %s\n", org.PaymentAddress)
	fmt.Printf("  Active: %t\n", org.Active)

	// -----------------------------------------------------------------------
	// Step 7: Verify credential chain
	// -----------------------------------------------------------------------
	printStep(7, "Payer verifies full credential chain")

	verifier := credential.NewVerifier()
	chainResult, err := verifier.VerifyControlPersonChain(
		entityVC, orgVC, personVC, chariotKP.PublicKey,
	)
	if err != nil {
		return fmt.Errorf("verify credential chain: %w", err)
	}
	if !chainResult.Valid {
		return fmt.Errorf("credential chain invalid: %s", chainResult.Error)
	}
	fmt.Printf("  Chain verification: VALID\n")
	fmt.Printf("  Entity DID: %s\n", chainResult.EntityDID)
	fmt.Printf("  Organization: %s\n", chainResult.OrganizationName)
	fmt.Printf("  Control Person: %s\n", chainResult.ControlPerson)

	// -----------------------------------------------------------------------
	// Step 8: Build payment message
	// -----------------------------------------------------------------------
	printStep(8, "Building ISO 20022-inspired payment message")

	instruction := &message.AgapayPaymentInstruction{
		PublicHeader: message.PublicHeader{
			SenderDID:    "did:web:givechariot.com:payers:vanguard",
			RecipientDID: nonprofitDID,
			RecipientEIN: "530196605",
			PaymentType:  "donor_advised_fund_grant",
		},
		PrivateBody: message.PrivateBody{
			Transactions: []message.Transaction{
				{
					TransactionID: "tx_live_01",
					Amount:        50000,
					Currency:      "USD",
					Description:   "General Operating Support - Annual Grant",
					Donation: &message.Donation{
						Type:             "donor_advised_fund_grant",
						OrganizationName: "Vanguard Charitable",
						FundName:         "John Doe Giving Fund",
						Purpose:          "General Operating Support",
						Note:             "Annual grant from the Doe family",
					},
					Donors: []message.Donor{
						{
							Name:  "John Doe",
							Email: "john.doe@example.com",
							Phone: "415-555-1212",
							Address: &message.PostalAddress{
								Line1: "123 Main St", City: "San Francisco",
								State: "CA", PostalCode: "94105", Country: "US",
							},
						},
					},
				},
			},
			SenderReference: "VNG-2026-00142",
			RemittanceInfo:  "DAF Grant - Q1 2026 Distribution",
		},
	}

	header, body, err := message.Split(instruction)
	if err != nil {
		return fmt.Errorf("split payment instruction: %w", err)
	}
	fmt.Printf("  Message ID: %s\n", header.MessageID)
	fmt.Printf("  Total Amount: $%.2f (USD)\n", float64(header.TotalAmount)/100)
	fmt.Printf("  Payment Type: %s\n", header.PaymentType)

	// -----------------------------------------------------------------------
	// Step 9: Encrypt and pin to IPFS
	// -----------------------------------------------------------------------
	printStep(9, "Encrypting private body and pinning to IPFS")

	envelope, err := message.Encrypt(body, nonprofitKP.EncryptionPublicKey)
	if err != nil {
		return fmt.Errorf("encrypt private body: %w", err)
	}

	cid, err := ipfsStore.PinJSON(envelope)
	if err != nil {
		return fmt.Errorf("pin to IPFS: %w", err)
	}
	header.PrivateDataCID = cid
	fmt.Printf("  IPFS CID: %s\n", cid)

	// -----------------------------------------------------------------------
	// Step 10: Send payment via Settler interface
	// -----------------------------------------------------------------------
	printStep(10, "Sending TIP-20 payment via AgapayPaymentRouter")

	headerJSON, _ := json.Marshal(header)
	usdcAmount := settlement.CentsToUSDCUnits(header.TotalAmount)

	result, err := settler.SendPayment(ctx, &settlement.PaymentParams{
		SignerKey:  signerKey,
		Recipient: org.PaymentAddress,
		Amount:    usdcAmount,
		MemoData:  headerJSON,
	})
	if err != nil {
		return fmt.Errorf("send payment: %w", err)
	}

	fmt.Printf("  Payment Transaction: %s\n", result.TxHash)
	fmt.Printf("  Amount: %d USDC units ($%.2f)\n", result.Amount, float64(header.TotalAmount)/100)
	fmt.Printf("  Chain: %s\n", result.ChainID)

	// -----------------------------------------------------------------------
	// Step 11: Decrypt
	// -----------------------------------------------------------------------
	printStep(11, "Recipient retrieves and decrypts payment data")

	var retrievedEnvelope message.EncryptedEnvelope
	if err := ipfsStore.RetrieveJSON(cid, &retrievedEnvelope); err != nil {
		return fmt.Errorf("retrieve from IPFS: %w", err)
	}
	fmt.Printf("  Retrieved from CID: %s\n", cid)

	decrypted, err := message.DecryptAndVerify(&retrievedEnvelope, nonprofitKP.EncryptionPrivateKey, header.PrivateDataHash)
	if err != nil {
		return fmt.Errorf("decrypt and verify: %w", err)
	}

	fmt.Printf("  Decryption: SUCCESS\n")
	fmt.Printf("  Hash verification: PASSED\n")
	fmt.Printf("  Transactions: %d\n", len(decrypted.Transactions))

	for i, tx := range decrypted.Transactions {
		fmt.Printf("\n  Transaction %d:\n", i+1)
		fmt.Printf("    Amount: $%.2f %s\n", float64(tx.Amount)/100, tx.Currency)
		fmt.Printf("    Description: %s\n", tx.Description)
		if tx.Donation != nil {
			fmt.Printf("    Donation Type: %s\n", tx.Donation.Type)
			fmt.Printf("    Fund: %s (%s)\n", tx.Donation.FundName, tx.Donation.OrganizationName)
		}
		for _, donor := range tx.Donors {
			fmt.Printf("    Donor: %s <%s>\n", donor.Name, donor.Email)
		}
	}

	// -----------------------------------------------------------------------
	// Summary
	// -----------------------------------------------------------------------
	printHeader("LIVE DEMO COMPLETE (TEMPO)")
	fmt.Println(strings.Join([]string{
		"This live demo executed the full Agapay flow on Tempo testnet:",
		"",
		"  1. Chariot issued 4 Verifiable Credentials (Entity, ControlPerson, Org, Address)",
		"  2. DID Document generated with CNAME-delegated hosting (DNS simulated)",
		"  3. Chariot registered as issuer on Tempo (real transaction)",
		"  4. American Red Cross registered on-chain with VC hash (real transaction)",
		"  5. Payer verified the full Entity -> Org -> ControlPerson trust chain",
		"  6. ISO 20022-inspired payment message built and split (public/private)",
		"  7. Private donor PII encrypted and pinned to local IPFS (real CID)",
		"  8. TIP-20 payment sent via AgapayPaymentRouter on Tempo (real transaction)",
		"  9. Recipient retrieved encrypted data from IPFS and decrypted successfully",
		"",
		"All on-chain transactions are on the Tempo Moderato testnet.",
	}, "\n"))

	return nil
}

func runLive(programIDOverride, rpcOverride, wsOverride, ipfsOverride string) error {
	printHeader("AGAPAY LIVE DEVNET DEMO")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := loadConfig()
	if err != nil {
		cfg = &liveConfig{
			Chain:       "solana",
			ProgramID:   programIDOverride,
			RPCURL:      envOrDefault("SOLANA_RPC_URL", "https://api.devnet.solana.com"),
			WSURL:       envOrDefault("SOLANA_WS_URL", "wss://api.devnet.solana.com"),
			IPFSURL:     envOrDefault("IPFS_API_URL", "http://localhost:5001"),
			MintAddress: "",
		}
		if cfg.ProgramID == "" {
			return fmt.Errorf("no config found and no --program-id provided. Run 'agapay setup' first")
		}
	}
	if programIDOverride != "" {
		cfg.ProgramID = programIDOverride
	}
	if rpcOverride != "" {
		cfg.RPCURL = rpcOverride
	}
	if wsOverride != "" {
		cfg.WSURL = wsOverride
	}
	if ipfsOverride != "" {
		cfg.IPFSURL = ipfsOverride
	}

	progPubkey, err := solanago.PublicKeyFromBase58(cfg.ProgramID)
	if err != nil {
		return fmt.Errorf("invalid program ID: %w", err)
	}

	fmt.Printf("  Program ID: %s\n", progPubkey)
	fmt.Printf("  RPC URL:    %s\n", cfg.RPCURL)
	fmt.Printf("  IPFS URL:   %s\n", cfg.IPFSURL)

	regClient := solanaregistry.NewClient(cfg.RPCURL, cfg.WSURL, progPubkey, chain.SolanaDevnet)
	rpcClient := regClient.GetRPCClient()
	ipfsClient := ipfs.NewClient(cfg.IPFSURL)

	// -----------------------------------------------------------------------
	// Step 1: Generate keypairs
	// -----------------------------------------------------------------------
	printStep(1, "Generating keypairs")

	var authorityKey solanago.PrivateKey
	if cfg.AuthorityKeypath != "" {
		keyBytes, err := hex.DecodeString(cfg.AuthorityKeypath)
		if err != nil {
			return fmt.Errorf("decode authority key: %w", err)
		}
		authorityKey = solanago.PrivateKey(keyBytes)
	} else {
		return fmt.Errorf("no authority key in config. Run 'agapay setup' first")
	}
	fmt.Printf("  Authority (payer): %s\n", authorityKey.PublicKey())

	balance, _ := solanasettlement.GetSOLBalance(ctx, rpcClient, authorityKey.PublicKey())
	balanceSOL := float64(balance) / float64(solanago.LAMPORTS_PER_SOL)
	fmt.Printf("  Authority balance: %.4f SOL\n", balanceSOL)
	if balance < solanago.LAMPORTS_PER_SOL/2 {
		return fmt.Errorf("authority has insufficient SOL (%.4f). Fund via https://faucet.solana.com\n  Address: %s",
			balanceSOL, authorityKey.PublicKey())
	}

	issuerSolanaKey, err := solanago.NewRandomPrivateKey()
	if err != nil {
		return fmt.Errorf("generate issuer solana key: %w", err)
	}
	fmt.Printf("  Issuer Solana key: %s\n", issuerSolanaKey.PublicKey())

	fmt.Printf("  Transferring 0.2 SOL to issuer key...\n")
	if err := solanasettlement.TransferSOL(ctx, rpcClient, authorityKey, issuerSolanaKey.PublicKey(), solanago.LAMPORTS_PER_SOL/5, cfg.WSURL); err != nil {
		return fmt.Errorf("fund issuer: %w", err)
	}

	chariotKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate chariot keypair: %w", err)
	}
	chariotDID := "did:web:givechariot.com"
	fmt.Printf("  Chariot (VC Issuer) DID: %s\n", chariotDID)

	nonprofitKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate nonprofit keypair: %w", err)
	}
	nonprofitDomain := "agapay.redcross.org"
	nonprofitDID := "did:web:" + nonprofitDomain
	fmt.Printf("  Nonprofit (Holder) DID: %s\n", nonprofitDID)

	personKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate person keypair: %w", err)
	}
	personDID := did.CreateDIDKey(personKP.PublicKey)
	fmt.Printf("  Control Person DID: %s\n", personDID)

	payerKey, err := solanago.NewRandomPrivateKey()
	if err != nil {
		return fmt.Errorf("generate payer key: %w", err)
	}
	fmt.Printf("  Payer Solana key: %s\n", payerKey.PublicKey())

	fmt.Printf("  Transferring 0.1 SOL to payer...\n")
	if err := solanasettlement.TransferSOL(ctx, rpcClient, authorityKey, payerKey.PublicKey(), solanago.LAMPORTS_PER_SOL/10, cfg.WSURL); err != nil {
		return fmt.Errorf("fund payer: %w", err)
	}

	// -----------------------------------------------------------------------
	// Step 2: Issue Verifiable Credentials
	// -----------------------------------------------------------------------
	printStep(2, "Issuing Verifiable Credentials")

	vcIssuer, err := credential.NewIssuer(chariotDID, chariotKP.PrivateKey)
	if err != nil {
		return fmt.Errorf("create issuer: %w", err)
	}

	entityVC, err := vcIssuer.IssueNonprofitEntity(nonprofitDID, &credential.NonprofitEntityClaims{
		EIN:       "530196605",
		LegalName: "American National Red Cross",
		PhysicalAddress: credential.PostalAddress{
			Line1: "430 17th St NW", City: "Washington", State: "DC",
			PostalCode: "20006", Country: "US",
		},
		IRSSubsectionCode: "03",
		IRSPub78:          true,
		Incorporation: credential.Incorporation{
			Date: "1881-05-21", State: "DC", Jurisdiction: "federal",
		},
	})
	if err != nil {
		return fmt.Errorf("issue entity VC: %w", err)
	}
	fmt.Printf("  NonprofitEntityCredential: %s...\n", entityVC[:60])

	personVC, err := vcIssuer.IssueControlPerson(personDID, &credential.ControlPersonClaims{
		FullName:  "Gail McGovern",
		Title:     "President & CEO",
		Email:     "ceo@redcross.org",
		EntityDID: nonprofitDID,
		EntityEIN: "530196605",
		Role:      "officer",
	})
	if err != nil {
		return fmt.Errorf("issue control person VC: %w", err)
	}
	fmt.Printf("  ControlPersonCredential:   %s...\n", personVC[:60])

	orgVC, err := vcIssuer.IssueOrganization(nonprofitDID, &credential.OrganizationClaims{
		OrganizationName: "American Red Cross",
		Domain:           "redcross.org",
		EntityDID:        nonprofitDID,
		EntityEIN:        "530196605",
		Affiliation:      "independent_nonprofit",
		NTEECode:         "P50",
		MissionStatement: "Prevent and alleviate human suffering in the face of emergencies.",
	})
	if err != nil {
		return fmt.Errorf("issue org VC: %w", err)
	}
	fmt.Printf("  OrganizationCredential:    %s...\n", orgVC[:60])

	addrVC, err := vcIssuer.IssueAddress(nonprofitDID, &credential.AddressClaims{
		OrganizationDID:         nonprofitDID,
		OrganizationEIN:         "530196605",
		AddressType:             "solana_wallet",
		SupportedPaymentMethods: []string{"usdc"},
		SolanaWallet:            issuerSolanaKey.PublicKey().String(),
	})
	if err != nil {
		return fmt.Errorf("issue address VC: %w", err)
	}
	fmt.Printf("  AddressCredential:         %s...\n", addrVC[:60])

	// -----------------------------------------------------------------------
	// Step 3: DID Document
	// -----------------------------------------------------------------------
	printStep(3, "Setting up CNAME-delegated DID hosting (simulated DNS)")

	hosting := did.NewHostingService()
	hosting.RegisterExistingDID(nonprofitDomain, nonprofitKP,
		did.CreateDIDWebDocument(nonprofitDomain, nonprofitKP, "https://api.givechariot.com/v1/organizations/org_redcross"))
	doc, _ := hosting.GetDocument(nonprofitDomain)
	docJSON, _ := json.MarshalIndent(doc, "  ", "  ")
	fmt.Printf("  DNS CNAME: %s. CNAME dids.givechariot.com.\n", nonprofitDomain)
	fmt.Printf("  DID Document:\n  %s\n", string(docJSON))

	// -----------------------------------------------------------------------
	// Step 4: Register issuer on-chain (via Registry interface)
	// -----------------------------------------------------------------------
	printStep(4, "Registering Chariot as issuer on Solana devnet")

	issuerAddr := chain.Address(issuerSolanaKey.PublicKey().String())
	issuerAccount, err := regClient.GetIssuer(ctx, issuerAddr)
	if err == nil && issuerAccount.Active {
		fmt.Printf("  Issuer already registered on-chain (skipping)\n")
		fmt.Printf("  Issuer DID: %s\n", issuerAccount.DID)
		fmt.Printf("  Issuer active: %t\n", issuerAccount.Active)
	} else {
		txHash, err := regClient.RegisterIssuer(ctx, registry.RegisterIssuerParams{
			AuthoritySigner: []byte(authorityKey),
			IssuerAuthority: issuerAddr,
			DID:             chariotDID,
		})
		if err != nil {
			return fmt.Errorf("register issuer on-chain: %w", err)
		}
		fmt.Printf("  Transaction: %s\n", txHash)
		fmt.Printf("  Explorer:    https://explorer.solana.com/tx/%s?cluster=devnet\n", txHash)

		issuerAccount, err = regClient.GetIssuer(ctx, issuerAddr)
		if err != nil {
			return fmt.Errorf("read issuer account: %w", err)
		}
		fmt.Printf("  Issuer on-chain DID: %s\n", issuerAccount.DID)
		fmt.Printf("  Issuer active: %t\n", issuerAccount.Active)
	}

	// -----------------------------------------------------------------------
	// Step 5: Register organization on-chain (via Registry interface)
	// -----------------------------------------------------------------------
	printStep(5, "Registering nonprofit organization on Solana devnet")

	var recipientATA solanago.PublicKey
	if cfg.MintAddress != "" {
		mintPubkey, err := solanago.PublicKeyFromBase58(cfg.MintAddress)
		if err != nil {
			return fmt.Errorf("parse mint address: %w", err)
		}

		recipientATA, err = solanasettlement.CreateATA(
			ctx, rpcClient, authorityKey,
			issuerSolanaKey.PublicKey(), mintPubkey, cfg.WSURL,
		)
		if err != nil {
			return fmt.Errorf("create recipient ATA: %w", err)
		}
		fmt.Printf("  Recipient USDC ATA: %s\n", recipientATA)
	} else {
		recipientATA = issuerSolanaKey.PublicKey()
		fmt.Printf("  Recipient address: %s (no test mint configured)\n", recipientATA)
	}

	vcHash := registry.ComputeVCHash(entityVC, orgVC, addrVC)

	existingOrg, orgErr := regClient.GetOrganization(ctx, "530196605")
	if orgErr == nil && existingOrg.Active {
		fmt.Printf("  Organization already registered on-chain (skipping)\n")
		fmt.Printf("  EIN: %s, Name: %s\n", existingOrg.EIN, existingOrg.Name)
	} else {
		txHash, err := regClient.RegisterOrganization(ctx, registry.RegisterOrgParams{
			IssuerSigner:   solanago.PrivateKey(issuerSolanaKey),
			EIN:            "530196605",
			Name:           "American Red Cross",
			Domain:         "redcross.org",
			DIDURI:         nonprofitDID,
			VCHash:         vcHash,
			PaymentAddress: chain.Address(recipientATA.String()),
		})
		if err != nil {
			return fmt.Errorf("register organization on-chain: %w", err)
		}
		fmt.Printf("  Transaction: %s\n", txHash)
		fmt.Printf("  Explorer:    https://explorer.solana.com/tx/%s?cluster=devnet\n", txHash)
	}

	// -----------------------------------------------------------------------
	// Step 6: Read back from registry (via Registry interface)
	// -----------------------------------------------------------------------
	printStep(6, "Reading organization from on-chain registry")

	org, err := regClient.GetOrganization(ctx, "530196605")
	if err != nil {
		return fmt.Errorf("read organization: %w", err)
	}
	fmt.Printf("  EIN: %s\n", org.EIN)
	fmt.Printf("  Name: %s\n", org.Name)
	fmt.Printf("  Domain: %s\n", org.Domain)
	fmt.Printf("  DID: %s\n", org.DIDURI)
	fmt.Printf("  VC Hash: %s\n", hex.EncodeToString(org.VCHash[:]))
	fmt.Printf("  Payment Address: %s\n", org.PaymentAddress)
	fmt.Printf("  Active: %t\n", org.Active)

	// -----------------------------------------------------------------------
	// Step 7: Verify credential chain
	// -----------------------------------------------------------------------
	printStep(7, "Payer verifies full credential chain (Entity + Org + ControlPerson)")

	verifier := credential.NewVerifier()
	chainResult, err := verifier.VerifyControlPersonChain(
		entityVC, orgVC, personVC, chariotKP.PublicKey,
	)
	if err != nil {
		return fmt.Errorf("verify credential chain: %w", err)
	}
	if !chainResult.Valid {
		return fmt.Errorf("credential chain invalid: %s", chainResult.Error)
	}
	fmt.Printf("  Chain verification: VALID\n")
	fmt.Printf("  Entity DID: %s\n", chainResult.EntityDID)
	fmt.Printf("  Organization: %s\n", chainResult.OrganizationName)
	fmt.Printf("  Control Person: %s\n", chainResult.ControlPerson)
	fmt.Printf("  All 3 VCs signed by same issuer: %s\n", chainResult.EntityResult.IssuerDID)

	// -----------------------------------------------------------------------
	// Step 8: Build payment message
	// -----------------------------------------------------------------------
	printStep(8, "Building ISO 20022-inspired payment message")

	instruction := &message.AgapayPaymentInstruction{
		PublicHeader: message.PublicHeader{
			SenderDID:    "did:web:givechariot.com:payers:vanguard",
			RecipientDID: nonprofitDID,
			RecipientEIN: "530196605",
			PaymentType:  "donor_advised_fund_grant",
		},
		PrivateBody: message.PrivateBody{
			Transactions: []message.Transaction{
				{
					TransactionID: "tx_live_01",
					Amount:        50000,
					Currency:      "USD",
					Description:   "General Operating Support - Annual Grant",
					Donation: &message.Donation{
						Type:             "donor_advised_fund_grant",
						OrganizationName: "Vanguard Charitable",
						FundName:         "John Doe Giving Fund",
						Purpose:          "General Operating Support",
						Note:             "Annual grant from the Doe family",
					},
					Donors: []message.Donor{
						{
							Name:  "John Doe",
							Email: "john.doe@example.com",
							Phone: "415-555-1212",
							Address: &message.PostalAddress{
								Line1: "123 Main St", City: "San Francisco",
								State: "CA", PostalCode: "94105", Country: "US",
							},
						},
					},
				},
			},
			SenderReference: "VNG-2026-00142",
			RemittanceInfo:  "DAF Grant - Q1 2026 Distribution",
		},
	}

	header, body, err := message.Split(instruction)
	if err != nil {
		return fmt.Errorf("split payment instruction: %w", err)
	}
	fmt.Printf("  Message ID: %s\n", header.MessageID)
	fmt.Printf("  Total Amount: $%.2f (USD)\n", float64(header.TotalAmount)/100)
	fmt.Printf("  Payment Type: %s\n", header.PaymentType)
	fmt.Printf("  Private Data Hash: %s\n", header.PrivateDataHash)

	// -----------------------------------------------------------------------
	// Step 9: Encrypt and pin to IPFS
	// -----------------------------------------------------------------------
	printStep(9, "Encrypting private body and pinning to IPFS")

	envelope, err := message.Encrypt(body, nonprofitKP.EncryptionPublicKey)
	if err != nil {
		return fmt.Errorf("encrypt private body: %w", err)
	}
	fmt.Printf("  Encryption: %s\n", envelope.Algorithm)

	cid, err := ipfsClient.PinJSON(envelope)
	if err != nil {
		return fmt.Errorf("pin to IPFS: %w\nIs the IPFS daemon running? ('ipfs daemon')", err)
	}
	header.PrivateDataCID = cid
	fmt.Printf("  IPFS CID: %s\n", cid)
	fmt.Printf("  IPFS Gateway: https://ipfs.io/ipfs/%s\n", cid)

	// -----------------------------------------------------------------------
	// Step 10: Send payment (via Settler interface)
	// -----------------------------------------------------------------------
	printStep(10, "Sending USDC payment with SPL Memo on devnet")

	if cfg.MintAddress != "" {
		mintPubkey, _ := solanago.PublicKeyFromBase58(cfg.MintAddress)

		payerATA, err := solanasettlement.CreateATA(
			ctx, rpcClient, authorityKey,
			payerKey.PublicKey(), mintPubkey, cfg.WSURL,
		)
		if err != nil {
			return fmt.Errorf("create payer ATA: %w", err)
		}
		fmt.Printf("  Payer USDC ATA: %s\n", payerATA)

		usdcAmount := settlement.CentsToUSDCUnits(header.TotalAmount)
		fmt.Printf("  Minting %d test USDC units ($%.2f) to payer...\n",
			usdcAmount, float64(header.TotalAmount)/100)
		if err := solanasettlement.MintTestTokens(
			ctx, rpcClient, authorityKey,
			mintPubkey, payerATA, usdcAmount, cfg.WSURL,
		); err != nil {
			return fmt.Errorf("mint test tokens: %w", err)
		}

		headerJSON, _ := json.Marshal(header)

		paymentClient := solanasettlement.NewPaymentClient(cfg.RPCURL, cfg.WSURL, mintPubkey, chain.SolanaDevnet)
		result, err := paymentClient.SendPayment(ctx, &settlement.PaymentParams{
			SignerKey:  solanago.PrivateKey(payerKey),
			Recipient: chain.Address(recipientATA.String()),
			Amount:    usdcAmount,
			MemoData:  headerJSON,
		})
		if err != nil {
			return fmt.Errorf("send payment: %w", err)
		}

		fmt.Printf("  Payment Transaction: %s\n", result.TxHash)
		fmt.Printf("  Explorer: https://explorer.solana.com/tx/%s?cluster=devnet\n", result.TxHash)
		fmt.Printf("  Amount: %d USDC units ($%.2f)\n", result.Amount, float64(header.TotalAmount)/100)
	} else {
		fmt.Printf("  (Skipped: no test mint configured. Run 'agapay setup' to create one.)\n")
		headerJSON, _ := json.MarshalIndent(header, "  ", "  ")
		fmt.Printf("  SPL Memo (would be on-chain):\n  %s\n", string(headerJSON))
	}

	// -----------------------------------------------------------------------
	// Step 11: Decrypt
	// -----------------------------------------------------------------------
	printStep(11, "Recipient retrieves and decrypts payment data from IPFS")

	var retrievedEnvelope message.EncryptedEnvelope
	if err := ipfsClient.RetrieveJSON(cid, &retrievedEnvelope); err != nil {
		return fmt.Errorf("retrieve from IPFS: %w", err)
	}
	fmt.Printf("  Retrieved from IPFS CID: %s\n", cid)

	decrypted, err := message.DecryptAndVerify(&retrievedEnvelope, nonprofitKP.EncryptionPrivateKey, header.PrivateDataHash)
	if err != nil {
		return fmt.Errorf("decrypt and verify: %w", err)
	}

	fmt.Printf("  Decryption: SUCCESS\n")
	fmt.Printf("  Hash verification: PASSED\n")
	fmt.Printf("  Transactions: %d\n", len(decrypted.Transactions))

	for i, tx := range decrypted.Transactions {
		fmt.Printf("\n  Transaction %d:\n", i+1)
		fmt.Printf("    Amount: $%.2f %s\n", float64(tx.Amount)/100, tx.Currency)
		fmt.Printf("    Description: %s\n", tx.Description)
		if tx.Donation != nil {
			fmt.Printf("    Donation Type: %s\n", tx.Donation.Type)
			fmt.Printf("    Fund: %s (%s)\n", tx.Donation.FundName, tx.Donation.OrganizationName)
			fmt.Printf("    Purpose: %s\n", tx.Donation.Purpose)
		}
		for _, donor := range tx.Donors {
			fmt.Printf("    Donor: %s <%s>\n", donor.Name, donor.Email)
			if donor.Address != nil {
				fmt.Printf("    Address: %s, %s, %s %s\n",
					donor.Address.Line1, donor.Address.City,
					donor.Address.State, donor.Address.PostalCode)
			}
		}
	}
	fmt.Printf("  Sender Reference: %s\n", decrypted.SenderReference)
	fmt.Printf("  Remittance Info: %s\n", decrypted.RemittanceInfo)

	// -----------------------------------------------------------------------
	// Summary
	// -----------------------------------------------------------------------
	printHeader("LIVE DEMO COMPLETE")

	lines := []string{
		"This live demo executed the full Agapay flow on real infrastructure:",
		"",
		"  1. Chariot issued 4 Verifiable Credentials (Entity, ControlPerson, Org, Address)",
		"  2. DID Document generated with CNAME-delegated hosting (DNS simulated)",
		"  3. Chariot registered as issuer on Solana devnet (real transaction)",
		"  4. American Red Cross registered on-chain with VC hash (real transaction)",
		"  5. Payer verified the full Entity -> Org -> ControlPerson trust chain",
		"  6. ISO 20022-inspired payment message built and split (public/private)",
		"  7. Private donor PII encrypted and pinned to local IPFS (real CID)",
	}
	if cfg.MintAddress != "" {
		lines = append(lines, "  8. USDC payment sent on Solana devnet with SPL Memo (real transaction)")
	} else {
		lines = append(lines, "  8. USDC payment skipped (no test mint)")
	}
	lines = append(lines,
		"  9. Recipient retrieved encrypted data from IPFS and decrypted successfully",
		"",
		"All on-chain transactions are verifiable on Solana Explorer (devnet).",
	)

	fmt.Println(strings.Join(lines, "\n"))
	return nil
}
