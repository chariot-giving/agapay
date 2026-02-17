package message

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// Decrypt decrypts an EncryptedEnvelope using the recipient's X25519 private key.
// Returns the decrypted PrivateBody.
func Decrypt(envelope *EncryptedEnvelope, recipientPrivateKey [32]byte) (*PrivateBody, error) {
	if envelope.Algorithm != algorithmName {
		return nil, fmt.Errorf("unsupported algorithm: %s", envelope.Algorithm)
	}

	// Decode base64 fields
	ephemeralPub, err := base64.StdEncoding.DecodeString(envelope.EphemeralPublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode ephemeral public key: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	// ECDH key agreement using recipient's private key and ephemeral public key
	sharedSecret, err := curve25519.X25519(recipientPrivateKey[:], ephemeralPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH key agreement: %w", err)
	}

	// Derive the same symmetric key
	symmetricKey, err := deriveKey(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("derive symmetric key: %w", err)
	}

	// Decrypt with AES-256-GCM
	block, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	// Parse the decrypted JSON
	var body PrivateBody
	if err := json.Unmarshal(plaintext, &body); err != nil {
		return nil, fmt.Errorf("parse decrypted private body: %w", err)
	}

	return &body, nil
}

// DecryptAndVerify decrypts the envelope and verifies the plaintext hash
// matches the expected hash from the on-chain public header.
func DecryptAndVerify(envelope *EncryptedEnvelope, recipientPrivateKey [32]byte, expectedHash string) (*PrivateBody, error) {
	body, err := Decrypt(envelope, recipientPrivateKey)
	if err != nil {
		return nil, err
	}

	// Recompute the hash of the decrypted plaintext
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal decrypted body for hash: %w", err)
	}

	hash := sha256.Sum256(bodyBytes)
	computedHash := "sha256:" + hex.EncodeToString(hash[:])

	if computedHash != expectedHash {
		return nil, fmt.Errorf("hash mismatch: computed %s, expected %s", computedHash, expectedHash)
	}

	return body, nil
}
