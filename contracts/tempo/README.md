# Agapay Contracts — Tempo

Solidity smart contracts for the Agapay payment network on [Tempo](https://docs.tempo.xyz), a payments-first blockchain with instant deterministic settlement, no native gas token, and TIP-20 stablecoins with built-in memo support.

## Contracts

| Contract | Description |
|----------|-------------|
| `AgapayRegistry` | On-chain nonprofit registry. Issuers register organizations after KYB/KYC. Organizations are keyed by EIN. |
| `AgapayPaymentRouter` | Routes stablecoin payments through TIP-20 `transferWithMemo`, emitting an `AgapayPayment` event with the full public header for off-chain indexing. |

### Testnet Deployments

| Contract | Address | Explorer |
|----------|---------|----------|
| AgapayRegistry | `0xA09f29cc2Eda412DC42dFEc9aF3717a512832Aa2` | [View](https://explore.tempo.xyz/address/0xA09f29cc2Eda412DC42dFEc9aF3717a512832Aa2) |
| AgapayPaymentRouter | `0x5406D2fe5DF84137B687acde7a5409778f9e8491` | [View](https://explore.tempo.xyz/address/0x5406D2fe5DF84137B687acde7a5409778f9e8491) |

**Network:** Tempo Testnet (Moderato) — Chain ID `42431`

## Prerequisites

Install Tempo's Foundry fork (extends standard Foundry with Tempo-specific features):

```bash
# Install foundryup if you don't have it
curl -L https://foundry.paradigm.xyz | bash

# Install the Tempo fork
foundryup -n tempo

# Verify — version should include "-tempo"
forge -V
```

## Build & Test

```bash
# From the repo root
forge build
forge test -vvv
```

## E2E Walkthrough — Tempo Testnet

This walkthrough deploys both contracts and runs the full Agapay flow: register an issuer, onboard a nonprofit, and send a stablecoin payment — all on Tempo testnet.

### 1. Set up environment

```bash
export TEMPO_RPC_URL=https://rpc.moderato.tempo.xyz
export VERIFIER_URL=https://contracts.tempo.xyz
export ALPHA_USD=0x20c0000000000000000000000000000000000001
```

### 2. Create and fund wallets

```bash
# Create deployer/authority wallet
cast wallet new
# → Address: 0x<DEPLOYER>
# → Private key: 0x<DEPLOYER_KEY>

export WALLET_ADDRESS=0x<DEPLOYER>
export PRIVATE_KEY=0x<DEPLOYER_KEY>

# Create a recipient wallet (simulates a nonprofit payment address)
cast wallet new
# → Address: 0x<RECIPIENT>

export RECIPIENT_ADDRESS=0x<RECIPIENT>

# Fund the deployer via Tempo's testnet faucet (1M of each testnet stablecoin)
cast rpc tempo_fundAddress $WALLET_ADDRESS --rpc-url $TEMPO_RPC_URL

# Verify balance
cast erc20 balance $ALPHA_USD $WALLET_ADDRESS --rpc-url $TEMPO_RPC_URL
```

### 3. Deploy contracts

Deploy both contracts with source verification. Tempo has no native gas token, so fees are paid in a TIP-20 stablecoin via `--tempo.fee-token`:

```bash
# Deploy AgapayRegistry
forge create contracts/tempo/src/AgapayRegistry.sol:AgapayRegistry \
  --tempo.fee-token $ALPHA_USD \
  --rpc-url $TEMPO_RPC_URL \
  --private-key $PRIVATE_KEY \
  --broadcast \
  --verify

# → Deployed to: 0x<REGISTRY>
export REGISTRY_ADDRESS=0x<REGISTRY>

# Deploy AgapayPaymentRouter
forge create contracts/tempo/src/AgapayPaymentRouter.sol:AgapayPaymentRouter \
  --tempo.fee-token $ALPHA_USD \
  --rpc-url $TEMPO_RPC_URL \
  --private-key $PRIVATE_KEY \
  --broadcast \
  --verify

# → Deployed to: 0x<ROUTER>
export ROUTER_ADDRESS=0x<ROUTER>
```

Both contracts are automatically verified on [contracts.tempo.xyz](https://contracts.tempo.xyz) and source code is visible in the [Tempo Explorer](https://explore.tempo.xyz).

### 4. Register an issuer

The deployer is the registry owner. Register yourself as an issuer (service provider):

```bash
cast send $REGISTRY_ADDRESS \
  'registerIssuer(address,string)' \
  $WALLET_ADDRESS 'did:web:givechariot.com' \
  --tempo.fee-token $ALPHA_USD \
  --rpc-url $TEMPO_RPC_URL \
  --private-key $PRIVATE_KEY
```

### 5. Register an organization

Register a nonprofit. Only active issuers can call this:

```bash
VC_HASH=$(cast keccak 'test-vc-content')

cast send $REGISTRY_ADDRESS \
  'registerOrganization(string,string,string,string,bytes32,address)' \
  '530196605' \
  'American Red Cross' \
  'redcross.org' \
  'did:web:agapay.redcross.org' \
  $VC_HASH \
  $RECIPIENT_ADDRESS \
  --tempo.fee-token $ALPHA_USD \
  --rpc-url $TEMPO_RPC_URL \
  --private-key $PRIVATE_KEY
```

### 6. Look up the organization

Read the organization data back from the registry:

```bash
cast call $REGISTRY_ADDRESS \
  'getOrganization(string)' '530196605' \
  --rpc-url $TEMPO_RPC_URL
```

This returns the ABI-encoded `Organization` struct with EIN, name, domain, DID URI, issuer address, VC hash, payment address, and timestamps.

### 7. Send a payment

Send a $500 AlphaUSD payment through the router. This uses Tempo's `cast batch-send` to atomically batch the token approval and payment in a single transaction:

```bash
AMOUNT=500000000  # $500 (6 decimals)
MEMO=$(cast keccak 'bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi')
PUBLIC_HEADER='{"version":"1.0","message_id":"msg_001","recipient_ein":"530196605","payment_type":"daf_grant","amount_cents":50000}'

# Encode the two calls
ENCODED_APPROVE=$(cast calldata 'approve(address,uint256)' $ROUTER_ADDRESS $AMOUNT)
ENCODED_SEND=$(cast calldata \
  'sendPayment(address,address,uint256,bytes32,bytes)' \
  $ALPHA_USD $RECIPIENT_ADDRESS $AMOUNT $MEMO \
  $(cast --from-utf8 "$PUBLIC_HEADER"))

# Batch into a single Tempo Transaction (atomic: approve + sendPayment)
cast batch-send \
  --tempo.fee-token $ALPHA_USD \
  --call "$ALPHA_USD::$ENCODED_APPROVE" \
  --call "$ROUTER_ADDRESS::$ENCODED_SEND" \
  --rpc-url $TEMPO_RPC_URL \
  --private-key $PRIVATE_KEY
```

### 8. Verify the payment

```bash
# Check recipient balance
cast erc20 balance $ALPHA_USD $RECIPIENT_ADDRESS --rpc-url $TEMPO_RPC_URL
# → 500000000 ($500)
```

The transaction emits these events:

| # | Event | Source | Data |
|---|-------|--------|------|
| 1 | `Approval` | AlphaUSD | Router approved for $500 |
| 2 | `Transfer` | AlphaUSD | Payer → Router (transferFrom) |
| 3 | `Transfer` | AlphaUSD | Router → Recipient (transferWithMemo) |
| 4 | `TransferWithMemo` | AlphaUSD | 32-byte memo (IPFS CID hash) |
| 5 | `AgapayPayment` | Router | Full JSON public header for indexing |
| 6 | `Transfer` | AlphaUSD | Fee payment to FeeManager |

## Data Durability

The payment data is persisted through three complementary mechanisms:

1. **IPFS** — Full public header stored content-addressed (source of truth)
2. **TIP-20 memo** — `keccak256(IPFS CID)` emitted as a `TransferWithMemo` event (on-chain pointer)
3. **AgapayPayment event** — Full JSON public header emitted for convenient off-chain indexing

## Running the Full E2E via Go CLI

The Go CLI (`cmd/agapay`) runs the complete Agapay flow against live Tempo testnet contracts -- including VC issuance, registry operations, credential verification, encrypted IPFS storage, and stablecoin settlement -- all through the `pkg/` Go libraries.

### 1. Deploy contracts (see steps above)

Use `forge create` to deploy `AgapayRegistry` and `AgapayPaymentRouter` to Tempo testnet.

### 2. Configure

```bash
go run ./cmd/agapay setup \
  --chain tempo \
  --registry 0x<REGISTRY_ADDRESS> \
  --router 0x<ROUTER_ADDRESS> \
  --token 0x20c0000000000000000000000000000000000001 \
  --private-key <HEX_PRIVATE_KEY_WITHOUT_0x>
```

This saves the configuration to `~/.agapay/config.json`. IPFS is optional -- the demo falls back to an in-memory mock if `ipfs daemon` isn't running.

### 3. Run the live demo

```bash
go run ./cmd/agapay live --chain tempo
```

This executes all 11 steps:

1. Generate identity keypairs (Ed25519 for VCs, X25519 for encryption)
2. Issue 4 Verifiable Credentials (Entity, ControlPerson, Organization, Address)
3. Generate DID Document with CNAME-delegated hosting
4. Register Chariot as issuer on Tempo (real transaction, or skip if exists)
5. Register American Red Cross on-chain (real transaction, or skip if exists)
6. Read organization back from registry
7. Verify full credential chain (Entity -> Org -> ControlPerson)
8. Build ISO 20022-inspired payment message (public header + encrypted private body)
9. Encrypt donor PII and pin to IPFS (or mock)
10. Send $500 TIP-20 payment via AgapayPaymentRouter (batched approve + sendPayment)
11. Retrieve and decrypt payment data, verify hash integrity

### 4. Look up an organization

```bash
go run ./cmd/agapay lookup --ein 530196605
```

## Tempo-Specific Features Used

- **No native gas token** — All fees paid in TIP-20 stablecoins (`--tempo.fee-token`)
- **Batch transactions** — `cast batch-send` / Tempo Transactions atomically batch approve + sendPayment (no separate approval tx needed)
- **TIP-20 `transferWithMemo`** — 32-byte on-chain memo attached to every payment transfer
- **Instant finality** — Sub-second deterministic settlement via Simplex BFT consensus
- **Source verification** — Contracts verified on deploy via `--verify` flag
