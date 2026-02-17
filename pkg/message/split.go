package message

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"crypto/rand"
	"encoding/hex"
)

// Split separates an AgapayPaymentInstruction into its public header and
// private body, computing the private body hash for integrity verification.
// The PublicHeader's PrivateDataCID must be set after encryption and IPFS pinning.
func Split(instruction *AgapayPaymentInstruction) (*PublicHeader, *PrivateBody, error) {
	body := &instruction.PrivateBody

	// Serialize the private body to compute its hash
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal private body: %w", err)
	}

	hash := sha256.Sum256(bodyBytes)
	hashHex := "sha256:" + hex.EncodeToString(hash[:])

	// Build the public header
	header := &instruction.PublicHeader
	header.PrivateDataHash = hashHex

	// Populate defaults if not set
	if header.Version == "" {
		header.Version = "1.0"
	}
	if header.MessageID == "" {
		header.MessageID = generateMessageID()
	}
	if header.CreatedAt == "" {
		header.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if header.NumberOfTransactions == 0 {
		header.NumberOfTransactions = len(body.Transactions)
	}
	if header.Currency == "" {
		header.Currency = "USD"
	}

	// Compute total amount from transactions if not set
	if header.TotalAmount == 0 {
		var total int64
		for _, tx := range body.Transactions {
			total += tx.Amount
		}
		header.TotalAmount = total
	}

	return header, body, nil
}

// ComputePrivateBodyHash computes the SHA-256 hash of a serialized private body.
func ComputePrivateBodyHash(body *PrivateBody) (string, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal private body: %w", err)
	}
	hash := sha256.Sum256(bodyBytes)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

// generateMessageID creates a random message identifier.
func generateMessageID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "msg_" + hex.EncodeToString(b)
}
