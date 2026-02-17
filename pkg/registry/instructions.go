package registry

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// Anchor instruction discriminators are sha256("global:<instruction_name>")[..8].
var (
	discRegisterIssuer       = computeDiscriminator("global:register_issuer")
	discRegisterOrganization = computeDiscriminator("global:register_organization")
	discUpdateOrganization   = computeDiscriminator("global:update_organization")
	discDeactivateOrg        = computeDiscriminator("global:deactivate_organization")
)

func computeDiscriminator(name string) [8]byte {
	hash := sha256.Sum256([]byte(name))
	var disc [8]byte
	copy(disc[:], hash[:8])
	return disc
}

// borshEncodeString encodes a string in Borsh format (4-byte LE length + UTF-8 bytes).
func borshEncodeString(s string) []byte {
	buf := make([]byte, 4+len(s))
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(s)))
	copy(buf[4:], s)
	return buf
}

// BuildRegisterIssuerInstruction builds the register_issuer instruction data and accounts.
func (c *Client) BuildRegisterIssuerInstruction(
	authority solana.PublicKey,
	issuerAuthority solana.PublicKey,
	did string,
) (solana.Instruction, error) {
	issuerPDA, _, err := c.DeriveIssuerPDA(issuerAuthority)
	if err != nil {
		return nil, fmt.Errorf("derive issuer PDA: %w", err)
	}

	// Instruction data: discriminator + borsh(did)
	data := make([]byte, 0, 8+4+len(did))
	data = append(data, discRegisterIssuer[:]...)
	data = append(data, borshEncodeString(did)...)

	return solana.NewInstruction(
		c.programID,
		solana.AccountMetaSlice{
			solana.NewAccountMeta(authority, true, true),          // authority (signer, mut)
			solana.NewAccountMeta(issuerAuthority, false, false),  // issuer_authority
			solana.NewAccountMeta(issuerPDA, true, false),         // issuer_account (init -> mut)
			solana.NewAccountMeta(solana.SystemProgramID, false, false), // system_program
		},
		data,
	), nil
}

// BuildRegisterOrganizationInstruction builds the register_organization instruction.
func (c *Client) BuildRegisterOrganizationInstruction(
	issuerAuthority solana.PublicKey,
	ein string,
	name string,
	domain string,
	didURI string,
	vcHash [32]byte,
	usdcAddress solana.PublicKey,
) (solana.Instruction, error) {
	issuerPDA, _, err := c.DeriveIssuerPDA(issuerAuthority)
	if err != nil {
		return nil, fmt.Errorf("derive issuer PDA: %w", err)
	}

	orgPDA, _, err := c.DeriveOrganizationPDA(ein)
	if err != nil {
		return nil, fmt.Errorf("derive org PDA: %w", err)
	}

	// Instruction data: discriminator + borsh(ein, name, domain, did_uri, vc_hash, usdc_address)
	data := make([]byte, 0, 512)
	data = append(data, discRegisterOrganization[:]...)
	data = append(data, borshEncodeString(ein)...)
	data = append(data, borshEncodeString(name)...)
	data = append(data, borshEncodeString(domain)...)
	data = append(data, borshEncodeString(didURI)...)
	data = append(data, vcHash[:]...)
	data = append(data, usdcAddress.Bytes()...)

	return solana.NewInstruction(
		c.programID,
		solana.AccountMetaSlice{
			solana.NewAccountMeta(issuerAuthority, true, true),          // issuer_authority (signer, mut)
			solana.NewAccountMeta(issuerPDA, false, false),              // issuer_account (PDA)
			solana.NewAccountMeta(orgPDA, true, false),                  // organization_account (init -> mut)
			solana.NewAccountMeta(solana.SystemProgramID, false, false), // system_program
		},
		data,
	), nil
}

// BuildDeactivateOrganizationInstruction builds the deactivate_organization instruction.
func (c *Client) BuildDeactivateOrganizationInstruction(
	issuerAuthority solana.PublicKey,
	ein string,
) (solana.Instruction, error) {
	issuerPDA, _, err := c.DeriveIssuerPDA(issuerAuthority)
	if err != nil {
		return nil, fmt.Errorf("derive issuer PDA: %w", err)
	}

	orgPDA, _, err := c.DeriveOrganizationPDA(ein)
	if err != nil {
		return nil, fmt.Errorf("derive org PDA: %w", err)
	}

	data := make([]byte, 8)
	copy(data, discDeactivateOrg[:])

	return solana.NewInstruction(
		c.programID,
		solana.AccountMetaSlice{
			solana.NewAccountMeta(issuerAuthority, false, true),  // issuer_authority (signer)
			solana.NewAccountMeta(issuerPDA, false, false),       // issuer_account
			solana.NewAccountMeta(orgPDA, true, false),           // organization_account (mut)
		},
		data,
	), nil
}

// SendRegisterIssuer builds, signs, sends, and confirms a register_issuer transaction.
func (c *Client) SendRegisterIssuer(
	ctx context.Context,
	authority solana.PrivateKey,
	issuerAuthority solana.PublicKey,
	did string,
	wsURL string,
) (solana.Signature, error) {
	ix, err := c.BuildRegisterIssuerInstruction(authority.PublicKey(), issuerAuthority, did)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("build instruction: %w", err)
	}

	return c.sendTransaction(ctx, []solana.Instruction{ix}, authority, wsURL)
}

// SendRegisterOrganization builds, signs, sends, and confirms a register_organization transaction.
func (c *Client) SendRegisterOrganization(
	ctx context.Context,
	issuerAuthority solana.PrivateKey,
	ein string,
	name string,
	domain string,
	didURI string,
	vcHash [32]byte,
	usdcAddress solana.PublicKey,
	wsURL string,
) (solana.Signature, error) {
	ix, err := c.BuildRegisterOrganizationInstruction(
		issuerAuthority.PublicKey(), ein, name, domain, didURI, vcHash, usdcAddress,
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("build instruction: %w", err)
	}

	return c.sendTransaction(ctx, []solana.Instruction{ix}, issuerAuthority, wsURL)
}

// sendTransaction is a helper that builds, signs, sends, and confirms a transaction.
func (c *Client) sendTransaction(
	ctx context.Context,
	instructions []solana.Instruction,
	payer solana.PrivateKey,
	wsURL string,
) (solana.Signature, error) {
	recent, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("get latest blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer.PublicKey()) {
			return &payer
		}
		return nil
	})
	if err != nil {
		return solana.Signature{}, fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	sig, err := confirm.SendAndConfirmTransaction(ctx, c.rpcClient, wsClient, tx)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("send and confirm: %w", err)
	}

	return sig, nil
}
