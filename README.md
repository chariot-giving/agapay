# Agapay

`Agapay` is the service that is responsible for Chariot Payments.

> `Agapay` is a play on the word `agape` which means "love" in Greek. It is a form of love that is selfless and unconditional.
It is the highest form of love that is given to others regardless of their merit.
It is a love that is sacrificial and giving. It is the love that God has for us.

## Overview

There are ~1.5M nonprofits in the US today that don't have an **easy** or **open** method
to accept electronic payments via any `open-network` payment method.

What if you could?

- pay any nonprofit
- instantly
- through the preferred payment method of the nonprofit
- attach data or documents to the payment
- have fraud and compliance checks done automatically
- and transparently track the payment through it's full lifecycle?

In a sense, that's exactly what `Agapay` does.

## How it works

### For the Recipients

Nonprofit entities are the recipients in the `Agapay` network.

To join the network, a nonprofit registers for a free account with `Chariot` and goes through a KYC process.
After they are approved and verified, they are given a bank account with a public-facing, non-sensitive
routing and account number.

This non-sensitive account and routing number is a valid `Agapay` address.

If a nonprofit acts as a fiscal sponsor for subsidiary organizations,
the `Agapay` network allows each subsidiary to exist as a separate recipient with their own `Agapay` address.

Additionally, the `Agapay` network allows for a single recipient to have multiple `Agapay` addresses
if they want to share certain addresses with specific groups of payers or for other reasons like accounting purposes.

Recipients can select their preferred payment method for each `Chariot` hosted `Agapay` address.

![Recipient Onboarding Flow](./docs/assets/recipient_onboarding_flow.png)

### For the Payers

Payers are the individuals or organizations that send payments to the recipients in the `Agapay` network.

To join the network, a payer registers for an account with `Chariot` and goes through a KYC process.
After they are approved and verified, they are given an FBO (For Benefit Of) bank account.

The payer then has 3 options to send payments to recipients:

1. **Send a payment via `Agapay` API to a recipient**
 In this flow, `Chariot` processes the request and pushes the payment from the payers FBO account to the recipient's account.
2. **Read the recipient's `Agapay` address via API and send a payment via their preferred payment method**
 In this flow, the payers read the non-sensitive account and routing numbers and create a payment intent
 for a payment that will be sent via the payers' preferred payment method.
3. **Delegate `Agapay` API access to Accounts Payable systems**
 In this flow, the payers delegate `Agapay` API access to their Accounts Payable systems via an OAuth application.
 The Accounts Payable systems can then read the recipient's `Agapay` address via API and send a payment via their preferred payment method.

![Payer Onboarding Flow](./docs/assets/payer_onboarding_flow.png)

## Getting Started

### Pre-requisites

- [NodeJS](https://nodejs.org/en/download/)
- [Yarn](https://yarnpkg.com/en/docs/install)
- [Docker](https://docs.docker.com/install/)
- [Golang](https://golang.org/doc/install)

### Build

Install project dependencies and build the project with the following command:

```bash
make
```

### Run tests

```bash
make test
```

### Configure environment variables

Copy the `.env.example` file to `.env` and update the values as needed.

```bash
cp .env.example .env
```

### Run locally

First run the following command to spin up the docker containers:

```bash
yarn docker:dev
```

Next, run the following command to initialize the database:

```bash
yarn db:init
```

### Call the API

```bash
curl -H "Authorization: Bearer chariot123" http://localhost:8088/recipients | jq .
```

```json
{
  "data": [
    {
      "id": "e8ff4be1-4603-4bb8-95f3-953c7b95882b",
      "name": "Chariot Giving Network",
      "ein": "931372175",
      "primary": true,
      "created_at": "2023-11-08T23:18:23.858Z"
    }
  ],
  "paging": {
    "cursors": {
      "after": "e8ff4be1-4603-4bb8-95f3-953c7b95882b"
    },
    "total": 1
  }
}
```

## API Reference

### Authorization

The API accepts [Bearer Authentication](https://datatracker.ietf.org/doc/html/rfc6750).
When you sign up for a Chariot account, we make you a pair of API keys:
one for production and one for our sandbox environment in which no real money moves.
You can create and revoke API keys from the dashboard and should securely store them using a secret management system.

### OpenAPI

The `Agapay` API is documented using [OpenAPI](./api/openapi.yaml).
This spec is in beta and subject to change. If you find it useful, or have feedback, let us know!

### Errors

The API uses standard HTTP response codes to indicate the success or failure of requests.
Codes in the 2xx range indicate success; codes in the 4xx and 5xx range indicate errors.
Error objects conform to [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807) and can be distinguished by their type attribute.
Errors will always have the same shape.

```json
{
  "status": 400,
  "type": "Bad Request",
  "title": "failed to create bank account",
  "detail": "your request contains invalid parameters: the name must be at least 3 characters long",
}
```

### Idempotency

The API supports idempotency for safely retrying requests without accidentally performing the same operation twice.
This is useful when an API call is disrupted in transit and you do not receive a response.
For example, if a request to create a payment does not respond due to a network connection error,
you can retry the request with the same idempotency key to guarantee that no more than one payment is created.

To perform an idempotent request, provide an additional, unique `Idempotency-Key` request header per intended request.
We recommend using a V4 UUID. Reusing the key in subsequent requests will return the same response code and body as
the original request along with an additional HTTP header (Idempotent-Replayed: true).
This applies to both success and error responses.
In situations where your request results in a validation error,
you'll need to update your request and retry with a new idempotency key.

Idempotency keys will persist in the API for at least 1 hour.
If an original request is still being processed when an idempotency key is reused, the API will return a
[409 Conflict](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/409) error.
Subsequent requests must be identical to the original request or the API will return a
[422 Unprocessable Entity](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/422) error.
We discourage setting an idempotency key on `GET` and `DELETE` requests as these requests are inherently idempotent.
