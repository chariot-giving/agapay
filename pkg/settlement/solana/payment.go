// Package solana implements the Agapay Settler interface for Solana,
// using SPL Token transfers with SPL Memo for payment metadata.
package solana

import (
	"context"
	"fmt"

	"github.com/chariot-giving/agapay/pkg/chain"
	"github.com/chariot-giving/agapay/pkg/settlement"
	solanago "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// Compile-time interface check.
var _ settlement.Settler = (*PaymentClient)(nil)

// PaymentClient sends USDC payments on Solana with SPL Memo.
type PaymentClient struct {
	rpcClient *rpc.Client
	wsURL     string
	mint      solanago.PublicKey
	chainID   chain.ID
}

// NewPaymentClient creates a new Solana payment client.
// The mint is the SPL token mint address (e.g., USDC) used to derive payer ATAs.
func NewPaymentClient(rpcURL string, wsURL string, mint solanago.PublicKey, id chain.ID) *PaymentClient {
	return &PaymentClient{
		rpcClient: rpc.New(rpcURL),
		wsURL:     wsURL,
		mint:      mint,
		chainID:   id,
	}
}

// ChainID returns the chain identifier for this client.
func (pc *PaymentClient) ChainID() chain.ID {
	return pc.chainID
}

// SendPayment sends a USDC transfer with an SPL Memo containing the payment metadata.
// The payer ATA is derived from SignerKey and the configured mint.
// The Recipient address should be the recipient's token account (ATA).
func (pc *PaymentClient) SendPayment(ctx context.Context, params *settlement.PaymentParams) (*settlement.PaymentResult, error) {
	payerKey := solanago.PrivateKey(params.SignerKey)
	payer := payerKey.PublicKey()

	// Derive payer ATA from signer public key + mint
	payerATA, _, err := solanago.FindAssociatedTokenAddress(payer, pc.mint)
	if err != nil {
		return nil, fmt.Errorf("derive payer ATA: %w", err)
	}

	recipientATA, err := solanago.PublicKeyFromBase58(string(params.Recipient))
	if err != nil {
		return nil, fmt.Errorf("parse recipient address: %w", err)
	}

	// SPL Memo instruction with raw UTF-8 data
	memoInstruction := solanago.NewInstruction(
		solanago.MemoProgramID,
		solanago.AccountMetaSlice{
			solanago.Meta(payer).SIGNER(),
		},
		params.MemoData,
	)

	// SPL Token Transfer instruction
	transferInst := token.NewTransferInstruction(
		params.Amount,
		payerATA,
		recipientATA,
		payer,
		[]solanago.PublicKey{},
	)
	transferInstruction := transferInst.Build()

	recent, err := pc.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("get recent blockhash: %w", err)
	}

	tx, err := solanago.NewTransaction(
		[]solanago.Instruction{
			memoInstruction,
			transferInstruction,
		},
		recent.Value.Blockhash,
		solanago.TransactionPayer(payer),
	)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solanago.PublicKey) *solanago.PrivateKey {
		if key.Equals(payer) {
			return &payerKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, pc.wsURL)
	if err != nil {
		return nil, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	sig, err := confirm.SendAndConfirmTransaction(ctx, pc.rpcClient, wsClient, tx)
	if err != nil {
		return nil, fmt.Errorf("send and confirm: %w", err)
	}

	return &settlement.PaymentResult{
		TxHash:  chain.TxHash(sig.String()),
		Amount:  params.Amount,
		ChainID: pc.chainID,
	}, nil
}

// GetRPCClient returns the underlying Solana RPC client.
func (pc *PaymentClient) GetRPCClient() *rpc.Client {
	return pc.rpcClient
}
