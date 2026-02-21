package solana

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/chariot-giving/agapay/pkg/chain"
	"github.com/chariot-giving/agapay/pkg/registry"
	solanago "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Compile-time interface check.
var _ registry.Registry = (*Client)(nil)

// Client interacts with the Agapay Registry Solana program.
type Client struct {
	rpcClient *rpc.Client
	programID solanago.PublicKey
	wsURL     string
	chainID   chain.ID
}

// NewClient creates a new Solana registry client.
func NewClient(rpcURL string, wsURL string, programID solanago.PublicKey, id chain.ID) *Client {
	return &Client{
		rpcClient: rpc.New(rpcURL),
		programID: programID,
		wsURL:     wsURL,
		chainID:   id,
	}
}

// ChainID returns the chain identifier for this client.
func (c *Client) ChainID() chain.ID {
	return c.chainID
}

// GetOrganization fetches an organization account by EIN and returns
// the chain-agnostic Organization type.
func (c *Client) GetOrganization(ctx context.Context, ein string) (*registry.Organization, error) {
	pda, _, err := c.DeriveOrganizationPDA(ein)
	if err != nil {
		return nil, fmt.Errorf("derive PDA: %w", err)
	}

	accountInfo, err := c.rpcClient.GetAccountInfo(ctx, pda)
	if err != nil {
		return nil, fmt.Errorf("get account info: %w", err)
	}
	if accountInfo == nil || accountInfo.Value == nil {
		return nil, fmt.Errorf("organization not found for EIN %s", ein)
	}

	data := accountInfo.Value.Data.GetBinary()
	org, err := deserializeOrganization(data)
	if err != nil {
		return nil, fmt.Errorf("deserialize organization: %w", err)
	}

	return &registry.Organization{
		EIN:            org.EIN,
		Name:           org.Name,
		Domain:         org.Domain,
		DIDURI:         org.DIDURI,
		IssuerID:       chain.Address(org.Issuer.String()),
		VCHash:         org.VCHash,
		PaymentAddress: chain.Address(org.USDCAddress.String()),
		Active:         org.Active,
		CreatedAt:      org.CreatedAt,
		UpdatedAt:      org.UpdatedAt,
	}, nil
}

// GetIssuer fetches an issuer account by authority address and returns
// the chain-agnostic Issuer type.
func (c *Client) GetIssuer(ctx context.Context, authority chain.Address) (*registry.Issuer, error) {
	pubkey, err := solanago.PublicKeyFromBase58(string(authority))
	if err != nil {
		return nil, fmt.Errorf("parse authority address: %w", err)
	}

	pda, _, err := c.DeriveIssuerPDA(pubkey)
	if err != nil {
		return nil, fmt.Errorf("derive PDA: %w", err)
	}

	accountInfo, err := c.rpcClient.GetAccountInfo(ctx, pda)
	if err != nil {
		return nil, fmt.Errorf("get account info: %w", err)
	}
	if accountInfo == nil || accountInfo.Value == nil {
		return nil, fmt.Errorf("issuer not found")
	}

	data := accountInfo.Value.Data.GetBinary()
	issuer, err := deserializeIssuer(data)
	if err != nil {
		return nil, fmt.Errorf("deserialize issuer: %w", err)
	}

	return &registry.Issuer{
		Authority: chain.Address(issuer.Authority.String()),
		DID:       issuer.DID,
		Active:    issuer.Active,
		CreatedAt: issuer.CreatedAt,
	}, nil
}

// RegisterIssuer registers a new issuer on the Solana registry.
func (c *Client) RegisterIssuer(ctx context.Context, params registry.RegisterIssuerParams) (chain.TxHash, error) {
	authority := solanago.PrivateKey(params.AuthoritySigner)
	issuerAuth, err := solanago.PublicKeyFromBase58(string(params.IssuerAuthority))
	if err != nil {
		return "", fmt.Errorf("parse issuer authority: %w", err)
	}

	ix, err := c.BuildRegisterIssuerInstruction(authority.PublicKey(), issuerAuth, params.DID)
	if err != nil {
		return "", fmt.Errorf("build instruction: %w", err)
	}

	sig, err := c.sendTransaction(ctx, []solanago.Instruction{ix}, authority)
	if err != nil {
		return "", err
	}
	return chain.TxHash(sig.String()), nil
}

// RegisterOrganization registers a new organization on the Solana registry.
func (c *Client) RegisterOrganization(ctx context.Context, params registry.RegisterOrgParams) (chain.TxHash, error) {
	issuerAuth := solanago.PrivateKey(params.IssuerSigner)
	paymentAddr, err := solanago.PublicKeyFromBase58(string(params.PaymentAddress))
	if err != nil {
		return "", fmt.Errorf("parse payment address: %w", err)
	}

	ix, err := c.BuildRegisterOrganizationInstruction(
		issuerAuth.PublicKey(), params.EIN, params.Name, params.Domain,
		params.DIDURI, params.VCHash, paymentAddr,
	)
	if err != nil {
		return "", fmt.Errorf("build instruction: %w", err)
	}

	sig, err := c.sendTransaction(ctx, []solanago.Instruction{ix}, issuerAuth)
	if err != nil {
		return "", err
	}
	return chain.TxHash(sig.String()), nil
}

