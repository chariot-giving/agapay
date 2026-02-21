package solana

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	solanago "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// Anchor instruction discriminators: sha256("global:<instruction_name>")[..8].
var (
	discRegisterIssuer       = computeDiscriminator("global:register_issuer")
	discRegisterOrganization = computeDiscriminator("global:register_organization")
	discDeactivateOrg        = computeDiscriminator("global:deactivate_organization")
)

func computeDiscriminator(name string) [8]byte {
	hash := sha256.Sum256([]byte(name))
	var disc [8]byte
	copy(disc[:], hash[:8])
	return disc
}

func borshEncodeString(s string) []byte {
	buf := make([]byte, 4+len(s))
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(s)))
	copy(buf[4:], s)
	return buf
}

// BuildRegisterIssuerInstruction builds the register_issuer instruction data and accounts.
func (c *Client) BuildRegisterIssuerInstruction(
	authority solanago.PublicKey,
	issuerAuthority solanago.PublicKey,
	did string,
) (solanago.Instruction, error) {
	issuerPDA, _, err := c.DeriveIssuerPDA(issuerAuthority)
	if err != nil {
		return nil, fmt.Errorf("derive issuer PDA: %w", err)
	}

	data := make([]byte, 0, 8+4+len(did))
	data = append(data, discRegisterIssuer[:]...)
	data = append(data, borshEncodeString(did)...)

	return solanago.NewInstruction(
		c.programID,
		solanago.AccountMetaSlice{
			solanago.NewAccountMeta(authority, true, true),
			solanago.NewAccountMeta(issuerAuthority, false, false),
			solanago.NewAccountMeta(issuerPDA, true, false),
			solanago.NewAccountMeta(solanago.SystemProgramID, false, false),
		},
		data,
	), nil
}

// BuildRegisterOrganizationInstruction builds the register_organization instruction.
func (c *Client) BuildRegisterOrganizationInstruction(
	issuerAuthority solanago.PublicKey,
	ein string,
	name string,
	domain string,
	didURI string,
	vcHash [32]byte,
	usdcAddress solanago.PublicKey,
) (solanago.Instruction, error) {
	issuerPDA, _, err := c.DeriveIssuerPDA(issuerAuthority)
	if err != nil {
		return nil, fmt.Errorf("derive issuer PDA: %w", err)
	}

	orgPDA, _, err := c.DeriveOrganizationPDA(ein)
	if err != nil {
		return nil, fmt.Errorf("derive org PDA: %w", err)
	}

	data := make([]byte, 0, 512)
	data = append(data, discRegisterOrganization[:]...)
	data = append(data, borshEncodeString(ein)...)
	data = append(data, borshEncodeString(name)...)
	data = append(data, borshEncodeString(domain)...)
	data = append(data, borshEncodeString(didURI)...)
	data = append(data, vcHash[:]...)
	data = append(data, usdcAddress.Bytes()...)

	return solanago.NewInstruction(
		c.programID,
		solanago.AccountMetaSlice{
			solanago.NewAccountMeta(issuerAuthority, true, true),
			solanago.NewAccountMeta(issuerPDA, false, false),
			solanago.NewAccountMeta(orgPDA, true, false),
			solanago.NewAccountMeta(solanago.SystemProgramID, false, false),
		},
		data,
	), nil
}

// BuildDeactivateOrganizationInstruction builds the deactivate_organization instruction.
func (c *Client) BuildDeactivateOrganizationInstruction(
	issuerAuthority solanago.PublicKey,
	ein string,
) (solanago.Instruction, error) {
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

	return solanago.NewInstruction(
		c.programID,
		solanago.AccountMetaSlice{
			solanago.NewAccountMeta(issuerAuthority, false, true),
			solanago.NewAccountMeta(issuerPDA, false, false),
			solanago.NewAccountMeta(orgPDA, true, false),
		},
		data,
	), nil
}

// sendTransaction builds, signs, sends, and confirms a transaction.
func (c *Client) sendTransaction(
	ctx context.Context,
	instructions []solanago.Instruction,
	payer solanago.PrivateKey,
) (solanago.Signature, error) {
	recent, err := c.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solanago.Signature{}, fmt.Errorf("get latest blockhash: %w", err)
	}

	tx, err := solanago.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solanago.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		return solanago.Signature{}, fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solanago.PublicKey) *solanago.PrivateKey {
		if key.Equals(payer.PublicKey()) {
			return &payer
		}
		return nil
	})
	if err != nil {
		return solanago.Signature{}, fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, c.wsURL)
	if err != nil {
		return solanago.Signature{}, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	sig, err := confirm.SendAndConfirmTransaction(ctx, c.rpcClient, wsClient, tx)
	if err != nil {
		return solanago.Signature{}, fmt.Errorf("send and confirm: %w", err)
	}

	return sig, nil
}
