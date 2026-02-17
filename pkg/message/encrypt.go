package message

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	hkdfInfo      = "agapay-payment-v1"
	aesKeySize    = 32 // AES-256
	nonceSize     = 12 // GCM standard nonce
	algorithmName = "ECDH-ES+AES256GCM"
)

// Encrypt encrypts a PrivateBody using hybrid encryption:
// 1. Generate ephemeral X25519 keypair
// 2. ECDH key agreement with recipient's X25519 public key
// 3. HKDF-SHA256 key derivation
// 4. AES-256-GCM encryption
//
// Returns an EncryptedEnvelope ready for IPFS storage.
func Encrypt(body *PrivateBody, recipientPublicKey [32]byte) (*EncryptedEnvelope, error) {
	// Serialize the private body to JSON
	plaintext, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal private body: %w", err)
	}

	// Generate ephemeral X25519 keypair
	var ephemeralPrivate [32]byte
	if _, err := io.ReadFull(rand.Reader, ephemeralPrivate[:]); err != nil {
		return nil, fmt.Errorf("generate ephemeral key: %w", err)
	}

	ephemeralPublic, err := curve25519.X25519(ephemeralPrivate[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("compute ephemeral public key: %w", err)
	}

	// ECDH key agreement
	sharedSecret, err := curve25519.X25519(ephemeralPrivate[:], recipientPublicKey[:])
	if err != nil {
		return nil, fmt.Errorf("ECDH key agreement: %w", err)
	}

	// Derive symmetric key via HKDF-SHA256
	symmetricKey, err := deriveKey(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("derive symmetric key: %w", err)
	}

	// Encrypt with AES-256-GCM
	block, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// GCM Seal appends the auth tag to the ciphertext
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedEnvelope{
		Version:            "1.0",
		Algorithm:          algorithmName,
		EphemeralPublicKey: base64.StdEncoding.EncodeToString(ephemeralPublic),
		Nonce:              base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:         base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

// deriveKey derives a 256-bit AES key from a shared secret using HKDF-SHA256.
func deriveKey(sharedSecret []byte) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, sharedSecret, nil, []byte(hkdfInfo))
	key := make([]byte, aesKeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("HKDF read: %w", err)
	}
	return key, nil
}
