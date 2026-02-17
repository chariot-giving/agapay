package credential

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Issuer signs Verifiable Credentials as JWTs using Ed25519.
type Issuer struct {
	DID        string
	PrivateKey ed25519.PrivateKey
	jwkKey     jwk.Key
}

// NewIssuer creates a new VC issuer with the given DID and Ed25519 private key.
func NewIssuer(did string, privateKey ed25519.PrivateKey) (*Issuer, error) {
	key, err := jwk.FromRaw(privateKey)
	if err != nil {
		return nil, fmt.Errorf("create JWK from private key: %w", err)
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.EdDSA); err != nil {
		return nil, fmt.Errorf("set algorithm on JWK: %w", err)
	}
	if err := key.Set(jwk.KeyIDKey, did+"#key-1"); err != nil {
		return nil, fmt.Errorf("set key ID on JWK: %w", err)
	}

	return &Issuer{
		DID:        did,
		PrivateKey: privateKey,
		jwkKey:     key,
	}, nil
}

// Issue creates and signs a Verifiable Credential as a JWT (VC-JWT).
// The VC is encoded as JWT claims per the W3C VC-JWT specification.
func (iss *Issuer) Issue(vc *VerifiableCredential) (string, error) {
	// Build JWT claims
	now := time.Now().UTC()
	builder := jwt.NewBuilder().
		Issuer(iss.DID).
		Subject(vc.CredentialSubject["id"].(string)).
		IssuedAt(now).
		NotBefore(now)

	// Set expiration if provided
	if vc.ExpirationDate != "" {
		exp, err := time.Parse(time.RFC3339, vc.ExpirationDate)
		if err == nil {
			builder = builder.Expiration(exp)
		}
	}

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("build JWT: %w", err)
	}

	// Set the vc claim containing the full credential
	vcClaim := map[string]interface{}{
		"@context":          vc.Context,
		"type":              vc.Type,
		"credentialSubject": vc.CredentialSubject,
	}
	if err := token.Set("vc", vcClaim); err != nil {
		return "", fmt.Errorf("set vc claim: %w", err)
	}

	// Sign the JWT with EdDSA
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.EdDSA, iss.jwkKey))
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return string(signed), nil
}

// IssueNonprofitEntity creates and signs a NonprofitEntityCredential.
func (iss *Issuer) IssueNonprofitEntity(subjectDID string, claims *NonprofitEntityClaims) (string, error) {
	claimsMap, err := structToMap(claims)
	if err != nil {
		return "", fmt.Errorf("convert claims to map: %w", err)
	}

	vc := NewVerifiableCredential(TypeNonprofitEntity, iss.DID, subjectDID, claimsMap)
	return iss.Issue(vc)
}

// IssueControlPerson creates and signs a ControlPersonCredential.
func (iss *Issuer) IssueControlPerson(subjectDID string, claims *ControlPersonClaims) (string, error) {
	claimsMap, err := structToMap(claims)
	if err != nil {
		return "", fmt.Errorf("convert claims to map: %w", err)
	}

	vc := NewVerifiableCredential(TypeControlPerson, iss.DID, subjectDID, claimsMap)
	return iss.Issue(vc)
}

// IssueOrganization creates and signs an OrganizationCredential.
func (iss *Issuer) IssueOrganization(subjectDID string, claims *OrganizationClaims) (string, error) {
	claimsMap, err := structToMap(claims)
	if err != nil {
		return "", fmt.Errorf("convert claims to map: %w", err)
	}

	vc := NewVerifiableCredential(TypeOrganization, iss.DID, subjectDID, claimsMap)
	return iss.Issue(vc)
}

// IssueAddress creates and signs an AddressCredential.
func (iss *Issuer) IssueAddress(subjectDID string, claims *AddressClaims) (string, error) {
	claimsMap, err := structToMap(claims)
	if err != nil {
		return "", fmt.Errorf("convert claims to map: %w", err)
	}

	vc := NewVerifiableCredential(TypeAddress, iss.DID, subjectDID, claimsMap)
	return iss.Issue(vc)
}

// structToMap converts a struct to a map[string]interface{} via JSON marshaling.
func structToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
