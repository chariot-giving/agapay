# Registry Specification

## Overview

The Agapay on-chain registry is a Solana program that maintains a public, permissionless-read index of verified nonprofit organizations and their payment addresses. It serves as the decentralized "phone book" of the charitable payment network.

## Design Goals

1. **Public and free**: Anyone can read the registry without authentication or fees (beyond Solana rent).
2. **Issuer-gated writes**: Only registered Service Providers (issuers) can create or update organization records.
3. **One record per EIN**: Deterministic Program Derived Addresses (PDAs) seeded by EIN prevent duplicates.
4. **VC-anchored**: Each organization record stores a hash of its associated Verifiable Credentials for on-chain integrity verification.

## Program ID

The Agapay Registry Program is deployed on Solana and identified by its program ID (generated at deployment time).

## Account Structures

### IssuerAccount

Represents a registered Service Provider authorized to write organization records.

**PDA seed**: `["issuer", issuer_authority_pubkey]`

| Field | Type | Size | Description |
|-------|------|------|-------------|
| `authority` | Pubkey | 32 | Signing authority for this issuer |
| `did` | String | 4 + 128 | DID URI of the issuer (e.g., `did:web:givechariot.com`) |
| `active` | bool | 1 | Whether this issuer is currently authorized |
| `created_at` | i64 | 8 | Unix timestamp of registration |

**Total size**: 8 (discriminator) + 32 + 132 + 1 + 8 = 181 bytes

### OrganizationAccount

Represents a verified nonprofit organization in the registry.

**PDA seed**: `["organization", ein_string]`

| Field | Type | Size | Description |
|-------|------|------|-------------|
| `ein` | String | 4 + 9 | Employer Identification Number |
| `name` | String | 4 + 128 | Organization name |
| `domain` | String | 4 + 128 | Verified web domain |
| `did_uri` | String | 4 + 128 | Organization's DID URI |
| `issuer` | Pubkey | 32 | Pubkey of the issuer who registered this org |
| `vc_hash` | [u8; 32] | 32 | SHA-256 hash of the issued VCs |
| `usdc_address` | Pubkey | 32 | USDC SPL token account for receiving payments |
| `active` | bool | 1 | Whether this organization is currently active |
| `created_at` | i64 | 8 | Unix timestamp of creation |
| `updated_at` | i64 | 8 | Unix timestamp of last update |

**Total size**: 8 (discriminator) + 13 + 132 + 132 + 132 + 32 + 32 + 32 + 1 + 8 + 8 = 530 bytes

## Instructions

### register_issuer

Registers a new Service Provider as an authorized issuer.

**Authorization**: Only the program authority (upgrade authority) can call this instruction.

**Parameters**:
- `did: String` -- The issuer's DID URI

**Accounts**:
- `authority` (signer) -- Program upgrade authority
- `issuer_account` (init, PDA) -- New IssuerAccount to create
- `issuer_authority` -- The pubkey that will sign on behalf of this issuer
- `system_program`

### register_organization

Creates a new organization record in the registry.

**Authorization**: The caller must be a registered, active issuer.

**Parameters**:
- `ein: String` -- 9-digit EIN
- `name: String` -- Organization name
- `domain: String` -- Verified web domain
- `did_uri: String` -- Organization's DID URI
- `vc_hash: [u8; 32]` -- SHA-256 hash of the VC bundle
- `usdc_address: Pubkey` -- USDC token account for payments

**Accounts**:
- `issuer_authority` (signer) -- Must match a registered issuer
- `issuer_account` (PDA) -- The issuer's IssuerAccount
- `organization_account` (init, PDA) -- New OrganizationAccount to create
- `system_program`

**Validations**:
- EIN must be exactly 9 characters
- Issuer must be active
- Organization PDA must not already exist (enforced by `init`)

### update_organization

Updates an existing organization record.

**Authorization**: The caller must be the same issuer who registered the organization.

**Parameters**:
- `name: Option<String>` -- Updated name
- `domain: Option<String>` -- Updated domain
- `did_uri: Option<String>` -- Updated DID URI
- `vc_hash: Option<[u8; 32]>` -- Updated VC hash
- `usdc_address: Option<Pubkey>` -- Updated USDC address

**Accounts**:
- `issuer_authority` (signer)
- `issuer_account` (PDA)
- `organization_account` (mut, PDA)

### deactivate_organization

Marks an organization as inactive in the registry.

**Authorization**: The caller must be the same issuer who registered the organization.

**Parameters**: None (EIN derived from the PDA seed)

**Accounts**:
- `issuer_authority` (signer)
- `issuer_account` (PDA)
- `organization_account` (mut, PDA)

## Querying the Registry

All accounts are readable by anyone via Solana RPC:

- **By EIN**: Derive the PDA from `["organization", ein]` and fetch the account.
- **By issuer**: Use `getProgramAccounts` with a `memcmp` filter on the `issuer` field.
- **All active**: Use `getProgramAccounts` with a `memcmp` filter on the `active` field.

## Security Considerations

- **Issuer authorization**: Only the program authority can register issuers, preventing unauthorized writes.
- **PDA uniqueness**: EIN-seeded PDAs make it impossible to create duplicate records for the same entity.
- **Immutable issuer link**: The `issuer` field on an OrganizationAccount is set at creation and only that issuer can update the record.
- **VC integrity**: The `vc_hash` allows anyone to verify that off-chain VCs match the on-chain commitment.
