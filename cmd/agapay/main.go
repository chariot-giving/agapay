// Command agapay is the CLI for the Agapay charitable payment network PoC.
// It demonstrates the full E2E flow: VC issuance, DID hosting, on-chain
// registry, privacy-preserving payment messages, and USDC settlement.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chariot-giving/agapay/pkg/credential"
	"github.com/chariot-giving/agapay/pkg/did"
	"github.com/chariot-giving/agapay/pkg/ipfs"
	"github.com/chariot-giving/agapay/pkg/message"
	"github.com/chariot-giving/agapay/pkg/registry"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "agapay",
		Short: "Agapay - Open Charitable Payment Network CLI",
		Long: `Agapay is a proof of concept for an open-source charitable identity
and payment network facilitated by verifiable credentials, an on-chain
nonprofit registry, and privacy-preserving stablecoin settlement.`,
	}

	rootCmd.AddCommand(demoCmd())
	rootCmd.AddCommand(setupCmd())
	rootCmd.AddCommand(liveCmd())
	rootCmd.AddCommand(issueCmd())
	rootCmd.AddCommand(verifyCmd())
	rootCmd.AddCommand(lookupCmd())
	rootCmd.AddCommand(payCmd())
	rootCmd.AddCommand(decryptCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func demoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "demo",
		Short: "Run the full E2E demo flow",
		Long: `Demonstrates the complete Agapay flow:
1. Generate keypairs for Chariot (issuer), nonprofit (holder), and payer
2. Issue Verifiable Credentials for a sample nonprofit
3. Generate DID Document with CNAME-delegated hosting
4. Simulate on-chain registry registration
5. Build an ISO 20022-inspired payment message
6. Encrypt donor PII and pin to IPFS (mock)
7. Simulate USDC payment with public header
8. Decrypt and verify payment data as the recipient`,
		RunE: runDemo,
	}
}

