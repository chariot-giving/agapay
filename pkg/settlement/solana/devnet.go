package solana

import (
	"context"
	"fmt"
	"time"

	solanago "github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// AirdropSOL requests an airdrop of SOL from the devnet faucet.
func AirdropSOL(ctx context.Context, rpcClient *rpc.Client, pubkey solanago.PublicKey, lamports uint64) error {
	sig, err := rpcClient.RequestAirdrop(ctx, pubkey, lamports, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("request airdrop: %w", err)
	}

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
func CreateTestMint(
	ctx context.Context,
	rpcClient *rpc.Client,
	payer solanago.PrivateKey,
	decimals uint8,
	wsURL string,
) (solanago.PublicKey, solanago.PrivateKey, error) {
	mintKeypair, err := solanago.NewRandomPrivateKey()
	if err != nil {
		return solanago.PublicKey{}, solanago.PrivateKey{}, fmt.Errorf("generate mint keypair: %w", err)
	}

	mintPubkey := mintKeypair.PublicKey()
	payerPubkey := payer.PublicKey()

	rentExempt, err := rpcClient.GetMinimumBalanceForRentExemption(ctx, token.MINT_SIZE, rpc.CommitmentFinalized)
	if err != nil {
		return solanago.PublicKey{}, solanago.PrivateKey{}, fmt.Errorf("get rent exemption: %w", err)
	}

	createAccountIx := system.NewCreateAccountInstruction(
		rentExempt,
		token.MINT_SIZE,
		solanago.TokenProgramID,
		payerPubkey,
		mintPubkey,
	).Build()

	initMintIx := token.NewInitializeMint2Instruction(
		decimals,
		payerPubkey,
		payerPubkey,
		mintPubkey,
	).Build()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solanago.PublicKey{}, solanago.PrivateKey{}, fmt.Errorf("get blockhash: %w", err)
	}

	tx, err := solanago.NewTransaction(
		[]solanago.Instruction{createAccountIx, initMintIx},
		recent.Value.Blockhash,
		solanago.TransactionPayer(payerPubkey),
	)
	if err != nil {
		return solanago.PublicKey{}, solanago.PrivateKey{}, fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solanago.PublicKey) *solanago.PrivateKey {
		if key.Equals(payerPubkey) {
			return &payer
		}
		if key.Equals(mintPubkey) {
			pk := solanago.PrivateKey(mintKeypair)
			return &pk
		}
		return nil
	})
	if err != nil {
		return solanago.PublicKey{}, solanago.PrivateKey{}, fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return solanago.PublicKey{}, solanago.PrivateKey{}, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	_, err = confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
	if err != nil {
		return solanago.PublicKey{}, solanago.PrivateKey{}, fmt.Errorf("send create mint: %w", err)
	}

	return mintPubkey, solanago.PrivateKey(mintKeypair), nil
}

// CreateATA creates an Associated Token Account for the given owner and mint.
func CreateATA(
	ctx context.Context,
	rpcClient *rpc.Client,
	payer solanago.PrivateKey,
	owner solanago.PublicKey,
	mint solanago.PublicKey,
	wsURL string,
) (solanago.PublicKey, error) {
	payerPubkey := payer.PublicKey()

	ata, _, err := solanago.FindAssociatedTokenAddress(owner, mint)
	if err != nil {
		return solanago.PublicKey{}, fmt.Errorf("find ATA address: %w", err)
	}

	acctInfo, err := rpcClient.GetAccountInfo(ctx, ata)
	if err == nil && acctInfo != nil && acctInfo.Value != nil {
		return ata, nil
	}

	createATAIx := associatedtokenaccount.NewCreateInstruction(
		payerPubkey,
		owner,
		mint,
	).Build()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solanago.PublicKey{}, fmt.Errorf("get blockhash: %w", err)
	}

	tx, err := solanago.NewTransaction(
		[]solanago.Instruction{createATAIx},
		recent.Value.Blockhash,
		solanago.TransactionPayer(payerPubkey),
	)
	if err != nil {
		return solanago.PublicKey{}, fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solanago.PublicKey) *solanago.PrivateKey {
		if key.Equals(payerPubkey) {
			return &payer
		}
		return nil
	})
	if err != nil {
		return solanago.PublicKey{}, fmt.Errorf("sign transaction: %w", err)
	}

	wsClient, err := ws.Connect(ctx, wsURL)
	if err != nil {
		return solanago.PublicKey{}, fmt.Errorf("connect websocket: %w", err)
	}
	defer wsClient.Close()

	_, err = confirm.SendAndConfirmTransaction(ctx, rpcClient, wsClient, tx)
	if err != nil {
		return solanago.PublicKey{}, fmt.Errorf("send create ATA: %w", err)
	}

	return ata, nil
}

// MintTestTokens mints tokens to a destination token account.
func MintTestTokens(
	ctx context.Context,
	rpcClient *rpc.Client,
	mintAuthority solanago.PrivateKey,
	mint solanago.PublicKey,
	destination solanago.PublicKey,
	amount uint64,
	wsURL string,
) error {
	authorityPubkey := mintAuthority.PublicKey()

	mintToIx := token.NewMintToInstruction(
		amount,
		mint,
		destination,
		authorityPubkey,
		[]solanago.PublicKey{},
	).Build()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("get blockhash: %w", err)
	}

	tx, err := solanago.NewTransaction(
		[]solanago.Instruction{mintToIx},
		recent.Value.Blockhash,
		solanago.TransactionPayer(authorityPubkey),
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solanago.PublicKey) *solanago.PrivateKey {
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
func GetSOLBalance(ctx context.Context, rpcClient *rpc.Client, pubkey solanago.PublicKey) (uint64, error) {
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
	from solanago.PrivateKey,
	to solanago.PublicKey,
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

	tx, err := solanago.NewTransaction(
		[]solanago.Instruction{transferIx},
		recent.Value.Blockhash,
		solanago.TransactionPayer(fromPubkey),
	)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	_, err = tx.Sign(func(key solanago.PublicKey) *solanago.PrivateKey {
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
