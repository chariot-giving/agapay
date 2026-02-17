# Payment Message Specification

## Overview

The Agapay Payment Message format (`AgapayPaymentInstruction`) defines a standardized structure for charitable payment data. It is inspired by [ISO 20022](https://www.iso20022.org/) `pain.001` (Customer Credit Transfer Initiation) and adapted for the unique requirements of charitable payments: donor privacy, grant data, and multiple donation types.

## ISO 20022 Alignment

The message structure maps to ISO 20022 concepts while using JSON instead of XML:

| ISO 20022 Element | ISO Tag | Agapay Field | Description |
|---|---|---|---|
| Group Header | `GrpHdr` | `header` | Message ID, timestamp, transaction count |
| Payment Information | `PmtInf` | `header.sender_did` | Payer identification |
| Credit Transfer Transaction | `CdtTrfTxInf` | `private_body.transactions[]` | Individual line items |
| Remittance Information | `RmtInf` | `private_body.transactions[].donation` | Grant/donation details |
| Creditor | `Cdtr` | `header.recipient_did` | Recipient identification |
| Amount | `Amt` | `header.total_amount` | Payment amount |

## Public vs. Private Data Split

Every `AgapayPaymentInstruction` is split into two parts to preserve donor privacy while maintaining on-chain transparency for amounts and payment metadata.

### Public Header

Stored on-chain in the Solana SPL Memo. Visible to everyone. Contains no PII.

```json
{
  "version": "1.0",
  "message_id": "msg_01j8rsfpswhg03ngse0pkscr3n",
  "created_at": "2026-02-16T12:00:00Z",
  "sender_did": "did:web:givechariot.com:payers:vanguard",
  "recipient_did": "did:web:agapay.redcross.org",
  "recipient_ein": "530196605",
  "total_amount": 50000,
  "currency": "USD",
  "number_of_transactions": 2,
  "payment_type": "donor_advised_fund_grant",
  "private_data_cid": "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi",
  "private_data_hash": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | yes | Protocol version (currently "1.0") |
| `message_id` | string | yes | Unique message identifier |
| `created_at` | string (ISO 8601) | yes | Message creation timestamp |
| `sender_did` | string | yes | DID of the sending entity |
| `recipient_did` | string | yes | DID of the recipient organization |
| `recipient_ein` | string | yes | EIN of the recipient entity |
| `total_amount` | integer | yes | Total amount in minor currency units (cents) |
| `currency` | string | yes | ISO 4217 currency code (e.g., "USD") |
| `number_of_transactions` | integer | yes | Count of individual donation line items |
| `payment_type` | string | yes | Primary donation type |
| `private_data_cid` | string | yes | IPFS CID of the encrypted private body |
| `private_data_hash` | string | yes | SHA-256 hash of the plaintext private body (for integrity verification) |

### Private Body

Encrypted and stored on IPFS. Only the recipient nonprofit can decrypt. Contains all PII and detailed grant data.

```json
{
  "transactions": [
    {
      "transaction_id": "tx_01",
      "amount": 25000,
      "currency": "USD",
      "description": "General Operating Support",
      "donation": {
        "type": "donor_advised_fund_grant",
        "organization_name": "Vanguard Charitable",
        "fund_name": "John Doe Giving Fund",
        "purpose": "General Operating Support",
        "note": "Annual grant from the Doe family"
      },
      "donors": [
        {
          "name": "John Doe",
          "email": "john.doe@example.com",
          "phone": "415-555-1212",
          "address": {
            "line1": "123 Main St",
            "city": "San Francisco",
            "state": "CA",
            "postalCode": "94105",
            "country": "US"
          }
        }
      ],
      "attachments": [
        {
          "type": "grant_letter",
          "description": "Annual grant letter",
          "ipfs_cid": "bafybeif..."
        }
      ]
    }
  ],
  "sender_reference": "VNG-2026-00142",
  "remittance_info": "DAF Grant - Q1 2026 Distribution"
}
```

## Donation Types

| Type | Description |
|------|-------------|
| `donor_advised_fund_grant` | Grant from a Donor-Advised Fund |
| `corporate_match` | Corporate matching gift program |
| `qualified_charitable_distribution` | Distribution from an IRA |

Each donation type has type-specific fields in the `donation` object:

### donor_advised_fund_grant
- `organization_name` -- Name of the DAF sponsor (e.g., "Vanguard Charitable")
- `fund_name` -- Name of the specific fund
- `purpose` -- Grant purpose
- `note` -- Donor note

### corporate_match
- `program_name` -- Name of the matching program
- `company_name` -- Employer name
- `employee_name` -- Employee who triggered the match

### qualified_charitable_distribution
- `account_name` -- Name on the IRA account
- `organization_name` -- Custodian name

## Encryption Scheme

### Key Discovery

The payer resolves the recipient's DID and extracts the X25519 `keyAgreement` public key:

1. Parse `recipient_did` (e.g., `did:web:agapay.redcross.org`)
2. Resolve the DID Document via HTTPS: `GET https://agapay.redcross.org/.well-known/did.json`
3. Extract the `keyAgreement` verification method of type `X25519KeyAgreementKey2020`
4. Decode the `publicKeyMultibase` value to obtain the raw X25519 public key

### Hybrid Encryption (ECDH + AES-256-GCM)

1. **Generate ephemeral keypair**: Create a random X25519 keypair (ephemeral private key, ephemeral public key)
2. **Key agreement**: Compute ECDH shared secret: `shared_secret = X25519(ephemeral_private, recipient_public)`
3. **Key derivation**: Derive a 256-bit symmetric key via HKDF-SHA256 with info string `"agapay-payment-v1"`
4. **Encrypt**: Encrypt the private body JSON bytes with AES-256-GCM using the derived key and a random 12-byte nonce
5. **Package**: Create an encrypted envelope containing:
   - `ephemeral_public_key` (32 bytes, base64)
   - `nonce` (12 bytes, base64)
   - `ciphertext` (variable length, base64)

### Encrypted Envelope Format

```json
{
  "version": "1.0",
  "algorithm": "ECDH-ES+AES256GCM",
  "ephemeral_public_key": "base64...",
  "nonce": "base64...",
  "ciphertext": "base64..."
}
```

### Decryption

1. Recipient retrieves the encrypted envelope from IPFS using the `private_data_cid`
2. Computes ECDH: `shared_secret = X25519(recipient_private, ephemeral_public)`
3. Derives the symmetric key via HKDF-SHA256 with the same info string
4. Decrypts the ciphertext with AES-256-GCM
5. Computes SHA-256 of the decrypted plaintext and verifies it matches `private_data_hash` from the on-chain public header
6. Parses the JSON private body

### Security Properties

- **Confidentiality**: Only the holder of the recipient's X25519 private key can decrypt.
- **Forward secrecy**: Each payment uses an ephemeral keypair; compromising the recipient's long-term key does not reveal past payments (if ephemeral keys are properly discarded).
- **Integrity**: The `private_data_hash` on-chain ensures the decrypted data matches the payer's original commitment.
- **Authenticity**: The on-chain transaction is signed by the payer's Solana key, providing non-repudiation.