func runDemo(cmd *cobra.Command, args []string) error {
	printHeader("AGAPAY PROOF OF CONCEPT - FULL E2E DEMO")

	// -----------------------------------------------------------------------
	// Step 1: Generate Keypairs
	// -----------------------------------------------------------------------
	printStep(1, "Generating keypairs")

	chariotKP, err := did.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generate chariot keypair: %w", err)
	}
	chariotDID := "did:web:givechariot.com"
	fmt.Printf("  Chariot (Issuer)  DID: %s\n", chariotDID)

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

	fmt.Printf("  Payer DID: did:web:givechariot.com:payers:vanguard\n")

	// -----------------------------------------------------------------------
	// Step 2: Issue Verifiable Credentials
	// -----------------------------------------------------------------------
	printStep(2, "Issuing Verifiable Credentials")

	issuer, err := credential.NewIssuer(chariotDID, chariotKP.PrivateKey)
	if err != nil {
		return fmt.Errorf("create issuer: %w", err)
	}

	// Entity VC
	entityVC, err := issuer.IssueNonprofitEntity(nonprofitDID, &credential.NonprofitEntityClaims{
		EIN:       "530196605",
		LegalName: "American National Red Cross",
		PhysicalAddress: credential.PostalAddress{
			Line1:      "430 17th St NW",
			City:       "Washington",
			State:      "DC",
			PostalCode: "20006",
			Country:    "US",
		},
		IRSSubsectionCode: "03",
		IRSPub78:          true,
		IRSRevocation:     false,
		OFACList:          false,
		Incorporation: credential.Incorporation{
			Date:         "1881-05-21",
			State:        "DC",
			Jurisdiction: "federal",
		},
	})
	if err != nil {
		return fmt.Errorf("issue entity VC: %w", err)
	}
	fmt.Printf("  NonprofitEntityCredential: %s...\n", entityVC[:80])

	// Control Person VC
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
	fmt.Printf("  ControlPersonCredential:   %s...\n", personVC[:80])

	// Organization VC
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
		return fmt.Errorf("issue organization VC: %w", err)
	}
	fmt.Printf("  OrganizationCredential:    %s...\n", orgVC[:80])

	// Address VC
	addrVC, err := issuer.IssueAddress(nonprofitDID, &credential.AddressClaims{
		OrganizationDID:         nonprofitDID,
		OrganizationEIN:         "530196605",
		AddressType:             "solana_wallet",
		SupportedPaymentMethods: []string{"usdc"},
		SolanaWallet:            "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
	})
	if err != nil {
		return fmt.Errorf("issue address VC: %w", err)
	}
	fmt.Printf("  AddressCredential:         %s...\n", addrVC[:80])

	// -----------------------------------------------------------------------
	// Step 3: Generate DID Document (CNAME-delegated hosting)
	// -----------------------------------------------------------------------
	printStep(3, "Setting up CNAME-delegated DID hosting")

	hosting := did.NewHostingService()
	hosting.RegisterExistingDID(nonprofitDomain, nonprofitKP,
		did.CreateDIDWebDocument(nonprofitDomain, nonprofitKP, "https://api.givechariot.com/v1/organizations/org_redcross"))

	doc, _ := hosting.GetDocument(nonprofitDomain)
	docJSON, _ := json.MarshalIndent(doc, "  ", "  ")
	fmt.Printf("  DNS CNAME: %s. CNAME dids.givechariot.com.\n", nonprofitDomain)
	fmt.Printf("  DID Document (served at https://%s/.well-known/did.json):\n", nonprofitDomain)
	fmt.Printf("  %s\n", string(docJSON))

	// -----------------------------------------------------------------------
	// Step 4: Simulate on-chain registry registration
	// -----------------------------------------------------------------------
	printStep(4, "Registering on the Solana on-chain registry (simulated)")

	vcHash := registry.ComputeVCHash(entityVC, orgVC, addrVC)
	fmt.Printf("  EIN: 530196605\n")
	fmt.Printf("  Organization: American Red Cross\n")
	fmt.Printf("  Domain: redcross.org\n")
	fmt.Printf("  DID: %s\n", nonprofitDID)
	fmt.Printf("  VC Hash: %s\n", hex.EncodeToString(vcHash[:]))
	fmt.Printf("  USDC Address: 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU\n")
	fmt.Printf("  Status: Active\n")
	fmt.Printf("  (In production, this would be a Solana transaction creating an OrganizationAccount PDA)\n")

	// -----------------------------------------------------------------------
	// Step 5: Verify a VC (payer side)
	// -----------------------------------------------------------------------
	printStep(5, "Payer verifies the organization's credentials")

	verifier := credential.NewVerifier()
	result, err := verifier.Verify(orgVC, chariotKP.PublicKey)
	if err != nil {
		return fmt.Errorf("verify VC: %w", err)
	}
	fmt.Printf("  Verification result: Valid=%t\n", result.Valid)
	fmt.Printf("  Issuer: %s\n", result.IssuerDID)
	fmt.Printf("  Subject: %s\n", result.SubjectDID)
	fmt.Printf("  Credential type: %s\n", strings.Join(result.CredentialType, ", "))

	// -----------------------------------------------------------------------
	// Step 6: Build ISO 20022-inspired payment message
	// -----------------------------------------------------------------------
	printStep(6, "Building ISO 20022-inspired payment message")

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
					TransactionID: "tx_01",
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
								Line1:      "123 Main St",
								City:       "San Francisco",
								State:      "CA",
								PostalCode: "94105",
								Country:    "US",
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
	fmt.Printf("  Transactions: %d\n", header.NumberOfTransactions)

	// -----------------------------------------------------------------------
	// Step 7: Encrypt PII and pin to IPFS
	// -----------------------------------------------------------------------
	printStep(7, "Encrypting private body and pinning to IPFS")

	envelope, err := message.Encrypt(body, nonprofitKP.EncryptionPublicKey)
	if err != nil {
		return fmt.Errorf("encrypt private body: %w", err)
	}
	fmt.Printf("  Encryption algorithm: %s\n", envelope.Algorithm)
	fmt.Printf("  Ephemeral public key: %s...\n", envelope.EphemeralPublicKey[:40])

	// Pin to mock IPFS
	mockIPFS := ipfs.NewMockClient()
	cid, err := mockIPFS.PinJSON(envelope)
	if err != nil {
		return fmt.Errorf("pin to IPFS: %w", err)
	}
	header.PrivateDataCID = cid
	fmt.Printf("  IPFS CID: %s\n", cid)

	// -----------------------------------------------------------------------
	// Step 8: Simulate USDC payment with public header
	// -----------------------------------------------------------------------
	printStep(8, "Sending USDC payment with public header (simulated)")

	headerJSON, _ := json.MarshalIndent(header, "  ", "  ")
	fmt.Printf("  SPL Memo (public header on-chain):\n  %s\n", string(headerJSON))
	fmt.Printf("  USDC Amount: %d units ($%.2f)\n",
		uint64(header.TotalAmount)*10000,
		float64(header.TotalAmount)/100)
	fmt.Printf("  From: Payer USDC ATA\n")
	fmt.Printf("  To: 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU\n")
	fmt.Printf("  (In production, this would be a Solana transaction with SPL Token transfer + Memo)\n")

	// -----------------------------------------------------------------------
	// Step 9: Recipient decrypts payment data
	// -----------------------------------------------------------------------
	printStep(9, "Recipient retrieves and decrypts payment data")

	// Retrieve from mock IPFS
	var retrievedEnvelope message.EncryptedEnvelope
	if err := mockIPFS.RetrieveJSON(cid, &retrievedEnvelope); err != nil {
		return fmt.Errorf("retrieve from IPFS: %w", err)
	}

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
	printHeader("DEMO COMPLETE")
	fmt.Println(`
This demo showed the full Agapay flow:

  1. Chariot (Service Provider) issued 4 Verifiable Credentials:
     - NonprofitEntityCredential (EIN + compliance)
     - ControlPersonCredential (KYC on officers)
     - OrganizationCredential (organization identity)
     - AddressCredential (payment address)

  2. A DID Document was generated for the nonprofit using
     CNAME-delegated hosting (agapay.redcross.org -> dids.givechariot.com)

  3. The organization was registered on the Solana on-chain registry
     with VC hash for integrity verification

  4. A payer built an ISO 20022-inspired payment message:
     - PUBLIC on-chain: $500.00 DAF grant to EIN 530196605
     - PRIVATE encrypted: donor name, email, address, fund details

  5. The private data was encrypted to the nonprofit's X25519 key
     and pinned to IPFS (only the nonprofit can decrypt)

  6. The nonprofit retrieved and decrypted the full payment data,
     verifying integrity against the on-chain hash commitment`)

	return nil
}

