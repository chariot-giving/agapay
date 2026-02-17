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

	"github.com/chariot-giving/agapay/pkg/credential"
	"github.com/chariot-giving/agapay/pkg/did"
	"github.com/chariot-giving/agapay/pkg/ipfs"
	"github.com/chariot-giving/agapay/pkg/message"
	"github.com/chariot-giving/agapay/pkg/registry"
	"github.com/chariot-giving/agapay/pkg/settlement"
	"github.com/gagliardetto/solana-go"
	"github.com/spf13/cobra"
)

// liveConfig holds the saved configuration from setup.
type liveConfig struct {
	ProgramID        string `json:"program_id"`
	RPCURL           string `json:"rpc_url"`
	WSURL            string `json:"ws_url"`
	IPFSURL          string `json:"ipfs_url"`
	MintAddress      string `json:"mint_address"`
	AuthorityKeypath string `json:"authority_keypath"`
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

// envOrDefault returns the environment variable value or a default.
func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func setupCmd() *cobra.Command {
	var programID, rpcURL, wsURL, ipfsURL string

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Provision devnet infrastructure for the live demo",
		Long: `Sets up all required devnet infrastructure:
1. Checks prerequisites (Solana CLI, IPFS daemon)
2. Generates Solana keypairs for authority, issuer, and payer
3. Airdrops devnet SOL to all accounts
4. Creates a test USDC mint (6 decimals)
5. Saves configuration to ~/.agapay/config.json

Requires: deployed Agapay Registry program (use 'anchor deploy')`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(programID, rpcURL, wsURL, ipfsURL)
		},
	}

	cmd.Flags().StringVar(&programID, "program-id", "", "Deployed Agapay Registry program ID (required)")
	cmd.Flags().StringVar(&rpcURL, "rpc", envOrDefault("SOLANA_RPC_URL", "https://api.devnet.solana.com"), "Solana RPC URL")
	cmd.Flags().StringVar(&wsURL, "ws", envOrDefault("SOLANA_WS_URL", "wss://api.devnet.solana.com"), "Solana WebSocket URL")
	cmd.Flags().StringVar(&ipfsURL, "ipfs", envOrDefault("IPFS_API_URL", "http://localhost:5001"), "IPFS API URL")
	cmd.MarkFlagRequired("program-id")

	return cmd
}

