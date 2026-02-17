# Identity Specification

## Overview

The Agapay identity layer provides verifiable, decentralized identifiers and credentials for all participants in the charitable payment network. It is built on W3C standards: [Decentralized Identifiers (DIDs)](https://www.w3.org/TR/did-core/) and [Verifiable Credentials (VCs)](https://www.w3.org/TR/vc-data-model-2.0/).

## DID Methods

### Issuers (Service Providers)

Service Providers use `did:web` anchored to their corporate domain.

- Example: `did:web:givechariot.com`
- Resolves to: `https://givechariot.com/.well-known/did.json`
- Trust anchor: The Service Provider's DNS-verified domain

The issuer DID Document includes Ed25519 verification keys used to sign Verifiable Credentials.

### Organizations (Holders)

Nonprofit organizations use `did:web` with **CNAME-delegated hosting** through their Service Provider.

- Example: `did:web:agapay.redcross.org`
- Resolves to: `https://agapay.redcross.org/.well-known/did.json`
- Trust anchor: The nonprofit's DNS domain

See [CNAME-Delegated DID Hosting](#cname-delegated-did-hosting) below.

### Control Persons

Individual officers and control persons use `did:key` for lightweight, infrastructure-free identifiers.

- Example: `did:key:z6Mkf5rGMoatrSj1f...`
- No hosting required; the public key is encoded directly in the DID
- Used as subjects for `ControlPersonCredential`

## CNAME-Delegated DID Hosting

### Problem

Nonprofits should not need to run DID infrastructure. Managing cryptographic keys, hosting DID documents, and provisioning TLS certificates is an unreasonable burden for charitable organizations focused on their mission.

### Solution

Chariot (or any Service Provider) hosts DID documents on behalf of nonprofits using DNS subdomain delegation. This is analogous to how email works with MX records or how CDNs work with CNAME records.

### Flow

1. **Domain verification**: During onboarding, the Service Provider verifies the nonprofit owns their domain (e.g., `redcross.org`) via a DNS TXT record:
   ```
   _agapay-verify.redcross.org.  TXT  "agapay-verify=abc123xyz789"
   ```

2. **CNAME delegation**: The nonprofit adds a single DNS CNAME record:
   ```
   agapay.redcross.org.  CNAME  dids.givechariot.com.
   ```

3. **TLS provisioning**: The Service Provider provisions a TLS certificate for `agapay.redcross.org` via Let's Encrypt (HTTP-01 or DNS-01 challenge).

4. **DID resolution**: The nonprofit's DID `did:web:agapay.redcross.org` resolves per the `did:web` specification to `https://agapay.redcross.org/.well-known/did.json`, which is served by the Service Provider's infrastructure through the CNAME.

5. **Key management**: The Service Provider generates and manages the nonprofit's key pairs:
   - **Ed25519** for VC verification (`verificationMethod`, `authentication`)
   - **X25519** for encryption (`keyAgreement`) -- used by payers to encrypt private payment data

### DID Document Structure

```json
{
  "@context": [
    "https://www.w3.org/ns/did/v1",
    "https://w3id.org/security/suites/ed25519-2020/v1"
  ],
  "id": "did:web:agapay.redcross.org",
  "verificationMethod": [
    {
      "id": "did:web:agapay.redcross.org#key-1",
      "type": "Ed25519VerificationKey2020",
      "controller": "did:web:agapay.redcross.org",
      "publicKeyMultibase": "z6Mkf5rGMoatrSj1f..."
    },
    {
      "id": "did:web:agapay.redcross.org#key-enc-1",
      "type": "X25519KeyAgreementKey2020",
      "controller": "did:web:agapay.redcross.org",
      "publicKeyMultibase": "z6LSbysY2xFMRpGMhb..."
    }
  ],
  "authentication": ["did:web:agapay.redcross.org#key-1"],
  "keyAgreement": ["did:web:agapay.redcross.org#key-enc-1"],
  "service": [
    {
      "id": "did:web:agapay.redcross.org#agapay",
      "type": "AgapayEndpoint",
      "serviceEndpoint": "https://api.givechariot.com/v1/organizations/org_..."
    }
  ]
}
```

### Trust and Self-Sovereignty Properties

- The DID is anchored to the **nonprofit's domain**, not the Service Provider's domain.
- The nonprofit can **revoke delegation** by removing the CNAME record and self-hosting.
- The nonprofit can **migrate** to a different Service Provider by changing the CNAME target.
- The Service Provider cannot impersonate the nonprofit without DNS control.

## Verifiable Credential Types

All credentials use W3C VC Data Model 2.0 with JWT-VC proof format (JWS with Ed25519).

### NonprofitEntityCredential

Attests to the existence and compliance status of a tax-exempt entity.

| Field | Type | Description |
|-------|------|-------------|
| `ein` | string | IRS Employer Identification Number (9 digits) |
| `legalName` | string | IRS-registered legal name |
| `physicalAddress` | object | Registered physical address |
| `irsSubsectionCode` | string | IRS subsection code (e.g., "03" for 501(c)(3)) |
| `irsPub78` | boolean | Listed on IRS Publication 78 |
| `irsRevocation` | boolean | Listed on IRS revocation list |
| `ofacList` | boolean | Listed on OFAC sanctions list |
| `incorporation` | object | Incorporation date, state, and jurisdiction |

### ControlPersonCredential

Attests to a control person's identity and relationship to an entity.

| Field | Type | Description |
|-------|------|-------------|
| `fullName` | string | Full legal name of the person |
| `title` | string | Title or role (e.g., "Executive Director") |
| `email` | string | Contact email address |
| `entityDid` | string | DID of the associated entity |
| `entityEin` | string | EIN of the associated entity |
| `role` | string | Role type: `officer`, `director`, `trustee` |

### OrganizationCredential

Attests to an operating organization and its relationship to a parent entity.

| Field | Type | Description |
|-------|------|-------------|
| `organizationName` | string | Operating name of the organization |
| `domain` | string | Verified web domain |
| `entityDid` | string | DID of the parent/sponsoring entity |
| `entityEin` | string | EIN of the parent/sponsoring entity |
| `affiliation` | string | Relationship type (see below) |
| `nteeCode` | string | NTEE classification code |
| `missionStatement` | string | Organization's mission statement |

Affiliation values: `independent_nonprofit`, `fiscal_sponsor`, `parent_organization`, `sponsored_organization`.

### AddressCredential

Attests to a verified payment address for an organization.

| Field | Type | Description |
|-------|------|-------------|
| `organizationDid` | string | DID of the organization |
| `organizationEin` | string | EIN of the organization's entity |
| `addressType` | string | `us_bank_account`, `solana_wallet`, `postal` |
| `supportedPaymentMethods` | array | Payment methods: `ach`, `rtp`, `wire`, `fednow`, `usdc` |
| `solanaWallet` | string | Solana public key (base58, for `solana_wallet` type) |
| `usBankAccount` | object | Account + routing number (for `us_bank_account` type) |

## Credential Lifecycle

1. **Issuance**: Service Provider performs KYB/KYC, then signs and issues VCs to the organization.
2. **Storage**: VCs are stored by the holder and their hash is anchored on-chain in the registry.
3. **Presentation**: The holder presents VCs to verifiers (payers) who validate the JWT signature against the issuer's DID.
4. **Revocation**: Service Provider can update or revoke VCs. On-chain registry status reflects active/inactive state.
