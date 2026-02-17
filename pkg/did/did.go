// Package did provides DID (Decentralized Identifier) creation and resolution
// for the Agapay network, supporting did:key and did:web methods.
package did

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/crypto/curve25519"
)

// KeyPair holds an Ed25519 signing keypair and a derived X25519 encryption keypair.
type KeyPair struct {
	// Ed25519 signing keys
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey

	// X25519 encryption keys (derived from Ed25519)
	EncryptionPublicKey  [32]byte
	EncryptionPrivateKey [32]byte
}

// GenerateKeyPair creates a new Ed25519 keypair and derives the corresponding
// X25519 encryption keypair for use in key agreement (ECDH).
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	kp := &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}

	// Derive X25519 keys from Ed25519 keys.
	// The Ed25519 private key seed (first 32 bytes) is used as the X25519 private scalar.
	copy(kp.EncryptionPrivateKey[:], priv.Seed())
	encPub, err := curve25519.X25519(kp.EncryptionPrivateKey[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("derive x25519 public key: %w", err)
	}
	copy(kp.EncryptionPublicKey[:], encPub)

	return kp, nil
}

// multibaseEncode encodes bytes using multibase with base58btc prefix ('z').
func multibaseEncode(data []byte) string {
	return "z" + base58Encode(data)
}

// multicodecEd25519 is the multicodec prefix for Ed25519 public keys (0xed01).
var multicodecEd25519 = []byte{0xed, 0x01}

// multicodecX25519 is the multicodec prefix for X25519 public keys (0xec01).
var multicodecX25519 = []byte{0xec, 0x01}

// CreateDIDKey creates a did:key identifier from an Ed25519 public key.
// Format: did:key:z<multibase(multicodec-ed25519 + public-key-bytes)>
func CreateDIDKey(pub ed25519.PublicKey) string {
	// Prepend the multicodec prefix for ed25519-pub
	data := append(multicodecEd25519, pub...)
	return "did:key:" + multibaseEncode(data)
}

// VerificationMethod represents a verification method in a DID Document.
type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
}

