package tempo

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tempoxyz/tempo-go/pkg/client"
)

// GetBalance returns the native balance of an address in wei.
func GetBalance(ctx context.Context, rpc *client.Client, address string) (*big.Int, error) {
	resp, err := rpc.SendRequest(ctx, "eth_getBalance", address, "latest")
	if err != nil {
		return nil, fmt.Errorf("eth_getBalance: %w", err)
	}
	if err := resp.CheckError(); err != nil {
		return nil, err
	}

	hexBalance, ok := resp.Result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	balance := new(big.Int)
	balance.SetString(hexBalance[2:], 16) // strip 0x
	return balance, nil
}

// GetTokenBalance returns the TIP-20 token balance of an address.
func GetTokenBalance(ctx context.Context, rpc *client.Client, tokenAddr common.Address, account string) (*big.Int, error) {
	// balanceOf(address) selector: 0x70a08231
	selector := common.FromHex("0x70a08231")
	data := make([]byte, 4+32)
	copy(data[:4], selector)
	addr := common.HexToAddress(account)
	copy(data[4+12:4+32], addr.Bytes())

	resp, err := rpc.SendRequest(ctx, "eth_call", map[string]string{
		"to":   tokenAddr.Hex(),
		"data": "0x" + common.Bytes2Hex(data),
	}, "latest")
	if err != nil {
		return nil, fmt.Errorf("eth_call balanceOf: %w", err)
	}
	if err := resp.CheckError(); err != nil {
		return nil, err
	}

	hexResult, ok := resp.Result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	balance := new(big.Int)
	balance.SetString(hexResult[2:], 16)
	return balance, nil
}
