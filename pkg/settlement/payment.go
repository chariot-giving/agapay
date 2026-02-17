package settlement

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// PaymentClient sends USDC payments on Solana with SPL Memo.
type PaymentClient struct {
	rpcClient *rpc.Client
	wsURL     string
}

// NewPaymentClient creates a new payment client.
func NewPaymentClient(rpcURL string, wsURL string) *PaymentClient {
	return &PaymentClient{
		rpcClient: rpc.New(rpcURL),
		wsURL:     wsURL,
	}
}

// SendPayment sends a USDC transfer with an SPL Memo containing the public header.
func (pc *PaymentClient) SendPayment(ctx context.Context, params *PaymentParams) (*PaymentResult, error) {
	payer := params.PayerWallet.PublicKey()

	// Build the SPL Memo instruction manually to pass raw UTF-8 data.
	// The library's MarshalWithEncoder Borsh-encodes the []byte with a
	// 4-byte length prefix, which produces invalid UTF-8 for the Memo program.
	memoInstruction := solana.NewInstruction(
		solana.MemoProgramID,
		solana.AccountMetaSlice{
			solana.Meta(payer).SIGNER(),
		},
		params.MemoData,
	)

	// Build the SPL Token Transfer instruction
	transferInst := token.NewTransferInstruction(
		params.Amount,
		params.PayerTokenAccount,
		params.RecipientTokenAccount,
		payer,
		[]solana.PublicKey{},
	)
	transferInstruction := transferInst.Build()

	// Get recent blockhash
	recent, err := pc.rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("get recent blockhash: %w", err)
	}

	// Build the transaction with both instructions
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			memoInstruction,
			transferInstruction,
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(payer),
	)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	// Sign the transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer) {
			return &params.PayerWallet
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	// Send and confirm
	wsClient, err := ws.Connect(ctx, pc.wsURL)
	if err != nil {
		return nil, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	sig, err := confirm.SendAndConfirmTransaction(ctx, pc.rpcClient, wsClient, tx)
	if err != nil {
		return nil, fmt.Errorf("send and confirm: %w", err)
	}

	return &PaymentResult{
		Signature: sig.String(),
		Amount:    params.Amount,
	}, nil
}

// GetRPCClient returns the underlying RPC client.
func (pc *PaymentClient) GetRPCClient() *rpc.Client {
	return pc.rpcClient
}
