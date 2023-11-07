# Agapay

`Agapay` is the service that is responsible for the Chariot Payments. It is a microservice that is part of the Chariot platform.

`Agapay` is a form of the word "agape" which means "love" in Greek. It is a form of love that is selfless and unconditional.
It is the highest form of love that is given to others regardless of their merit.
It is a love that is sacrificial and giving. It is the love that God has for us.

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
curl -H "Authorization: Bearer chariot123" http://localhost:8088/accounts | jq .
```
