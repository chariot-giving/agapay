package registry

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Client interacts with the Agapay Registry Solana program.
type Client struct {
	rpcClient *rpc.Client
	programID solana.PublicKey
}

// NewClient creates a new registry client.
func NewClient(rpcURL string, programID solana.PublicKey) *Client {
	return &Client{
		rpcClient: rpc.New(rpcURL),
		programID: programID,
	}
}

// DeriveIssuerPDA derives the PDA for an issuer account.
func (c *Client) DeriveIssuerPDA(issuerAuthority solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{
			[]byte("issuer"),
			issuerAuthority.Bytes(),
		},
		c.programID,
	)
}

// DeriveOrganizationPDA derives the PDA for an organization account.
func (c *Client) DeriveOrganizationPDA(ein string) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{
			[]byte("organization"),
			[]byte(ein),
		},
		c.programID,
	)
}

// GetOrganization fetches an organization account by EIN.
func (c *Client) GetOrganization(ctx context.Context, ein string) (*OrganizationAccount, error) {
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

	return org, nil
}

// GetIssuer fetches an issuer account by authority public key.
func (c *Client) GetIssuer(ctx context.Context, authority solana.PublicKey) (*IssuerAccount, error) {
	pda, _, err := c.DeriveIssuerPDA(authority)
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

	return issuer, nil
}

// ComputeVCHash computes a SHA-256 hash of one or more VC JWT tokens.
func ComputeVCHash(vcTokens ...string) [32]byte {
	h := sha256.New()
	for _, token := range vcTokens {
		h.Write([]byte(token))
	}
	var hash [32]byte
	copy(hash[:], h.Sum(nil))
	return hash
}

// GetProgramID returns the program ID.
func (c *Client) GetProgramID() solana.PublicKey {
	return c.programID
}

// GetRPCClient returns the underlying RPC client.
func (c *Client) GetRPCClient() *rpc.Client {
	return c.rpcClient
}

// deserializeIssuer deserializes an IssuerAccount from Anchor account data.
// Anchor format: 8-byte discriminator + fields.
func deserializeIssuer(data []byte) (*IssuerAccount, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for discriminator")
	}

	offset := 8 // skip discriminator
	issuer := &IssuerAccount{}

	// authority: Pubkey (32 bytes)
	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for authority")
	}
	copy(issuer.Authority[:], data[offset:offset+32])
	offset += 32

	// did: String (4-byte length prefix + data)
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

	// active: bool (1 byte)
	if offset+1 > len(data) {
		return nil, fmt.Errorf("data too short for active")
	}
	issuer.Active = data[offset] != 0
	offset++

	// created_at: i64 (8 bytes)
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

	offset := 8 // skip discriminator
	org := &OrganizationAccount{}

	// Helper to read a Borsh string (4-byte LE length + data)
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

	// ein
	org.EIN, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read ein: %w", err)
	}

	// name
	org.Name, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read name: %w", err)
	}

	// domain
	org.Domain, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read domain: %w", err)
	}

	// did_uri
	org.DIDURI, err = readString()
	if err != nil {
		return nil, fmt.Errorf("read did_uri: %w", err)
	}

	// issuer: Pubkey (32 bytes)
	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for issuer pubkey")
	}
	copy(org.Issuer[:], data[offset:offset+32])
	offset += 32

	// vc_hash: [u8; 32]
	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for vc_hash")
	}
	copy(org.VCHash[:], data[offset:offset+32])
	offset += 32

	// usdc_address: Pubkey (32 bytes)
	if offset+32 > len(data) {
		return nil, fmt.Errorf("data too short for usdc_address")
	}
	copy(org.USDCAddress[:], data[offset:offset+32])
	offset += 32

	// active: bool (1 byte)
	if offset+1 > len(data) {
		return nil, fmt.Errorf("data too short for active")
	}
	org.Active = data[offset] != 0
	offset++

	// created_at: i64 (8 bytes)
	if offset+8 > len(data) {
		return nil, fmt.Errorf("data too short for created_at")
	}
	org.CreatedAt = int64(binary.LittleEndian.Uint64(data[offset:]))
	offset += 8

	// updated_at: i64 (8 bytes)
	if offset+8 > len(data) {
		return nil, fmt.Errorf("data too short for updated_at")
	}
	org.UpdatedAt = int64(binary.LittleEndian.Uint64(data[offset:]))

	return org, nil
}
