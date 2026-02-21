// Package tempo implements the Agapay Settler interface for Tempo (EVM).
// Payments are sent using TIP-20 transferWithMemo via the AgapayPaymentRouter
// contract, using Tempo Transactions (Type 0x76) from the tempo-go SDK.
package tempo

import (
	"context"
	"fmt"
	"math/big"

	"github.com/chariot-giving/agapay/pkg/chain"
	"github.com/chariot-giving/agapay/pkg/settlement"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tempoxyz/tempo-go/pkg/client"
	"github.com/tempoxyz/tempo-go/pkg/signer"
	"github.com/tempoxyz/tempo-go/pkg/transaction"
)

// Compile-time interface check.
var _ settlement.Settler = (*PaymentClient)(nil)

// PaymentClient sends stablecoin payments on Tempo via the AgapayPaymentRouter.
type PaymentClient struct {
	rpc           *client.Client
	routerAddress common.Address
	tokenAddress  common.Address
	chainID       int64
	id            chain.ID
}

// NewPaymentClient creates a new Tempo payment client.
//
// Parameters:
//   - rpcURL: Tempo JSON-RPC endpoint
//   - routerAddr: Deployed AgapayPaymentRouter contract address
//   - tokenAddr: TIP-20 token contract address (e.g., USDC on Tempo)
//   - chainID: Numeric EVM chain ID for Tempo network
//   - id: Agapay chain identifier (e.g., chain.TempoModerato)
func NewPaymentClient(
	rpcURL string,
	routerAddr common.Address,
	tokenAddr common.Address,
	chainID int64,
	id chain.ID,
	opts ...client.Option,
) *PaymentClient {
	return &PaymentClient{
		rpc:           client.New(rpcURL, opts...),
		routerAddress: routerAddr,
		tokenAddress:  tokenAddr,
		chainID:       chainID,
		id:            id,
	}
}

func (pc *PaymentClient) ChainID() chain.ID {
	return pc.id
}

// SendPayment executes a TIP-20 payment through the AgapayPaymentRouter.
//
// It builds a Tempo Transaction (Type 0x76) that batches two calls atomically:
//  1. TIP-20 approve(router, amount) - authorize the router to spend tokens
//  2. AgapayPaymentRouter.sendPayment(...) - transfers with memo and emits event
//
// The 32-byte memo is computed as keccak256(MemoData), linking the on-chain
// transfer to the full public header stored on IPFS.
func (pc *PaymentClient) SendPayment(ctx context.Context, params *settlement.PaymentParams) (*settlement.PaymentResult, error) {
	sgn, err := pc.signerFromKey(params.SignerKey)
	if err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}

	recipient := common.HexToAddress(string(params.Recipient))
	amount := new(big.Int).SetUint64(params.Amount)

	// 32-byte memo: keccak256 of the public header (links to IPFS CID content)
	memo := crypto.Keccak256Hash(params.MemoData)

	// Call 1: TIP-20 approve(router, amount)
	approveData := encodeApprove(pc.routerAddress, amount)

	// Call 2: AgapayPaymentRouter.sendPayment(token, to, amount, memo, publicHeader)
	routerData := encodeSendPayment(pc.tokenAddress, recipient, amount, memo, params.MemoData)

	nonce, err := pc.rpc.GetTransactionCount(ctx, sgn.Address().Hex())
	if err != nil {
		return nil, fmt.Errorf("get nonce: %w", err)
	}

	tokenAddr := pc.tokenAddress
	routerAddr := pc.routerAddress

	tx := transaction.NewDefault(pc.chainID)
	tx.Nonce = nonce
	tx.Gas = 2_000_000
	tx.MaxFeePerGas = big.NewInt(25_000_000_000)
	tx.MaxPriorityFeePerGas = big.NewInt(2_000_000_000)
	tx.Calls = []transaction.Call{
		{To: &tokenAddr, Value: big.NewInt(0), Data: approveData},
		{To: &routerAddr, Value: big.NewInt(0), Data: routerData},
	}

	if err := transaction.SignTransaction(tx, sgn); err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	serialized, err := transaction.Serialize(tx, nil)
	if err != nil {
		return nil, fmt.Errorf("serialize transaction: %w", err)
	}

	hash, err := pc.rpc.SendRawTransaction(ctx, serialized)
	if err != nil {
		return nil, fmt.Errorf("send transaction: %w", err)
	}

	return &settlement.PaymentResult{
		TxHash:  chain.TxHash(hash),
		Amount:  params.Amount,
		ChainID: pc.id,
	}, nil
}

// RPCClient returns the underlying Tempo RPC client.
func (pc *PaymentClient) RPCClient() *client.Client {
	return pc.rpc
}

func (pc *PaymentClient) signerFromKey(key []byte) (*signer.Signer, error) {
	hexKey := "0x" + common.Bytes2Hex(key)
	return signer.NewSigner(hexKey)
}

// ABI encoding helpers.

var (
	// approve(address,uint256)
	selApprove = crypto.Keccak256([]byte("approve(address,uint256)"))[:4]

	// sendPayment(address,address,uint256,bytes32,bytes)
	selSendPayment = crypto.Keccak256([]byte("sendPayment(address,address,uint256,bytes32,bytes)"))[:4]
)

func encodeApprove(spender common.Address, amount *big.Int) []byte {
	data := make([]byte, 4+64)
	copy(data[:4], selApprove)
	copy(data[4+12:4+32], spender.Bytes())
	amount.FillBytes(data[4+32 : 4+64])
	return data
}

func encodeSendPayment(
	token common.Address,
	to common.Address,
	amount *big.Int,
	memo common.Hash,
	publicHeader []byte,
) []byte {
	// sendPayment(address token, address to, uint256 amount, bytes32 memo, bytes publicHeader)
	// 5 params: 3 static (address, address, uint256) + 1 static (bytes32) + 1 dynamic (bytes)
	data := make([]byte, 4)
	copy(data, selSendPayment)

	// token (address)
	data = append(data, padLeft32(token.Bytes())...)

	// to (address)
	data = append(data, padLeft32(to.Bytes())...)

	// amount (uint256)
	amountBytes := make([]byte, 32)
	amount.FillBytes(amountBytes)
	data = append(data, amountBytes...)

	// memo (bytes32)
	data = append(data, memo.Bytes()...)

	// offset to bytes (5th param, offset = 5 * 32 = 160)
	data = append(data, padLeft32(big.NewInt(160).Bytes())...)

	// bytes: length + padded data
	data = append(data, padLeft32(big.NewInt(int64(len(publicHeader))).Bytes())...)
	padded := len(publicHeader)
	if padded%32 != 0 {
		padded = padded + (32 - padded%32)
	}
	buf := make([]byte, padded)
	copy(buf, publicHeader)
	data = append(data, buf...)

	return data
}

func padLeft32(b []byte) []byte {
	if len(b) >= 32 {
		return b[:32]
	}
	pad := make([]byte, 32)
	copy(pad[32-len(b):], b)
	return pad
}
