package credential

import (
	"testing"

	"github.com/chariot-giving/agapay/pkg/did"
)

func TestIssueAndVerifyNonprofitEntity(t *testing.T) {
	// Generate issuer keypair
	kp, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	issuerDID := "did:web:givechariot.com"
	issuer, err := NewIssuer(issuerDID, kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewIssuer() error = %v", err)
	}

	subjectDID := "did:web:agapay.redcross.org"
	claims := &NonprofitEntityClaims{
		EIN:       "530196605",
		LegalName: "American Red Cross",
		PhysicalAddress: PostalAddress{
			Line1:      "430 17th St NW",
			City:       "Washington",
			State:      "DC",
			PostalCode: "20006",
			Country:    "US",
		},
		IRSSubsectionCode: "03",
		IRSPub78:          true,
	}

	token, err := issuer.IssueNonprofitEntity(subjectDID, claims)
	if err != nil {
		t.Fatalf("IssueNonprofitEntity() error = %v", err)
	}

	if token == "" {
		t.Fatal("IssueNonprofitEntity() returned empty token")
	}

	// Verify the token
	verifier := NewVerifier()
	result, err := verifier.Verify(token, kp.PublicKey)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Valid {
		t.Fatalf("Verify() result.Valid = false, error = %s", result.Error)
	}
	if result.IssuerDID != issuerDID {
		t.Errorf("IssuerDID = %s, want %s", result.IssuerDID, issuerDID)
	}
	if result.SubjectDID != subjectDID {
		t.Errorf("SubjectDID = %s, want %s", result.SubjectDID, subjectDID)
	}

	// Check that NonprofitEntityCredential is in the types
	found := false
	for _, ct := range result.CredentialType {
		if ct == string(TypeNonprofitEntity) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CredentialType %v does not contain %s", result.CredentialType, TypeNonprofitEntity)
	}

	// Check claims
	if ein, ok := result.CredentialSubject["ein"].(string); !ok || ein != "530196605" {
		t.Errorf("EIN claim = %v, want 530196605", result.CredentialSubject["ein"])
	}
}

func TestIssueAndVerifyControlPerson(t *testing.T) {
	kp, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	issuer, err := NewIssuer("did:web:givechariot.com", kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewIssuer() error = %v", err)
	}

	personKP, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	personDID := did.CreateDIDKey(personKP.PublicKey)

	claims := &ControlPersonClaims{
		FullName:  "Jane Smith",
		Title:     "Executive Director",
		Email:     "jane@redcross.org",
		EntityDID: "did:web:agapay.redcross.org",
		EntityEIN: "530196605",
		Role:      "officer",
	}

	token, err := issuer.IssueControlPerson(personDID, claims)
	if err != nil {
		t.Fatalf("IssueControlPerson() error = %v", err)
	}

	verifier := NewVerifier()
	result, err := verifier.Verify(token, kp.PublicKey)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Valid {
		t.Fatalf("result not valid: %s", result.Error)
	}
	if result.SubjectDID != personDID {
		t.Errorf("SubjectDID = %s, want %s", result.SubjectDID, personDID)
	}
}

func TestIssueAndVerifyOrganization(t *testing.T) {
	kp, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	issuer, err := NewIssuer("did:web:givechariot.com", kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewIssuer() error = %v", err)
	}

	claims := &OrganizationClaims{
		OrganizationName: "American Red Cross",
		Domain:           "redcross.org",
		EntityDID:        "did:web:agapay.redcross.org",
		EntityEIN:        "530196605",
		Affiliation:      "independent_nonprofit",
		NTEECode:         "P50",
	}

	token, err := issuer.IssueOrganization("did:web:agapay.redcross.org", claims)
	if err != nil {
		t.Fatalf("IssueOrganization() error = %v", err)
	}

	verifier := NewVerifier()
	result, err := verifier.Verify(token, kp.PublicKey)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Valid {
		t.Fatalf("result not valid: %s", result.Error)
	}
}

func TestIssueAndVerifyAddress(t *testing.T) {
	kp, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	issuer, err := NewIssuer("did:web:givechariot.com", kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewIssuer() error = %v", err)
	}

	claims := &AddressClaims{
		OrganizationDID:         "did:web:agapay.redcross.org",
		OrganizationEIN:         "530196605",
		AddressType:             "solana_wallet",
		SupportedPaymentMethods: []string{"usdc"},
		SolanaWallet:            "7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU",
	}

	token, err := issuer.IssueAddress("did:web:agapay.redcross.org", claims)
	if err != nil {
		t.Fatalf("IssueAddress() error = %v", err)
	}

	verifier := NewVerifier()
	result, err := verifier.Verify(token, kp.PublicKey)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Valid {
		t.Fatalf("result not valid: %s", result.Error)
	}
}

func TestVerifyWithWrongKey(t *testing.T) {
	kp, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	wrongKP, err := did.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	issuer, err := NewIssuer("did:web:givechariot.com", kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewIssuer() error = %v", err)
	}

	claims := &OrganizationClaims{
		OrganizationName: "Test Org",
		Domain:           "test.org",
		EntityDID:        "did:web:agapay.test.org",
		EntityEIN:        "123456789",
		Affiliation:      "independent_nonprofit",
	}

	token, err := issuer.IssueOrganization("did:web:agapay.test.org", claims)
	if err != nil {
		t.Fatalf("IssueOrganization() error = %v", err)
	}

	verifier := NewVerifier()
	result, err := verifier.Verify(token, wrongKP.PublicKey)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Valid {
		t.Error("Verify() with wrong key should return Valid=false")
	}
}
