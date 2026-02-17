// Package credential provides Verifiable Credential issuance and verification
// for the Agapay network, using W3C VC Data Model 2.0 with JWT-VC proof format.
package credential

import "time"

// CredentialType identifies the type of Agapay credential.
type CredentialType string

const (
	TypeNonprofitEntity CredentialType = "NonprofitEntityCredential"
	TypeControlPerson   CredentialType = "ControlPersonCredential"
	TypeOrganization    CredentialType = "OrganizationCredential"
	TypeAddress         CredentialType = "AddressCredential"
)

// VerifiableCredential represents a W3C Verifiable Credential.
type VerifiableCredential struct {
	Context           []string               `json:"@context"`
	ID                string                 `json:"id,omitempty"`
	Type              []string               `json:"type"`
	Issuer            string                 `json:"issuer"`
	IssuanceDate      string                 `json:"issuanceDate"`
	ExpirationDate    string                 `json:"expirationDate,omitempty"`
	CredentialSubject map[string]interface{} `json:"credentialSubject"`
}

// NewVerifiableCredential creates a new VC with standard fields populated.
func NewVerifiableCredential(vcType CredentialType, issuerDID string, subjectDID string, claims map[string]interface{}) *VerifiableCredential {
	subject := make(map[string]interface{})
	subject["id"] = subjectDID
	for k, v := range claims {
		subject[k] = v
	}

	return &VerifiableCredential{
		Context: []string{
			"https://www.w3.org/ns/credentials/v2",
			"https://w3id.org/security/suites/ed25519-2020/v1",
		},
		Type:              []string{"VerifiableCredential", string(vcType)},
		Issuer:            issuerDID,
		IssuanceDate:      time.Now().UTC().Format(time.RFC3339),
		CredentialSubject: subject,
	}
}

// NonprofitEntityClaims contains the claims for a NonprofitEntityCredential.
type NonprofitEntityClaims struct {
	EIN                string        `json:"ein"`
	LegalName          string        `json:"legalName"`
	PhysicalAddress    PostalAddress `json:"physicalAddress"`
	IRSSubsectionCode  string        `json:"irsSubsectionCode,omitempty"`
	IRSPub78           bool          `json:"irsPub78"`
	IRSRevocation      bool          `json:"irsRevocation"`
	OFACList           bool          `json:"ofacList"`
	Incorporation      Incorporation `json:"incorporation,omitempty"`
}

// PostalAddress represents a mailing address.
type PostalAddress struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

// Incorporation contains incorporation details.
type Incorporation struct {
	Date         string `json:"incorporationDate"`
	State        string `json:"incorporationState"`
	Jurisdiction string `json:"incorporationJurisdiction"`
}

// ControlPersonClaims contains the claims for a ControlPersonCredential.
type ControlPersonClaims struct {
	FullName  string `json:"fullName"`
	Title     string `json:"title"`
	Email     string `json:"email"`
	EntityDID string `json:"entityDid"`
	EntityEIN string `json:"entityEin"`
	Role      string `json:"role"` // officer, director, trustee
}

// OrganizationClaims contains the claims for an OrganizationCredential.
type OrganizationClaims struct {
	OrganizationName string `json:"organizationName"`
	Domain           string `json:"domain"`
	EntityDID        string `json:"entityDid"`
	EntityEIN        string `json:"entityEin"`
	Affiliation      string `json:"affiliation"`
	NTEECode         string `json:"nteeCode,omitempty"`
	MissionStatement string `json:"missionStatement,omitempty"`
}

// AddressClaims contains the claims for an AddressCredential.
type AddressClaims struct {
	OrganizationDID        string   `json:"organizationDid"`
	OrganizationEIN        string   `json:"organizationEin"`
	AddressType            string   `json:"addressType"` // us_bank_account, solana_wallet, postal
	SupportedPaymentMethods []string `json:"supportedPaymentMethods"`
	SolanaWallet           string   `json:"solanaWallet,omitempty"`
	USBankAccount          *USBankAccount `json:"usBankAccount,omitempty"`
}

// USBankAccount represents a US bank account address.
type USBankAccount struct {
	AccountNumber string `json:"accountNumber"`
	RoutingNumber string `json:"routingNumber"`
}