// ServiceEndpoint represents a service in a DID Document.
type ServiceEndpoint struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// Document represents a DID Document.
type Document struct {
	Context            []string             `json:"@context"`
	ID                 string               `json:"id"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
	Authentication     []string             `json:"authentication"`
	KeyAgreement       []string             `json:"keyAgreement,omitempty"`
	Service            []ServiceEndpoint    `json:"service,omitempty"`
}

// CreateDIDWebDocument creates a DID Document for a did:web identifier.
// The domain is used as the DID (e.g., "agapay.redcross.org" -> "did:web:agapay.redcross.org").
func CreateDIDWebDocument(domain string, kp *KeyPair, serviceEndpoint string) *Document {
	didID := "did:web:" + domain

	// Encode Ed25519 public key for verificationMethod
	ed25519Multi := append(multicodecEd25519, kp.PublicKey...)
	ed25519Encoded := multibaseEncode(ed25519Multi)

	// Encode X25519 public key for keyAgreement
	x25519Multi := append(multicodecX25519, kp.EncryptionPublicKey[:]...)
	x25519Encoded := multibaseEncode(x25519Multi)

	doc := &Document{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/ed25519-2020/v1",
		},
		ID: didID,
		VerificationMethod: []VerificationMethod{
			{
				ID:                 didID + "#key-1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         didID,
				PublicKeyMultibase: ed25519Encoded,
			},
			{
				ID:                 didID + "#key-enc-1",
				Type:               "X25519KeyAgreementKey2020",
				Controller:         didID,
				PublicKeyMultibase: x25519Encoded,
			},
		},
		Authentication: []string{didID + "#key-1"},
		KeyAgreement:   []string{didID + "#key-enc-1"},
	}

	if serviceEndpoint != "" {
		doc.Service = []ServiceEndpoint{
			{
				ID:              didID + "#agapay",
				Type:            "AgapayEndpoint",
				ServiceEndpoint: serviceEndpoint,
			},
		}
	}

	return doc
}

// ResolveDIDWeb resolves a did:web identifier by fetching the DID Document over HTTPS.
// did:web:example.com -> https://example.com/.well-known/did.json
// did:web:example.com:path:to -> https://example.com/path/to/did.json
func ResolveDIDWeb(did string) (*Document, error) {
	if !strings.HasPrefix(did, "did:web:") {
		return nil, fmt.Errorf("not a did:web identifier: %s", did)
	}

	parts := strings.SplitN(did, ":", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid did:web format: %s", did)
	}

	domainAndPath := parts[2]
	pathParts := strings.Split(domainAndPath, ":")

	var url string
	if len(pathParts) == 1 {
		url = "https://" + pathParts[0] + "/.well-known/did.json"
	} else {
		url = "https://" + pathParts[0] + "/" + strings.Join(pathParts[1:], "/") + "/did.json"
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch DID document from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DID document fetch returned status %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read DID document body: %w", err)
	}

	var doc Document
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse DID document: %w", err)
	}

	return &doc, nil
}

// ExtractVerificationKey extracts the Ed25519 public key from a DID Document's
// first verification method of type Ed25519VerificationKey2020.
func ExtractVerificationKey(doc *Document) (ed25519.PublicKey, error) {
	for _, vm := range doc.VerificationMethod {
		if vm.Type == "Ed25519VerificationKey2020" {
			return decodeMultibaseEd25519(vm.PublicKeyMultibase)
		}
	}
	return nil, fmt.Errorf("no Ed25519VerificationKey2020 found in DID document")
}

// ExtractEncryptionKey extracts the X25519 public key from a DID Document's
// first verification method of type X25519KeyAgreementKey2020.
func ExtractEncryptionKey(doc *Document) ([32]byte, error) {
	var key [32]byte
	for _, vm := range doc.VerificationMethod {
		if vm.Type == "X25519KeyAgreementKey2020" {
			raw, err := decodeMultibaseX25519(vm.PublicKeyMultibase)
			if err != nil {
				return key, err
			}
			copy(key[:], raw)
			return key, nil
		}
	}
	return key, fmt.Errorf("no X25519KeyAgreementKey2020 found in DID document")
}

// decodeMultibaseEd25519 decodes a multibase-encoded Ed25519 public key.
func decodeMultibaseEd25519(encoded string) (ed25519.PublicKey, error) {
	if len(encoded) == 0 || encoded[0] != 'z' {
		return nil, fmt.Errorf("unsupported multibase encoding (expected 'z' prefix)")
	}
	data, err := base58Decode(encoded[1:])
	if err != nil {
		return nil, fmt.Errorf("base58 decode: %w", err)
	}
	// Strip multicodec prefix (0xed01)
	if len(data) < 2+ed25519.PublicKeySize {
		return nil, fmt.Errorf("decoded key too short")
	}
	if data[0] != 0xed || data[1] != 0x01 {
		return nil, fmt.Errorf("unexpected multicodec prefix for ed25519")
	}
	return ed25519.PublicKey(data[2:]), nil
}

// decodeMultibaseX25519 decodes a multibase-encoded X25519 public key.
func decodeMultibaseX25519(encoded string) ([]byte, error) {
	if len(encoded) == 0 || encoded[0] != 'z' {
		return nil, fmt.Errorf("unsupported multibase encoding (expected 'z' prefix)")
	}
	data, err := base58Decode(encoded[1:])
	if err != nil {
		return nil, fmt.Errorf("base58 decode: %w", err)
	}
	if len(data) < 2+32 {
		return nil, fmt.Errorf("decoded key too short")
	}
	if data[0] != 0xec || data[1] != 0x01 {
		return nil, fmt.Errorf("unexpected multicodec prefix for x25519")
	}
	return data[2:], nil
}

// base58Encode encodes bytes to base58btc (Bitcoin alphabet).
func base58Encode(data []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Count leading zeros
	var leadingZeros int
	for _, b := range data {
		if b != 0 {
			break
		}
		leadingZeros++
	}

	// Convert to big integer and encode
	size := len(data)*138/100 + 1
	buf := make([]byte, size)
	var length int

	for _, b := range data {
		carry := int(b)
		for j := 0; j < length || carry != 0; j++ {
			if j == length {
				length = j + 1
			}
			carry += 256 * int(buf[j])
			buf[j] = byte(carry % 58)
			carry /= 58
		}
	}

	result := make([]byte, leadingZeros+length)
	for i := 0; i < leadingZeros; i++ {
		result[i] = alphabet[0]
	}
	for i := 0; i < length; i++ {
		result[leadingZeros+i] = alphabet[buf[length-1-i]]
	}

	return string(result)
}

// base58Decode decodes a base58btc string to bytes.
func base58Decode(s string) ([]byte, error) {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Build reverse lookup
	var lookup [128]int
	for i := range lookup {
		lookup[i] = -1
	}
	for i, c := range alphabet {
		lookup[c] = i
	}

	// Count leading '1's (zeros in base58)
	var leadingZeros int
	for _, c := range s {
		if c != '1' {
			break
		}
		leadingZeros++
	}

	size := len(s)*733/1000 + 1
	buf := make([]byte, size)
	var length int

	for _, c := range s {
		if int(c) >= len(lookup) || lookup[c] == -1 {
			return nil, fmt.Errorf("invalid base58 character: %c", c)
		}
		carry := lookup[c]
		for j := 0; j < length || carry != 0; j++ {
			if j == length {
				length = j + 1
			}
			carry += 58 * int(buf[j])
			buf[j] = byte(carry % 256)
			carry /= 256
		}
	}

	result := make([]byte, leadingZeros+length)
	for i := length - 1; i >= 0; i-- {
		result[leadingZeros+length-1-i] = buf[i]
	}

	return result, nil
}
