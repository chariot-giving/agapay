// Package tempo implements the Agapay Registry interface for Tempo (EVM).
// It interacts with the AgapayRegistry Solidity contract using the tempo-go
// SDK and Tempo Transactions (Type 0x76).
package tempo

import (
	"context"
	"fmt"
	"math/big"

	"github.com/chariot-giving/agapay/pkg/chain"
	"github.com/chariot-giving/agapay/pkg/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tempoxyz/tempo-go/pkg/client"
	"github.com/tempoxyz/tempo-go/pkg/signer"
	"github.com/tempoxyz/tempo-go/pkg/transaction"
)

// Compile-time interface check.
var _ registry.Registry = (*Client)(nil)

// Client interacts with the AgapayRegistry contract on Tempo.
type Client struct {
	rpc             *client.Client
	registryAddress common.Address
	chainID         int64
	id              chain.ID
	ownerSigner     *signer.Signer
}

// NewClient creates a new Tempo registry client.
func NewClient(rpcURL string, registryAddr common.Address, chainID int64, id chain.ID, opts ...client.Option) *Client {
	return &Client{
		rpc:             client.New(rpcURL, opts...),
		registryAddress: registryAddr,
		chainID:         chainID,
		id:              id,
	}
}

// WithOwnerSigner sets the default owner signer used for issuer registration.
func (c *Client) WithOwnerSigner(s *signer.Signer) *Client {
	c.ownerSigner = s
	return c
}

func (c *Client) ChainID() chain.ID {
	return c.id
}

// GetOrganization reads an organization from the AgapayRegistry contract.
func (c *Client) GetOrganization(ctx context.Context, ein string) (*registry.Organization, error) {
	callData := encodeGetOrganization(ein)

	resp, err := c.rpc.SendRequest(ctx, "eth_call", map[string]string{
		"to":   c.registryAddress.Hex(),
		"data": "0x" + common.Bytes2Hex(callData),
	}, "latest")
	if err != nil {
		return nil, fmt.Errorf("eth_call getOrganization: %w", err)
	}
	if err := resp.CheckError(); err != nil {
		return nil, fmt.Errorf("getOrganization reverted: %w", err)
	}

	resultHex, ok := resp.Result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	return decodeOrganization(resultHex)
}

// GetIssuer reads an issuer from the AgapayRegistry contract.
func (c *Client) GetIssuer(ctx context.Context, authority chain.Address) (*registry.Issuer, error) {
	addr := common.HexToAddress(string(authority))
	callData := encodeGetIssuer(addr)

	resp, err := c.rpc.SendRequest(ctx, "eth_call", map[string]string{
		"to":   c.registryAddress.Hex(),
		"data": "0x" + common.Bytes2Hex(callData),
	}, "latest")
	if err != nil {
		return nil, fmt.Errorf("eth_call getIssuer: %w", err)
	}
	if err := resp.CheckError(); err != nil {
		return nil, fmt.Errorf("getIssuer reverted: %w", err)
	}

	resultHex, ok := resp.Result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	return decodeIssuer(resultHex)
}

// RegisterIssuer registers a new issuer on the AgapayRegistry contract.
// The AuthoritySigner must be the contract owner.
func (c *Client) RegisterIssuer(ctx context.Context, params registry.RegisterIssuerParams) (chain.TxHash, error) {
	sgn, err := c.signerFromKey(params.AuthoritySigner)
	if err != nil {
		return "", fmt.Errorf("create signer: %w", err)
	}

	issuerAddr := common.HexToAddress(string(params.IssuerAuthority))
	callData := encodeRegisterIssuer(issuerAddr, params.DID)

	return c.sendTx(ctx, sgn, callData, 500_000)
}

// RegisterOrganization registers a new organization on the AgapayRegistry contract.
func (c *Client) RegisterOrganization(ctx context.Context, params registry.RegisterOrgParams) (chain.TxHash, error) {
	sgn, err := c.signerFromKey(params.IssuerSigner)
	if err != nil {
		return "", fmt.Errorf("create signer: %w", err)
	}

	paymentAddr := common.HexToAddress(string(params.PaymentAddress))
	callData := encodeRegisterOrganization(
		params.EIN, params.Name, params.Domain, params.DIDURI,
		params.VCHash, paymentAddr,
	)

	return c.sendTx(ctx, sgn, callData, 800_000)
}

// DeactivateOrganization deactivates an organization on the AgapayRegistry contract.
func (c *Client) DeactivateOrganization(ctx context.Context, params registry.DeactivateOrgParams) (chain.TxHash, error) {
	sgn, err := c.signerFromKey(params.IssuerSigner)
	if err != nil {
		return "", fmt.Errorf("create signer: %w", err)
	}

	callData := encodeDeactivateOrganization(params.EIN)

	return c.sendTx(ctx, sgn, callData, 200_000)
}

// RegistryAddress returns the contract address for external use.
func (c *Client) RegistryAddress() common.Address {
	return c.registryAddress
}

// RPCClient returns the underlying Tempo RPC client.
func (c *Client) RPCClient() *client.Client {
	return c.rpc
}

// sendTx builds, signs, and sends a Tempo Transaction (Type 0x76).
func (c *Client) sendTx(ctx context.Context, sgn *signer.Signer, callData []byte, gas uint64) (chain.TxHash, error) {
	nonce, err := c.rpc.GetTransactionCount(ctx, sgn.Address().Hex())
	if err != nil {
		return "", fmt.Errorf("get nonce: %w", err)
	}

	registryAddr := c.registryAddress
	tx := transaction.NewDefault(c.chainID)
	tx.Nonce = nonce
	tx.Gas = gas
	tx.MaxFeePerGas = big.NewInt(25_000_000_000)
	tx.MaxPriorityFeePerGas = big.NewInt(2_000_000_000)
	tx.Calls = []transaction.Call{{
		To:    &registryAddr,
		Value: big.NewInt(0),
		Data:  callData,
	}}

	if err := transaction.SignTransaction(tx, sgn); err != nil {
		return "", fmt.Errorf("sign transaction: %w", err)
	}

	serialized, err := transaction.Serialize(tx, nil)
	if err != nil {
		return "", fmt.Errorf("serialize transaction: %w", err)
	}

	hash, err := c.rpc.SendRawTransaction(ctx, serialized)
	if err != nil {
		return "", fmt.Errorf("send transaction: %w", err)
	}

	return chain.TxHash(hash), nil
}

func (c *Client) signerFromKey(key []byte) (*signer.Signer, error) {
	hexKey := "0x" + common.Bytes2Hex(key)
	return signer.NewSigner(hexKey)
}
