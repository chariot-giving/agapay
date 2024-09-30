# How it works

## For the Recipients

To join the network, an `Organization` works with a `Service Provider` (e.g. Chariot).
In order to onboard, the `Service Provider` needs to trust the nonprofit `Entity` and `Organization`.
The `Service Provider` will perform KYB/KYC checks on the `Entity` and `Organization` and issue them
public-facing `Addresses` along with `Verifiable Credentials` attesting to the authenticity of the `Entity`, `Organization`, and `Address`.

In Chariot's case, organizations are issued a `Chariot` bank account
with public-facing, non-sensitive routing and account numbers.

This non-sensitive account and routing number is a valid `US Bank Account Address`.

If a nonprofit entity acts as a fiscal sponsor for subsidiary organizations,
the `Agapay` network allows each subsidiary to exist as a separate `Organization` with their own `Address`.
Note that in this case, the parent entity is responsible for the financial obligations of the subsidiary entities
and therefore issuance of an `Address` is conditional upon the parent entity's approval (part of the KYB/KYC process).

Additionally, the `Agapay` network allows for a single `Organization` to have multiple `Addresses`
if they want to share certain addresses with specific groups of payers or for other reasons
like accounting purposes or fund designations.

Organizations can indicate their preferred `Address` to other payers in the network.

## For the Payers

Payers are the individuals or organizations that send payments to `Organizations` in the `Agapay` network.

While we envision a world where anyone could be a payer and use the network for free, we also recognize that would come with its own set of challenges.
Ultimately, an open network built upon `Verifiable Credentials` still operates in a "triangle of trust" between issuer (Chariot), holder (Nonprofit Organization), and verifier (Payer).
Note that because `Verifiable Credentials` can be created by anyone, the verifier (Payer) needs to decide if they trust the issuer.
For now, it's easier if the verifier has a direct relationship with the issuer (Chariot).

To join the network, a prospective payer registers for an account with `Chariot` and goes through an onboarding process.
After they are approved and verified, they are given API keys.
If we wanted to support end-to-end payment solutions, we would also set up a payment account (FBO) for the payer.

The payer then has 3 options to send payments to `Organizations`:

1. **Read the Organization's Address via API and send a payment via their preferred payment method**
 In this flow, the payers call the API to read the public-facing `Address` identifier
 and create a payment intent for a payment that will be sent via the appropriate payment rail.
 This option is ideal for payers that manage their own Accounts Payable systems or already have capabilities to send payments.
2. **Delegate `Chariot` API access to Accounts Payable systems**
 In this flow, the payers delegate `Chariot` API access to their Accounts Payable systems via an OAuth application.
 The Accounts Payable systems can then read the Organization's `Address` via API and send a payment via the appropriate payment network.
 Note that this method requires Chariot to offer OAuth and for charitable AP systems to have built an integration with the `Chariot` API.
3. **Use Chariot's hosted payment solution**
 In this flow, the payers use Chariot's hosted payment solution to send payments to the `Organizations`.
 This option is ideal for payers that do not want to manage their own Accounts Payable systems and/or want to offload the responsibility of sending payments altogether.
 In the future, `Chariot` could also offer our own Accounts Payable system that is integrated with the `Chariot` API.
