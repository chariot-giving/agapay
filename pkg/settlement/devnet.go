package settlement

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// AirdropSOL requests an airdrop of SOL from the devnet faucet.
func AirdropSOL(ctx context.Context, rpcClient *rpc.Client, pubkey solana.PublicKey, lamports uint64) error {
	sig, err := rpcClient.RequestAirdrop(ctx, pubkey, lamports, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("request airdrop: %w", err)
	}

	// Poll for confirmation
	for i := 0; i < 30; i++ {
		time.Sleep(time.Second)
		status, err := rpcClient.GetSignatureStatuses(ctx, false, sig)
		if err != nil {
			continue
		}
		if status != nil && status.Value != nil && len(status.Value) > 0 && status.Value[0] != nil {
			if status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized ||
				status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusConfirmed {
				return nil
			}
		}
	}

	return fmt.Errorf("airdrop confirmation timeout for sig %s", sig)
}

// CreateTestMint creates a new SPL token mint on devnet (acts as "test USDC").
// Returns the mint public key and the mint authority keypair.
func CreateTestMint(
	ctx context.Context,
	rpcClient *rpc.Client,
	payer solana.PrivateKey,
	decimals uint8,
	wsURL string,
) (solana.PublicKey, solana.PrivateKey, error) {
	mintKeypair, err := solana.NewRandomPrivateKey()
	if err != nil {
		return solana.PublicKey{}, solana.PrivateKey{}, fmt.Errorf("generate mint keypair: %w", err)
	}

	mintPubkey := mintKeypair.PublicKey()
	payerPubkey := payer.PublicKey()

	// Calculate rent-exempt minimum for a Mint account (82 bytes)
	rentExempt, err := rpcClient.GetMinimumBalanceForRentExemption(ctx, token.MINT_SIZE, rpc.CommitmentFinalized)
	if err != nil {
		return solana.PublicKey{}, solana.PrivateKey{}, fmt.Errorf("get rent exemption: %w", err)
	}

	// Build instructions: create account + initialize mint
	createAccountIx := system.NewCreateAccountInstruction(
		rentExempt,
		token.MINT_SIZE,
		solana.TokenProgramID,
		payerPubkey,
		mintPubkey,
	).Build()

	initMintIx := token.NewInitializeMint2Instruction(
		decimals,
		payerPubkey,  // mint authority
		payerPubkey,  // freeze authority
		mintPubkey,
	).Build()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.PublicKey{}, solana.PrivateKey{}, fmt.Errorf("get blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{createAccountIx, initMintIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(payerPubkey),
	)
	if err != nil {
		return solana.PublicKey{}, solana.PrivateKey{}, fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payerPubkey) {
			return &payer
		}
		if key.Equals(mintPubkey) {
			pk := solana.PrivateKey(mintKeypair)
			return &pk
		}
		return nil
	})
	if err != nil {
		return solana.PublicKey{}, solana.PrivateKey{}, fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return solana.PublicKey{}, solana.PrivateKey{}, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	_, err = confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
	if err != nil {
		return solana.PublicKey{}, solana.PrivateKey{}, fmt.Errorf("send create mint: %w", err)
	}

	return mintPubkey, solana.PrivateKey(mintKeypair), nil
}

// CreateATA creates an Associated Token Account for the given owner and mint.
// Returns the ATA address.
func CreateATA(
	ctx context.Context,
	rpcClient *rpc.Client,
	payer solana.PrivateKey,
	owner solana.PublicKey,
	mint solana.PublicKey,
	wsURL string,
) (solana.PublicKey, error) {
	payerPubkey := payer.PublicKey()

	ata, _, err := solana.FindAssociatedTokenAddress(owner, mint)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("find ATA address: %w", err)
	}

	// Check if ATA already exists
	acctInfo, err := rpcClient.GetAccountInfo(ctx, ata)
	if err == nil && acctInfo != nil && acctInfo.Value != nil {
		return ata, nil // already exists
	}

	createATAIx := associatedtokenaccount.NewCreateInstruction(
		payerPubkey,
		owner,
		mint,
	).Build()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("get blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{createATAIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(payerPubkey),
	)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payerPubkey) {
			return &payer
		}
		return nil
	})
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	_, err = confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("send create ATA: %w", err)
	}

	return ata, nil
}

// MintTestTokens mints tokens to a destination token account.
func MintTestTokens(
	ctx context.Context,
	rpcClient *rpc.Client,
	mintAuthority solana.PrivateKey,
	mint solana.PublicKey,
	destination solana.PublicKey,
	amount uint64,
	wsURL string,
) error {
	authorityPubkey := mintAuthority.PublicKey()

	mintToIx := token.NewMintToInstruction(
		amount,
		mint,
		destination,
		authorityPubkey,
		[]solana.PublicKey{},
	).Build()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("get blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{mintToIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(authorityPubkey),
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(authorityPubkey) {
			return &mintAuthority
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	_, err = confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
	if err != nil {
		return fmt.Errorf("send mint tokens: %w", err)
	}

	return nil
}

// GetSOLBalance returns the SOL balance of an account in lamports.
func GetSOLBalance(ctx context.Context, rpcClient *rpc.Client, pubkey solana.PublicKey) (uint64, error) {
	balance, err := rpcClient.GetBalance(ctx, pubkey, rpc.CommitmentFinalized)
	if err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return balance.Value, nil
}

// TransferSOL sends SOL from one account to another.
func TransferSOL(
	ctx context.Context,
	rpcClient *rpc.Client,
	from solana.PrivateKey,
	to solana.PublicKey,
	lamports uint64,
	wsURL string,
) error {
	fromPubkey := from.PublicKey()

	transferIx := system.NewTransferInstruction(
		lamports,
		fromPubkey,
		to,
	).Build()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("get blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{transferIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(fromPubkey),
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(fromPubkey) {
			return &from
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	_, err = confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
	if err != nil {
		return fmt.Errorf("send SOL transfer: %w", err)
	}

	return nil
}