func issueCmd() *cobra.Command {
	var ein, name, domain string

	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue Verifiable Credentials for a nonprofit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(ein) != 9 {
				return fmt.Errorf("EIN must be exactly 9 digits")
			}

			// Generate issuer keypair
			chariotKP, err := did.GenerateKeyPair()
			if err != nil {
				return err
			}
			chariotDID := "did:web:givechariot.com"

			issuer, err := credential.NewIssuer(chariotDID, chariotKP.PrivateKey)
			if err != nil {
				return err
			}

			nonprofitDID := "did:web:agapay." + domain

			orgVC, err := issuer.IssueOrganization(nonprofitDID, &credential.OrganizationClaims{
				OrganizationName: name,
				Domain:           domain,
				EntityDID:        nonprofitDID,
				EntityEIN:        ein,
				Affiliation:      "independent_nonprofit",
			})
			if err != nil {
				return err
			}

			fmt.Printf("Issued OrganizationCredential:\n%s\n", orgVC)

			entityVC, err := issuer.IssueNonprofitEntity(nonprofitDID, &credential.NonprofitEntityClaims{
				EIN:       ein,
				LegalName: name,
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nIssued NonprofitEntityCredential:\n%s\n", entityVC)
			return nil
		},
	}

	cmd.Flags().StringVar(&ein, "ein", "", "Employer Identification Number (9 digits)")
	cmd.Flags().StringVar(&name, "name", "", "Organization name")
	cmd.Flags().StringVar(&domain, "domain", "", "Organization domain (e.g., redcross.org)")
	cmd.MarkFlagRequired("ein")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("domain")

	return cmd
}

