// Package message implements the Agapay Payment Instruction format,
// an ISO 20022-inspired message structure with privacy-preserving
// public/private data split and hybrid encryption.
package message

// AgapayPaymentInstruction is the complete payment message before splitting.
type AgapayPaymentInstruction struct {
	PublicHeader PublicHeader `json:"public_header"`
	PrivateBody  PrivateBody `json:"private_body"`
}

// PublicHeader contains non-PII payment metadata stored on-chain in SPL Memo.
type PublicHeader struct {
	Version              string `json:"version"`
	MessageID            string `json:"message_id"`
	CreatedAt            string `json:"created_at"`
	SenderDID            string `json:"sender_did"`
	RecipientDID         string `json:"recipient_did"`
	RecipientEIN         string `json:"recipient_ein"`
	TotalAmount          int64  `json:"total_amount"`
	Currency             string `json:"currency"`
	NumberOfTransactions int    `json:"number_of_transactions"`
	PaymentType          string `json:"payment_type"`
	PrivateDataCID       string `json:"private_data_cid"`
	PrivateDataHash      string `json:"private_data_hash"`
}

// PrivateBody contains PII and detailed grant data, encrypted and stored on IPFS.
type PrivateBody struct {
	Transactions   []Transaction `json:"transactions"`
	SenderReference string       `json:"sender_reference,omitempty"`
	RemittanceInfo  string       `json:"remittance_info,omitempty"`
}

// Transaction represents an individual donation line item.
type Transaction struct {
	TransactionID string       `json:"transaction_id,omitempty"`
	Amount        int64        `json:"amount"`
	Currency      string       `json:"currency"`
	Description   string       `json:"description,omitempty"`
	Donation      *Donation    `json:"donation,omitempty"`
	Donors        []Donor      `json:"donors,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`
}

// Donation contains type-specific donation details.
type Donation struct {
	Type             string `json:"type"`
	OrganizationName string `json:"organization_name,omitempty"`
	FundName         string `json:"fund_name,omitempty"`
	Purpose          string `json:"purpose,omitempty"`
	Note             string `json:"note,omitempty"`
	ProgramName      string `json:"program_name,omitempty"`
	CompanyName      string `json:"company_name,omitempty"`
	EmployeeName     string `json:"employee_name,omitempty"`
	AccountName      string `json:"account_name,omitempty"`
}

// Donor contains PII about a donor.
type Donor struct {
	Name    string         `json:"name,omitempty"`
	Email   string         `json:"email,omitempty"`
	Phone   string         `json:"phone,omitempty"`
	Address *PostalAddress `json:"address,omitempty"`
}

// PostalAddress is a physical mailing address.
type PostalAddress struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

// Attachment references a file stored on IPFS.
type Attachment struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	IPFSCID     string `json:"ipfs_cid,omitempty"`
}

// EncryptedEnvelope is the encrypted form of the PrivateBody stored on IPFS.
type EncryptedEnvelope struct {
	Version            string `json:"version"`
	Algorithm          string `json:"algorithm"`
	EphemeralPublicKey string `json:"ephemeral_public_key"`
	Nonce              string `json:"nonce"`
	Ciphertext         string `json:"ciphertext"`
}