func runSetup(programID, rpcURL, wsURL, ipfsURL string) error {
	printHeader("AGAPAY DEVNET SETUP")
	ctx := context.Background()

	// Validate program ID
	progPubkey, err := solana.PublicKeyFromBase58(programID)
	if err != nil {
		return fmt.Errorf("invalid program ID: %w", err)
	}
	fmt.Printf("  Program ID: %s\n", progPubkey)
	fmt.Printf("  RPC URL:    %s\n", rpcURL)
	fmt.Printf("  WS URL:     %s\n", wsURL)
	fmt.Printf("  IPFS URL:   %s\n", ipfsURL)

	// Check IPFS daemon
	printStep(1, "Checking prerequisites")
	resp, err := http.Post(ipfsURL+"/api/v0/id", "", nil)
	if err != nil {
		return fmt.Errorf("IPFS daemon not reachable at %s: %w\nRun 'ipfs daemon' first", ipfsURL, err)
	}
	resp.Body.Close()
	fmt.Printf("  IPFS daemon: OK (%s)\n", ipfsURL)

	rpcClient := settlement.NewPaymentClient(rpcURL, wsURL).GetRPCClient()

	// Load or generate authority keypair (persisted so re-runs reuse the same key)
	printStep(2, "Loading or generating Solana keypair")
	var authorityKey solana.PrivateKey
	existingCfg, cfgErr := loadConfig()
	if cfgErr == nil && existingCfg.AuthorityKeypath != "" {
		keyBytes, err := hex.DecodeString(existingCfg.AuthorityKeypath)
		if err == nil && len(keyBytes) == 64 {
			authorityKey = solana.PrivateKey(keyBytes)
			fmt.Printf("  Reusing existing authority: %s\n", authorityKey.PublicKey())
		}
	}
	if authorityKey == nil {
		ak, err := solana.NewRandomPrivateKey()
		if err != nil {
			return fmt.Errorf("generate authority key: %w", err)
		}
		authorityKey = solana.PrivateKey(ak)
		fmt.Printf("  Generated new authority: %s\n", authorityKey.PublicKey())
	}

	// Save config early so the keypair is persisted even if airdrop fails
	cfg := &liveConfig{
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

	// Check balance and airdrop if needed
	printStep(3, "Checking SOL balance / airdropping")
	balance, err := settlement.GetSOLBalance(ctx, rpcClient, authorityKey.PublicKey())
	if err != nil {
		fmt.Printf("  Could not check balance: %v\n", err)
		balance = 0
	}
	balanceSOL := float64(balance) / float64(solana.LAMPORTS_PER_SOL)
	fmt.Printf("  Current balance: %.4f SOL\n", balanceSOL)

	if balance < solana.LAMPORTS_PER_SOL {
		fmt.Printf("  Requesting airdrop of 2 SOL...\n")
		if err := settlement.AirdropSOL(ctx, rpcClient, authorityKey.PublicKey(), 2*solana.LAMPORTS_PER_SOL); err != nil {
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

	// Create test USDC mint (skip if already created)
	if cfg.MintAddress != "" {
		printStep(4, "Test USDC mint already exists")
		fmt.Printf("  Mint: %s\n", cfg.MintAddress)
	} else {
		printStep(4, "Creating test USDC mint (6 decimals)")
		mintPubkey, _, err := settlement.CreateTestMint(ctx, rpcClient, authorityKey, 6, wsURL)
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
	var programID, rpcURL, wsURL, ipfsURL string

	cmd := &cobra.Command{
		Use:   "live",
		Short: "Run the full E2E demo on Solana devnet with real IPFS",
		Long: `Executes the complete Agapay flow against real infrastructure:
1. Issue all 4 Verifiable Credentials
2. Register Chariot as issuer on Solana devnet
3. Register nonprofit organization on-chain
4. Verify full credential chain (Entity + Org + ControlPerson)
5. Build ISO 20022-inspired payment, encrypt PII
6. Pin encrypted data to local IPFS (real CID)
7. Send USDC transfer with SPL Memo on devnet
8. Retrieve and decrypt payment data from IPFS

Requires: 'agapay setup' to have been run first (or pass --program-id)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLive(programID, rpcURL, wsURL, ipfsURL)
		},
	}

	cmd.Flags().StringVar(&programID, "program-id", "", "Override program ID from config")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "Override Solana RPC URL")
	cmd.Flags().StringVar(&wsURL, "ws", "", "Override Solana WebSocket URL")
	cmd.Flags().StringVar(&ipfsURL, "ipfs", "", "Override IPFS API URL")

	return cmd
}

func runLive(programIDOverride, rpcOverride, wsOverride, ipfsOverride string) error {
	printHeader("AGAPAY LIVE DEVNET DEMO")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Load config (with overrides)
	cfg, err := loadConfig()
	if err != nil {
		// Try to build config from flags/env
		cfg = &liveConfig{
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

	progPubkey, err := solana.PublicKeyFromBase58(cfg.ProgramID)
	if err != nil {
		return fmt.Errorf("invalid program ID: %w", err)
	}

	fmt.Printf("  Program ID: %s\n", progPubkey)
	fmt.Printf("  RPC URL:    %s\n", cfg.RPCURL)
	fmt.Printf("  IPFS URL:   %s\n", cfg.IPFSURL)

	// Initialize clients
	regClient := registry.NewClient(cfg.RPCURL, progPubkey)
	rpcClient := regClient.GetRPCClient()
	ipfsClient := ipfs.NewClient(cfg.IPFSURL)

	// -----------------------------------------------------------------------
	// Step 1: Generate keypairs
	// -----------------------------------------------------------------------
	printStep(1, "Generating keypairs")

	// Authority keypair (pre-funded via 'agapay setup' or web faucet -- no airdrops needed)
	var authorityKey solana.PrivateKey
	if cfg.AuthorityKeypath != "" {
		keyBytes, err := hex.DecodeString(cfg.AuthorityKeypath)
		if err != nil {
			return fmt.Errorf("decode authority key: %w", err)
		}
		authorityKey = solana.PrivateKey(keyBytes)
	} else {
		return fmt.Errorf("no authority key in config. Run 'agapay setup' first")
	}
	fmt.Printf("  Authority (payer): %s\n", authorityKey.PublicKey())

	// Check authority balance
	balance, _ := settlement.GetSOLBalance(ctx, rpcClient, authorityKey.PublicKey())
	balanceSOL := float64(balance) / float64(solana.LAMPORTS_PER_SOL)
	fmt.Printf("  Authority balance: %.4f SOL\n", balanceSOL)
	if balance < solana.LAMPORTS_PER_SOL/2 {
		return fmt.Errorf("authority has insufficient SOL (%.4f). Fund via https://faucet.solana.com\n  Address: %s",
			balanceSOL, authorityKey.PublicKey())
	}

	// Issuer keypair for Solana (separate from authority to match Anchor program design)
	issuerSolanaKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return fmt.Errorf("generate issuer solana key: %w", err)
	}
	fmt.Printf("  Issuer Solana key: %s\n", issuerSolanaKey.PublicKey())

	// Fund issuer key with SOL transfer from authority (no airdrop needed)
	fmt.Printf("  Transferring 0.2 SOL to issuer key...\n")
	if err := settlement.TransferSOL(ctx, rpcClient, authorityKey, issuerSolanaKey.PublicKey(), solana.LAMPORTS_PER_SOL/5, cfg.WSURL); err != nil {
		return fmt.Errorf("fund issuer: %w", err)
	}

	// Chariot identity keypair (for VC signing)
	chariotKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate chariot keypair: %w", err)
	}
	chariotDID := "did:web:givechariot.com"
	fmt.Printf("  Chariot (VC Issuer) DID: %s\n", chariotDID)

	// Nonprofit identity keypair
	nonprofitKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate nonprofit keypair: %w", err)
	}
	nonprofitDomain := "agapay.redcross.org"
	nonprofitDID := "did:web:" + nonprofitDomain
	fmt.Printf("  Nonprofit (Holder) DID: %s\n", nonprofitDID)

	// Control person keypair
	personKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate person keypair: %w", err)
	}
	personDID := did.CreateDIDKey(personKP.PublicKey)
	fmt.Printf("  Control Person DID: %s\n", personDID)

	// Payer keypair (for USDC transfer)
	payerKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return fmt.Errorf("generate payer key: %w", err)
	}
	fmt.Printf("  Payer Solana key: %s\n", payerKey.PublicKey())

	// Fund payer key with SOL transfer from authority (no airdrop needed)
	fmt.Printf("  Transferring 0.1 SOL to payer...\n")
	if err := settlement.TransferSOL(ctx, rpcClient, authorityKey, payerKey.PublicKey(), solana.LAMPORTS_PER_SOL/10, cfg.WSURL); err != nil {
		return fmt.Errorf("fund payer: %w", err)
	}

	// -----------------------------------------------------------------------
	// Step 2: Issue Verifiable Credentials
	// -----------------------------------------------------------------------
	printStep(2, "Issuing Verifiable Credentials")

	issuer, err := credential.NewIssuer(chariotDID, chariotKP.PrivateKey)
	if err != nil {
		return fmt.Errorf("create issuer: %w", err)
	}

	entityVC, err := issuer.IssueNonprofitEntity(nonprofitDID, &credential.NonprofitEntityClaims{
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

	personVC, err := issuer.IssueControlPerson(personDID, &credential.ControlPersonClaims{
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

	orgVC, err := issuer.IssueOrganization(nonprofitDID, &credential.OrganizationClaims{
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

	addrVC, err := issuer.IssueAddress(nonprofitDID, &credential.AddressClaims{
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
	// Step 3: Generate DID Document (CNAME-delegated hosting simulation)
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
	// Step 4: Register Chariot as issuer on Solana devnet (REAL)
	// -----------------------------------------------------------------------
	printStep(4, "Registering Chariot as issuer on Solana devnet")

	// Check if issuer already exists (idempotent re-runs)
	issuerAccount, err := regClient.GetIssuer(ctx, issuerSolanaKey.PublicKey())
	if err == nil && issuerAccount.Active {
		fmt.Printf("  Issuer already registered on-chain (skipping)\n")
		fmt.Printf("  Issuer DID: %s\n", issuerAccount.DID)
		fmt.Printf("  Issuer active: %t\n", issuerAccount.Active)
	} else {
		sig, err := regClient.SendRegisterIssuer(
			ctx,
			authorityKey,
			issuerSolanaKey.PublicKey(),
			chariotDID,
			cfg.WSURL,
		)
		if err != nil {
			return fmt.Errorf("register issuer on-chain: %w", err)
		}
		fmt.Printf("  Transaction: %s\n", sig)
		fmt.Printf("  Explorer:    https://explorer.solana.com/tx/%s?cluster=devnet\n", sig)

		issuerAccount, err = regClient.GetIssuer(ctx, issuerSolanaKey.PublicKey())
		if err != nil {
			return fmt.Errorf("read issuer account: %w", err)
		}
		fmt.Printf("  Issuer on-chain DID: %s\n", issuerAccount.DID)
		fmt.Printf("  Issuer active: %t\n", issuerAccount.Active)
	}

	// -----------------------------------------------------------------------
	// Step 5: Register nonprofit organization on-chain (REAL)
	// -----------------------------------------------------------------------
	printStep(5, "Registering nonprofit organization on Solana devnet")

	// Create recipient USDC ATA first (needed for the org registration)
	var recipientATA solana.PublicKey
	if cfg.MintAddress != "" {
		mintPubkey, err := solana.PublicKeyFromBase58(cfg.MintAddress)
		if err != nil {
			return fmt.Errorf("parse mint address: %w", err)
		}

		recipientATA, err = settlement.CreateATA(
			ctx, rpcClient, authorityKey,
			issuerSolanaKey.PublicKey(), mintPubkey, cfg.WSURL,
		)
		if err != nil {
			return fmt.Errorf("create recipient ATA: %w", err)
		}
		fmt.Printf("  Recipient USDC ATA: %s\n", recipientATA)
	} else {
		// Use a placeholder if no mint configured
		recipientATA = issuerSolanaKey.PublicKey()
		fmt.Printf("  Recipient address: %s (no test mint configured)\n", recipientATA)
	}

	vcHash := registry.ComputeVCHash(entityVC, orgVC, addrVC)

	// Check if organization already exists (idempotent re-runs)
	existingOrg, orgErr := regClient.GetOrganization(ctx, "530196605")
	if orgErr == nil && existingOrg.Active {
		fmt.Printf("  Organization already registered on-chain (skipping)\n")
		fmt.Printf("  EIN: %s, Name: %s\n", existingOrg.EIN, existingOrg.Name)
	} else {
		regSig, err := regClient.SendRegisterOrganization(
			ctx,
			solana.PrivateKey(issuerSolanaKey),
			"530196605",
			"American Red Cross",
			"redcross.org",
			nonprofitDID,
			vcHash,
			recipientATA,
			cfg.WSURL,
		)
		if err != nil {
			return fmt.Errorf("register organization on-chain: %w", err)
		}
		fmt.Printf("  Transaction: %s\n", regSig)
		fmt.Printf("  Explorer:    https://explorer.solana.com/tx/%s?cluster=devnet\n", regSig)
	}

	// -----------------------------------------------------------------------
	// Step 6: Read back organization from registry (REAL)
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
	fmt.Printf("  Active: %t\n", org.Active)

	// -----------------------------------------------------------------------
	// Step 7: Payer verifies full credential chain
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
	// Step 8: Build ISO 20022-inspired payment message
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
	// Step 9: Encrypt PII and pin to IPFS (REAL)
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
	// Step 10: Send USDC payment with SPL Memo on devnet (REAL)
	// -----------------------------------------------------------------------
	printStep(10, "Sending USDC payment with SPL Memo on devnet")

	if cfg.MintAddress != "" {
		mintPubkey, _ := solana.PublicKeyFromBase58(cfg.MintAddress)

		// Create payer ATA and mint test tokens
		payerATA, err := settlement.CreateATA(
			ctx, rpcClient, authorityKey,
			payerKey.PublicKey(), mintPubkey, cfg.WSURL,
		)
		if err != nil {
			return fmt.Errorf("create payer ATA: %w", err)
		}
		fmt.Printf("  Payer USDC ATA: %s\n", payerATA)

		// Mint test USDC to payer (authority is mint authority)
		usdcAmount := settlement.CentsToUSDCUnits(header.TotalAmount)
		fmt.Printf("  Minting %d test USDC units ($%.2f) to payer...\n",
			usdcAmount, float64(header.TotalAmount)/100)
		if err := settlement.MintTestTokens(
			ctx, rpcClient, authorityKey,
			mintPubkey, payerATA, usdcAmount, cfg.WSURL,
		); err != nil {
			return fmt.Errorf("mint test tokens: %w", err)
		}

		// Build memo from public header
		headerJSON, _ := json.Marshal(header)

		// Send real USDC transfer
		paymentClient := settlement.NewPaymentClient(cfg.RPCURL, cfg.WSURL)
		result, err := paymentClient.SendPayment(ctx, &settlement.PaymentParams{
			PayerWallet:           solana.PrivateKey(payerKey),
			PayerTokenAccount:     payerATA,
			RecipientTokenAccount: recipientATA,
			Amount:                usdcAmount,
			MemoData:              headerJSON,
		})
		if err != nil {
			return fmt.Errorf("send payment: %w", err)
		}

		fmt.Printf("  Payment Transaction: %s\n", result.Signature)
		fmt.Printf("  Explorer: https://explorer.solana.com/tx/%s?cluster=devnet\n", result.Signature)
		fmt.Printf("  Amount: %d USDC units ($%.2f)\n", result.Amount, float64(header.TotalAmount)/100)
	} else {
		fmt.Printf("  (Skipped: no test mint configured. Run 'agapay setup' to create one.)\n")
		headerJSON, _ := json.MarshalIndent(header, "  ", "  ")
		fmt.Printf("  SPL Memo (would be on-chain):\n  %s\n", string(headerJSON))
	}

	// -----------------------------------------------------------------------
	// Step 11: Recipient retrieves and decrypts payment data from IPFS (REAL)
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