// DeactivateOrganization deactivates an organization on the Solana registry.
func (c *Client) DeactivateOrganization(ctx context.Context, params registry.DeactivateOrgParams) (chain.TxHash, error) {
	issuerAuth := solanago.PrivateKey(params.IssuerSigner)

	ix, err := c.BuildDeactivateOrganizationInstruction(issuerAuth.PublicKey(), params.EIN)
	if err != nil {
		return "", fmt.Errorf("build instruction: %w", err)
	}

	sig, err := c.sendTransaction(ctx, []solanago.Instruction{ix}, issuerAuth)
	if err != nil {
		return "", err
	}
	return chain.TxHash(sig.String()), nil
}

// DeriveIssuerPDA derives the PDA for an issuer account.
func (c *Client) DeriveIssuerPDA(issuerAuthority solanago.PublicKey) (solanago.PublicKey, uint8, error) {
	return solanago.FindProgramAddress(
		[][]byte{
			[]byte("issuer"),
			issuerAuthority.Bytes(),
		},
		c.programID,
	)
}

// DeriveOrganizationPDA derives the PDA for an organization account.
func (c *Client) DeriveOrganizationPDA(ein string) (solanago.PublicKey, uint8, error) {
	return solanago.FindProgramAddress(
		[][]byte{
			[]byte("organization"),
			[]byte(ein),
		},
		c.programID,
	)
}

// GetProgramID returns the Solana program ID.
func (c *Client) GetProgramID() solanago.PublicKey {
	return c.programID
}

// GetRPCClient returns the underlying Solana RPC client for devnet operations.
func (c *Client) GetRPCClient() *rpc.Client {
	return c.rpcClient
}

// deserializeIssuer deserializes an IssuerAccount from Anchor account data.
func deserializeIssuer(data []byte) (*IssuerAccount, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for discriminator")
	}

	offset := 8 // skip Anchor discriminator
	issuer := &IssuerAccount{}

	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for authority")
	}
	copy(issuer.Authority[:], data[offset:offset+32])
	offset += 32

	if offset+4 > len(data) {
		return nil, fmt.Errorf("data too short for did length")
	}
	didLen := int(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4
	if offset+didLen > len(data) {
		return nil, fmt.Errorf("data too short for did")
	}
	issuer.DID = string(data[offset : offset+didLen])
	offset += didLen

	if offset+1 > len(data) {
		return nil, fmt.Errorf("data too short for active")
	}
	issuer.Active = data[offset] != 0
	offset++

	if offset+8 > len(data) {
		return nil, fmt.Errorf("data too short for created_at")
	}
	issuer.CreatedAt = int64(binary.LittleEndian.Uint64(data[offset:]))

	return issuer, nil
}

// deserializeOrganization deserializes an OrganizationAccount from Anchor account data.
func deserializeOrganization(data []byte) (*OrganizationAccount, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for discriminator")
	}

	offset := 8 // skip Anchor discriminator
	org := &OrganizationAccount{}

	readString := func() (string, error) {
		if offset+4 > len(data) {
			return "", fmt.Errorf("data too short for string length at offset %d", offset)
		}
		strLen := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4
		if offset+strLen > len(data) {
			return "", fmt.Errorf("data too short for string data at offset %d", offset)
		}
		s := string(data[offset : offset+strLen])
		offset += strLen
		return s, nil
	}

	var err error

	org.EIN, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read ein: %w", err)
	}

	org.Name, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read name: %w", err)
	}

	org.Domain, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read domain: %w", err)
	}

	org.DIDURI, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read did_uri: %w", err)
	}

	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for issuer pubkey")
	}
	copy(org.Issuer[:], data[offset:offset+32])
	offset += 32

	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for vc_hash")
	}
	copy(org.VCHash[:], data[offset:offset+32])
	offset += 32

	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for usdc_address")
	}
	copy(org.USDCAddress[:], data[offset:offset+32])
	offset += 32

	if offset+1 > len(data) {
		return nil, fmt.Errorf("data too short for active")
	}
	org.Active = data[offset] != 0
	offset++

	if offset+8 > len(data) {
		return nil, fmt.Errorf("data too short for created_at")
	}
	org.CreatedAt = int64(binary.LittleEndian.Uint64(data[offset:]))
	offset += 8

	if offset+8 > len(data) {
		return nil, fmt.Errorf("data too short for updated_at")
	}
	org.UpdatedAt = int64(binary.LittleEndian.Uint64(data[offset:]))

	return org, nil
}
