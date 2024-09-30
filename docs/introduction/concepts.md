# Concepts

`Agapay` is a network for charitable payments.

## Entity

An `Entity` represents a tax-exempt (as defined by the IRS) nonprofit or a fiscal sponsor for a charitable organization.
An entity is a legal entity with a name, EIN (Federal Tax ID), officers, and a physical address.
There are approximately 1.5M nonprofits in the US.

## Organization

An `Organization` represents an operating entity or an operating sub-organization of a parent or sponsoring entity.
Organizations can receive payments from payers on the network.
Organizations are associated with a web domain, logo, and a set of verified addresses where they can receive payments.

## Address

A `Address` is a public-facing identifier that can be used to send payments to an `Organization`.
An Organization is said to be "addressable" if it has at least one `Address`.
An Address can take many forms depending on the type of payment rail used:

### Postal Address

A postal address is a mailing address where physical mail can be received.
Postal addresses are used to send paper checks.

### US Bank Account

A US bank account is a financial account that can be used to securely receive electronic payments.
A bank account address is usually composed of an account number and a routing number.
The routing number is the American Bankers' Association (ABA) Routing Transit Number (RTN).
The account number is an identifier specific to the receiving bank that uniquely identifies an account that can receive payments.
Bank account addresses (account and routing numbers) can be used to send ACH, RTP, wire transfers, and FedNow payments.

## Service Provider

A `Service Provider` (or `Issuer`) is a 3rd-party organization that issues verifiable `Addresses` to nonprofit `Organizations`.
Chariot is the first `Service Provider` to issue `Addresses` to nonprofits.
This doesn't necessarily mean Chariot will be the only `Service Provider` in the network.

## Verifiable Credential

A `Verifiable Credential` or `VC` is a digital credential that is issued to an `Organization` by a `Service Provider`.
The `VC` proves authenticity and validity of an `Entity`, `Organization`, or `Address` and allows anyone to instantaneously verify the information presented.
For more information about `VC`s, see the [W3C Verifiable Credential specification](https://www.w3.org/TR/vc-data-model/).
