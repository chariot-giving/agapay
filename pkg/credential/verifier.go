package credential

import (
	"crypto/ed25519"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Verifier validates JWT-VC credentials.
type Verifier struct{}

// NewVerifier creates a new VC verifier.
func NewVerifier() *Verifier {
	return &Verifier{}
}

// VerificationResult contains the result of verifying a VC-JWT.
type VerificationResult struct {
	Valid             bool
	IssuerDID         string
	SubjectDID        string
	CredentialType    []string
	CredentialSubject map[string]interface{}
	Error             string
}

// Verify parses and verifies a JWT-VC token using the provided issuer public key.
// It validates the signature, expiration, and extracts the VC claims.
func (v *Verifier) Verify(tokenString string, issuerPublicKey ed25519.PublicKey) (*VerificationResult, error) {
	result := &VerificationResult{}

	// Create JWK from the issuer's public key
	key, err := jwk.FromRaw(issuerPublicKey)
	if err != nil {
		result.Error = fmt.Sprintf("create JWK: %v", err)
		return result, nil
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.EdDSA); err != nil {
		result.Error = fmt.Sprintf("set algorithm: %v", err)
		return result, nil
	}

	// Parse and verify the JWT
	token, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.EdDSA, key), jwt.WithValidate(true))
	if err != nil {
		result.Error = fmt.Sprintf("verify JWT: %v", err)
		return result, nil
	}

	result.IssuerDID = token.Issuer()
	result.SubjectDID = token.Subject()

	// Extract the vc claim
	vcClaim, ok := token.Get("vc")
	if !ok {
		result.Error = "missing 'vc' claim in JWT"
		return result, nil
	}

	vcMap, ok := vcClaim.(map[string]interface{})
	if !ok {
		result.Error = "invalid 'vc' claim format"
		return result, nil
	}

	// Extract type
	if types, ok := vcMap["type"].([]interface{}); ok {
		for _, t := range types {
			if s, ok := t.(string); ok {
				result.CredentialType = append(result.CredentialType, s)
			}
		}
	}

	// Extract credentialSubject
	if subject, ok := vcMap["credentialSubject"].(map[string]interface{}); ok {
		result.CredentialSubject = subject
	}

	result.Valid = true
	return result, nil
}

// ChainVerificationResult contains the result of verifying the full credential trust chain.
type ChainVerificationResult struct {
	Valid            bool
	EntityResult     *VerificationResult
	OrgResult        *VerificationResult
	PersonResult     *VerificationResult
	EntityDID        string
	OrganizationName string
	ControlPerson    string
	Error            string
}

// VerifyControlPersonChain verifies the full trust chain: Entity VC, Organization VC,
// and Control Person VC. It confirms that:
//  1. All three VCs are signed by the same issuer (valid signature)
//  2. The ControlPersonCredential's entityDid matches the Entity VC's subject
//  3. The OrganizationCredential's entityDid matches the Entity VC's subject
//  4. All VCs are valid (not expired, correct signature)
//
// This gives the payer confidence that a verified human officer is associated
// with the organization they are about to pay.
func (v *Verifier) VerifyControlPersonChain(
	entityVCToken string,
	orgVCToken string,
	personVCToken string,
	issuerPublicKey ed25519.PublicKey,
) (*ChainVerificationResult, error) {
	result := &ChainVerificationResult{}

	// Step 1: Verify all three VCs
	entityResult, err := v.Verify(entityVCToken, issuerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("verify entity VC: %w", err)
	}
	result.EntityResult = entityResult
	if !entityResult.Valid {
		result.Error = fmt.Sprintf("entity VC invalid: %s", entityResult.Error)
		return result, nil
	}

	orgResult, err := v.Verify(orgVCToken, issuerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("verify org VC: %w", err)
	}
	result.OrgResult = orgResult
	if !orgResult.Valid {
		result.Error = fmt.Sprintf("organization VC invalid: %s", orgResult.Error)
		return result, nil
	}

	personResult, err := v.Verify(personVCToken, issuerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("verify person VC: %w", err)
	}
	result.PersonResult = personResult
	if !personResult.Valid {
		result.Error = fmt.Sprintf("control person VC invalid: %s", personResult.Error)
		return result, nil
	}

	// Step 2: Verify all VCs share the same issuer
	if entityResult.IssuerDID != orgResult.IssuerDID || entityResult.IssuerDID != personResult.IssuerDID {
		result.Error = fmt.Sprintf("issuer mismatch: entity=%s, org=%s, person=%s",
			entityResult.IssuerDID, orgResult.IssuerDID, personResult.IssuerDID)
		return result, nil
	}

	// Step 3: Verify the entity DID links
	entitySubjectDID := entityResult.SubjectDID

	// The Organization VC's entityDid claim must match the Entity VC's subject
	orgEntityDID, _ := orgResult.CredentialSubject["entityDid"].(string)
	if orgEntityDID == "" {
		orgEntityDID, _ = orgResult.CredentialSubject["entityDID"].(string)
	}
	if orgEntityDID != entitySubjectDID {
		result.Error = fmt.Sprintf("organization entityDid (%s) does not match entity subject (%s)",
			orgEntityDID, entitySubjectDID)
		return result, nil
	}

	// The ControlPerson VC's entityDid claim must match the Entity VC's subject
	personEntityDID, _ := personResult.CredentialSubject["entityDid"].(string)
	if personEntityDID == "" {
		personEntityDID, _ = personResult.CredentialSubject["entityDID"].(string)
	}
	if personEntityDID != entitySubjectDID {
		result.Error = fmt.Sprintf("control person entityDid (%s) does not match entity subject (%s)",
			personEntityDID, entitySubjectDID)
		return result, nil
	}

	// Extract useful fields
	result.EntityDID = entitySubjectDID
	if name, ok := orgResult.CredentialSubject["organizationName"].(string); ok {
		result.OrganizationName = name
	}
	if name, ok := personResult.CredentialSubject["fullName"].(string); ok {
		result.ControlPerson = name
	}

	result.Valid = true
	return result, nil
}

// VerifyWithDIDResolver verifies a JWT-VC by first resolving the issuer's DID
// to obtain their public key, then verifying the JWT signature.
// The resolver function takes a DID string and returns the Ed25519 public key.
func (v *Verifier) VerifyWithDIDResolver(tokenString string, resolver func(did string) (ed25519.PublicKey, error)) (*VerificationResult, error) {
	result := &VerificationResult{}

	// First, parse the JWT without verification to extract the issuer
	token, err := jwt.Parse([]byte(tokenString), jwt.WithVerify(false), jwt.WithValidate(false))
	if err != nil {
		result.Error = fmt.Sprintf("parse JWT: %v", err)
		return result, nil
	}

	issuerDID := token.Issuer()
	if issuerDID == "" {
		result.Error = "missing issuer in JWT"
		return result, nil
	}

	// Resolve the issuer's DID to get their public key
	pubKey, err := resolver(issuerDID)
	if err != nil {
		result.Error = fmt.Sprintf("resolve issuer DID %s: %v", issuerDID, err)
		return result, nil
	}

	// Now verify with the resolved key
	return v.Verify(tokenString, pubKey)
}
