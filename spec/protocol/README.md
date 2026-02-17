# Agapay Protocol Specification

This directory contains the protocol specifications for the Agapay charitable payment network.

## Documents

- [Identity](./identity.md) -- DID methods, CNAME-delegated DID hosting, and Verifiable Credential schemas
- [Registry](./registry.md) -- On-chain nonprofit registry on Solana
- [Messages](./messages.md) -- ISO 20022-inspired payment message format with privacy-preserving data split
- [Settlement](./settlement.md) -- Stablecoin (USDC) settlement on Solana with SPL Memo

## Design Principles

1. **Open and Public** -- The registry and payment amounts are fully transparent on-chain.
2. **Privacy-Preserving** -- Donor PII is encrypted and stored off-chain; only the intended recipient can decrypt.
3. **Self-Sovereign** -- Nonprofits own their identifiers and can migrate between Service Providers.
4. **Interoperable** -- Payment messages align with ISO 20022 concepts for future compatibility with traditional financial systems.
5. **Separation of Concerns** -- Identity, registry, data, and settlement are independent layers.
