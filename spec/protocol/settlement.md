# Settlement Specification

## Overview

The Agapay settlement layer handles the actual movement of funds between payers and nonprofit organizations. Payments are executed as USDC stablecoin transfers on Solana, with structured payment data attached via SPL Memo.

## Payment Flow

```
1. Payer looks up organization by EIN in the on-chain registry
2. Payer resolves the organization's DID to get encryption keys
3. Payer constructs an AgapayPaymentInstruction
4. Payer splits the instruction into public header and private body
5. Payer encrypts the private body to the recipient's X25519 public key
6. Payer pins the encrypted envelope to IPFS, receives CID
7. Payer sends USDC transfer with SPL Memo containing the public header (including CID)
8. Recipient detects the incoming USDC transfer and reads the SPL Memo
9. Recipient extracts the IPFS CID from the public header
10. Recipient retrieves and decrypts the private body from IPFS
11. Recipient verifies the private_data_hash matches the on-chain commitment
12. Recipient processes the payment data (donor info, grant details, attachments)
```

## USDC on Solana

### Token Details

- **Token**: USDC (USD Coin by Circle)
- **Solana Program**: SPL Token or Token-2022
- **Mint Address (Mainnet)**: `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`
- **Mint Address (Devnet)**: Use a test USDC mint for development
- **Decimals**: 6 (1 USDC = 1,000,000 units)

### Token Accounts

Each organization in the registry has a `usdc_address` field pointing to their USDC Associated Token Account (ATA) on Solana. This account is:

- Created during onboarding by the Service Provider
- Stored in the on-chain OrganizationAccount PDA
- The destination for all USDC payments to that organization

## Transaction Structure

Each Agapay payment is a single Solana transaction containing two instructions:

### Instruction 1: SPL Memo

The SPL Memo program (`MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr`) attaches the public header JSON to the transaction.

**Data**: UTF-8 encoded JSON of the `PublicHeader` (see [messages.md](./messages.md))

The memo is permanently stored on-chain and indexable, allowing anyone to:
- See the payment amount and currency
- Identify the sender and recipient (by DID and EIN)
- Determine the payment type (DAF grant, corporate match, QCD)
- Locate the encrypted private data on IPFS (via CID)
- Verify private data integrity (via hash)

### Instruction 2: SPL Token Transfer

Standard SPL Token transfer of USDC from the payer's token account to the organization's token account.

**Accounts**:
- Source: Payer's USDC ATA
- Destination: Organization's USDC ATA (from registry `usdc_address`)
- Authority: Payer's wallet (signer)

**Amount**: `total_amount` from the public header, converted to USDC minor units (multiply cents by 10,000 for 6 decimal places, i.e., $500.00 = 500,000,000 units)

## Amount Representation

- **Public header**: Amounts in minor currency units (cents). `50000` = $500.00 USD.
- **USDC transfer**: Amounts in token minor units (6 decimals). $500.00 = `500000000` USDC units.
- **Conversion**: `usdc_units = (header_amount / 100) * 1_000_000` or equivalently `header_amount * 10_000`

## Privacy Model

### What is Public (on-chain)

- Payment amount and currency
- Sender DID (payer identity)
- Recipient DID and EIN
- Payment type (e.g., "donor_advised_fund_grant")
- Number of individual transactions/line items
- IPFS CID of encrypted private data
- SHA-256 hash of plaintext private data
- Solana transaction signature and timestamp

### What is Private (encrypted on IPFS)

- Donor names, emails, phone numbers, addresses
- Fund names and account details
- Grant letters and attached documents
- Detailed remittance information
- Sender-specific reference numbers

### Why This Split

- **Regulatory transparency**: Payment amounts and nonprofit identities should be auditable for compliance (BSA/AML).
- **Donor privacy**: Individual donor PII must be protected. Donors have a reasonable expectation that their personal information is not publicly broadcast.
- **Data utility**: The recipient needs full payment details for reconciliation, receipting, and stewardship. The public does not.

## Error Handling

### Insufficient Funds

If the payer's USDC balance is insufficient, the Solana transaction will fail. The encrypted private data on IPFS remains but no payment is recorded on-chain.

### Invalid Recipient

If the organization's `usdc_address` in the registry is invalid or the token account is closed, the transfer will fail. Payers should verify the registry entry is active before sending.

### IPFS Availability

If the encrypted data cannot be pinned to IPFS before sending the payment, the payer should not proceed. The CID must be available in the public header at transaction time.

## Future Considerations

- **Multi-token support**: The settlement layer can be extended to support other stablecoins (USDT, PYUSD) or wrapped assets.
- **Cross-chain**: The registry and payment data format are chain-agnostic. Future implementations could settle on Ethereum L2s, Stellar, or other networks.
- **Streaming payments**: Solana's speed enables streaming/micro-payment use cases for recurring donations.
- **Compliance hooks**: Smart contract-level checks (e.g., OFAC screening) could be added as optional middleware.
