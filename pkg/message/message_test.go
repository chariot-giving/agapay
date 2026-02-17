package message

import (
	"testing"

	"github.com/chariot-giving/agapay/pkg/did"
)

func TestSplitAndEncryptDecryptRoundTrip(t *testing.T) {
	// Generate recipient keypair
	recipientKP, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// Build a test payment instruction
	instruction := &AgapayPaymentInstruction{
		PublicHeader: PublicHeader{
			SenderDID:    "did:web:givechariot.com:payers:vanguard",
			RecipientDID: "did:web:agapay.redcross.org",
			RecipientEIN: "530196605",
			PaymentType:  "donor_advised_fund_grant",
		},
		PrivateBody: PrivateBody{
			Transactions: []Transaction{
				{
					TransactionID: "tx_01",
					Amount:        50000,
					Currency:      "USD",
					Description:   "General Operating Support",
					Donation: &Donation{
						Type:             "donor_advised_fund_grant",
						OrganizationName: "Vanguard Charitable",
						FundName:         "John Doe Giving Fund",
						Purpose:          "General Operating Support",
						Note:             "Annual grant from the Doe family",
					},
					Donors: []Donor{
						{
							Name:  "John Doe",
							Email: "john.doe@example.com",
							Phone: "415-555-1212",
							Address: &PostalAddress{
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
			RemittanceInfo:  "DAF Grant - Q1 2026",
		},
	}

	// Split into public and private parts
	header, body, err := Split(instruction)
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}

	// Verify header fields
	if header.Version != "1.0" {
		t.Errorf("header.Version = %s, want 1.0", header.Version)
	}
	if header.TotalAmount != 50000 {
		t.Errorf("header.TotalAmount = %d, want 50000", header.TotalAmount)
	}
	if header.NumberOfTransactions != 1 {
		t.Errorf("header.NumberOfTransactions = %d, want 1", header.NumberOfTransactions)
	}
	if header.PrivateDataHash == "" {
		t.Error("header.PrivateDataHash is empty")
	}
	if header.RecipientEIN != "530196605" {
		t.Errorf("header.RecipientEIN = %s, want 530196605", header.RecipientEIN)
	}

	// Encrypt the private body
	envelope, err := Encrypt(body, recipientKP.EncryptionPublicKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if envelope.Algorithm != "ECDH-ES+AES256GCM" {
		t.Errorf("envelope.Algorithm = %s, want ECDH-ES+AES256GCM", envelope.Algorithm)
	}
	if envelope.EphemeralPublicKey == "" {
		t.Error("envelope.EphemeralPublicKey is empty")
	}
	if envelope.Ciphertext == "" {
		t.Error("envelope.Ciphertext is empty")
	}

	// Decrypt and verify
	decrypted, err := DecryptAndVerify(envelope, recipientKP.EncryptionPrivateKey, header.PrivateDataHash)
	if err != nil {
		t.Fatalf("DecryptAndVerify() error = %v", err)
	}

	// Verify decrypted data matches original
	if len(decrypted.Transactions) != 1 {
		t.Fatalf("len(decrypted.Transactions) = %d, want 1", len(decrypted.Transactions))
	}

	tx := decrypted.Transactions[0]
	if tx.Amount != 50000 {
		t.Errorf("tx.Amount = %d, want 50000", tx.Amount)
	}
	if tx.Donors[0].Name != "John Doe" {
		t.Errorf("tx.Donors[0].Name = %s, want John Doe", tx.Donors[0].Name)
	}
	if tx.Donors[0].Email != "john.doe@example.com" {
		t.Errorf("tx.Donors[0].Email = %s, want john.doe@example.com", tx.Donors[0].Email)
	}
	if decrypted.SenderReference != "VNG-2026-00142" {
		t.Errorf("SenderReference = %s, want VNG-2026-00142", decrypted.SenderReference)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	recipientKP, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	wrongKP, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	body := &PrivateBody{
		Transactions: []Transaction{
			{Amount: 1000, Currency: "USD"},
		},
	}

	envelope, err := Encrypt(body, recipientKP.EncryptionPublicKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with the wrong key -- should fail
	_, err = Decrypt(envelope, wrongKP.EncryptionPrivateKey)
	if err == nil {
		t.Error("Decrypt() with wrong key should return error")
	}
}

func TestHashVerificationFailure(t *testing.T) {
	recipientKP, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	body := &PrivateBody{
		Transactions: []Transaction{
			{Amount: 1000, Currency: "USD"},
		},
	}

	envelope, err := Encrypt(body, recipientKP.EncryptionPublicKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Use a wrong hash
	wrongHash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	_, err = DecryptAndVerify(envelope, recipientKP.EncryptionPrivateKey, wrongHash)
	if err == nil {
		t.Error("DecryptAndVerify() with wrong hash should return error")
	}
}