func verifyCmd() *cobra.Command {
	var vcToken string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a Verifiable Credential JWT",
		RunE: func(cmd *cobra.Command, args []string) error {
			// For the PoC, we parse without verification to show claims
			// In production, you'd resolve the issuer DID to get the public key
			fmt.Println("Parsing VC-JWT (signature verification requires issuer public key)...")

			verifier := credential.NewVerifier()

			// Generate a dummy key since we can't verify without the issuer key
			kp, _ := did.GenerateKeyPair()
			result, err := verifier.Verify(vcToken, kp.PublicKey)
			if err != nil {
				return err
			}

			if result.Valid {
				fmt.Printf("Valid: true\n")
				fmt.Printf("Issuer: %s\n", result.IssuerDID)
				fmt.Printf("Subject: %s\n", result.SubjectDID)
				fmt.Printf("Type: %s\n", strings.Join(result.CredentialType, ", "))
				claimsJSON, _ := json.MarshalIndent(result.CredentialSubject, "", "  ")
				fmt.Printf("Claims:\n%s\n", string(claimsJSON))
			} else {
				fmt.Printf("Valid: false\nError: %s\n", result.Error)
				fmt.Println("(This is expected if the issuer public key is not available)")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&vcToken, "vc", "", "The VC-JWT token to verify")
	cmd.MarkFlagRequired("vc")

	return cmd
}

func lookupCmd() *cobra.Command {
	var ein, rpcURL string

	cmd := &cobra.Command{
		Use:   "lookup",
		Short: "Look up an organization on the Solana registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Looking up EIN %s on the Agapay registry...\n\n", ein)

			if rpcURL != "" {
				// Real Solana lookup
				// For now, simulate since we need a deployed program
				fmt.Printf("RPC URL: %s\n", rpcURL)
			}

			// Simulate lookup result
			fmt.Printf("Organization found:\n")
			fmt.Printf("  EIN: %s\n", ein)
			fmt.Printf("  (Full lookup requires a deployed Agapay Registry program on Solana)\n")
			fmt.Printf("  Use 'agapay demo' to see the full flow with simulated data.\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&ein, "ein", "", "EIN to look up (9 digits)")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "Solana RPC URL (optional)")
	cmd.MarkFlagRequired("ein")

	return cmd
}

func payCmd() *cobra.Command {
	var ein, paymentType, donorName, donorEmail string
	var amount int64

	cmd := &cobra.Command{
		Use:   "pay",
		Short: "Send a payment to a nonprofit",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Building payment to EIN %s...\n\n", ein)

			// Generate a recipient keypair for encryption demo
			recipientKP, _ := did.GenerateKeyPair()

			instruction := &message.AgapayPaymentInstruction{
				PublicHeader: message.PublicHeader{
					SenderDID:    "did:web:givechariot.com:payers:demo",
					RecipientDID: "did:web:agapay." + ein + ".org",
					RecipientEIN: ein,
					PaymentType:  paymentType,
					CreatedAt:    time.Now().UTC().Format(time.RFC3339),
				},
				PrivateBody: message.PrivateBody{
					Transactions: []message.Transaction{
						{
							Amount:   amount,
							Currency: "USD",
							Donation: &message.Donation{Type: paymentType},
							Donors: []message.Donor{
								{Name: donorName, Email: donorEmail},
							},
						},
					},
				},
			}

			header, body, err := message.Split(instruction)
			if err != nil {
				return err
			}

			// Encrypt
			envelope, err := message.Encrypt(body, recipientKP.EncryptionPublicKey)
			if err != nil {
				return err
			}

			// Pin to mock IPFS
			mockIPFS := ipfs.NewMockClient()
			cid, err := mockIPFS.PinJSON(envelope)
			if err != nil {
				return err
			}
			header.PrivateDataCID = cid

			headerJSON, _ := json.MarshalIndent(header, "", "  ")
			fmt.Printf("Public Header (on-chain):\n%s\n\n", string(headerJSON))
			fmt.Printf("Private Data CID: %s\n", cid)
			fmt.Printf("USDC Amount: %d units ($%.2f)\n",
				uint64(amount)*10000,
				float64(amount)/100)
			fmt.Printf("\n(In production, this would submit a Solana transaction)\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&ein, "ein", "", "Recipient EIN (9 digits)")
	cmd.Flags().Int64Var(&amount, "amount", 0, "Amount in cents")
	cmd.Flags().StringVar(&paymentType, "type", "donor_advised_fund_grant", "Payment type")
	cmd.Flags().StringVar(&donorName, "donor-name", "", "Donor name (PII, will be encrypted)")
	cmd.Flags().StringVar(&donorEmail, "donor-email", "", "Donor email (PII, will be encrypted)")
	cmd.MarkFlagRequired("ein")
	cmd.MarkFlagRequired("amount")

	return cmd
}

func decryptCmd() *cobra.Command {
	var cid, keyHex string

	cmd := &cobra.Command{
		Use:   "decrypt",
		Short: "Decrypt payment data from IPFS",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Decrypting data from IPFS CID %s...\n\n", cid)
			fmt.Println("(This command requires access to the recipient's X25519 private key")
			fmt.Println("and a running IPFS node or the mock store from a prior 'pay' command.)")
			fmt.Println("\nUse 'agapay demo' to see the full encrypt/decrypt flow.")

			if keyHex != "" {
				keyBytes, err := hex.DecodeString(keyHex)
				if err != nil {
					return fmt.Errorf("decode key: %w", err)
				}
				if len(keyBytes) != 32 {
					return fmt.Errorf("key must be 32 bytes (64 hex chars)")
				}
				fmt.Printf("Key provided: %s...\n", keyHex[:16])
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cid, "cid", "", "IPFS CID of encrypted data")
	cmd.Flags().StringVar(&keyHex, "key", "", "Recipient X25519 private key (hex)")
	cmd.MarkFlagRequired("cid")

	return cmd
}

func printHeader(title string) {
	line := strings.Repeat("=", len(title)+4)
	fmt.Printf("\n%s\n  %s\n%s\n\n", line, title, line)
}

func printStep(num int, description string) {
	fmt.Printf("\n--- Step %d: %s ---\n\n", num, description)
}

// Ensure sha256 import is used
var _ = sha256.New
