package did

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if len(kp.PublicKey) != 32 {
		t.Errorf("PublicKey length = %d, want 32", len(kp.PublicKey))
	}
	if len(kp.PrivateKey) != 64 {
		t.Errorf("PrivateKey length = %d, want 64", len(kp.PrivateKey))
	}
	if kp.EncryptionPublicKey == [32]byte{} {
		t.Error("EncryptionPublicKey is zero")
	}
	if kp.EncryptionPrivateKey == [32]byte{} {
		t.Error("EncryptionPrivateKey is zero")
	}
}

func TestCreateDIDKey(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	did := CreateDIDKey(kp.PublicKey)
	if !strings.HasPrefix(did, "did:key:z") {
		t.Errorf("CreateDIDKey() = %s, want prefix did:key:z", did)
	}
}

func TestCreateDIDWebDocument(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	domain := "agapay.redcross.org"
	doc := CreateDIDWebDocument(domain, kp, "https://api.givechariot.com/v1/orgs/123")

	if doc.ID != "did:web:agapay.redcross.org" {
		t.Errorf("doc.ID = %s, want did:web:agapay.redcross.org", doc.ID)
	}
	if len(doc.VerificationMethod) != 2 {
		t.Errorf("len(VerificationMethod) = %d, want 2", len(doc.VerificationMethod))
	}
	if doc.VerificationMethod[0].Type != "Ed25519VerificationKey2020" {
		t.Errorf("first VM type = %s, want Ed25519VerificationKey2020", doc.VerificationMethod[0].Type)
	}
	if doc.VerificationMethod[1].Type != "X25519KeyAgreementKey2020" {
		t.Errorf("second VM type = %s, want X25519KeyAgreementKey2020", doc.VerificationMethod[1].Type)
	}
	if len(doc.Authentication) != 1 {
		t.Errorf("len(Authentication) = %d, want 1", len(doc.Authentication))
	}
	if len(doc.KeyAgreement) != 1 {
		t.Errorf("len(KeyAgreement) = %d, want 1", len(doc.KeyAgreement))
	}
	if len(doc.Service) != 1 {
		t.Errorf("len(Service) = %d, want 1", len(doc.Service))
	}
}

func TestExtractKeys(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	doc := CreateDIDWebDocument("agapay.example.org", kp, "")

	// Extract and verify Ed25519 key
	verKey, err := ExtractVerificationKey(doc)
	if err != nil {
		t.Fatalf("ExtractVerificationKey() error = %v", err)
	}
	if !verKey.Equal(kp.PublicKey) {
		t.Error("extracted verification key does not match original")
	}

	// Extract and verify X25519 key
	encKey, err := ExtractEncryptionKey(doc)
	if err != nil {
		t.Fatalf("ExtractEncryptionKey() error = %v", err)
	}
	if encKey != kp.EncryptionPublicKey {
		t.Error("extracted encryption key does not match original")
	}
}

func TestBase58RoundTrip(t *testing.T) {
	testCases := [][]byte{
		{},
		{0},
		{0, 0, 0},
		{1, 2, 3, 4, 5},
		{0xff, 0xfe, 0xfd},
	}

	for _, tc := range testCases {
		encoded := base58Encode(tc)
		decoded, err := base58Decode(encoded)
		if err != nil {
			t.Errorf("base58Decode(%s) error = %v", encoded, err)
			continue
		}
		if len(tc) == 0 && len(decoded) == 0 {
			continue
		}
		if string(decoded) != string(tc) {
			t.Errorf("round-trip failed: got %v, want %v", decoded, tc)
		}
	}
}

func TestHostingService(t *testing.T) {
	svc := NewHostingService()

	domain := "agapay.testnonprofit.org"
	_, doc, err := svc.RegisterDID(domain, "https://api.givechariot.com/v1/orgs/test")
	if err != nil {
		t.Fatalf("RegisterDID() error = %v", err)
	}

	if doc.ID != "did:web:"+domain {
		t.Errorf("doc.ID = %s, want did:web:%s", doc.ID, domain)
	}

	// Test HTTP handler
	handler := svc.Handler()

	// Valid request
	req := httptest.NewRequest("GET", "/.well-known/did.json", nil)
	req.Host = domain
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var fetched Document
	if err := json.Unmarshal(w.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if fetched.ID != doc.ID {
		t.Errorf("fetched.ID = %s, want %s", fetched.ID, doc.ID)
	}

	// Unknown domain
	req2 := httptest.NewRequest("GET", "/.well-known/did.json", nil)
	req2.Host = "unknown.org"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotFound {
		t.Errorf("unknown domain status = %d, want %d", w2.Code, http.StatusNotFound)
	}

	// Wrong path
	req3 := httptest.NewRequest("GET", "/other/path", nil)
	req3.Host = domain
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusNotFound {
		t.Errorf("wrong path status = %d, want %d", w3.Code, http.StatusNotFound)
	}
}
