package tempo

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/chariot-giving/agapay/pkg/chain"
	"github.com/chariot-giving/agapay/pkg/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Function selectors (first 4 bytes of keccak256 of the function signature).
var (
	// registerIssuer(address,string)
	selRegisterIssuer = crypto.Keccak256([]byte("registerIssuer(address,string)"))[:4]

	// registerOrganization(string,string,string,string,bytes32,address)
	selRegisterOrg = crypto.Keccak256([]byte("registerOrganization(string,string,string,string,bytes32,address)"))[:4]

	// deactivateOrganization(string)
	selDeactivateOrg = crypto.Keccak256([]byte("deactivateOrganization(string)"))[:4]

	// getOrganization(string)
	selGetOrg = crypto.Keccak256([]byte("getOrganization(string)"))[:4]

	// getIssuer(address)
	selGetIssuer = crypto.Keccak256([]byte("getIssuer(address)"))[:4]
)

// ABI encoding helpers. These manually encode ABI calldata to avoid pulling
// in the full go-ethereum ABI package, keeping the dependency surface small.

func encodeGetOrganization(ein string) []byte {
	// getOrganization(string)
	// selector + offset(32) + len(32) + data(padded)
	data := make([]byte, 4)
	copy(data, selGetOrg)

	// offset to string data
	data = append(data, padLeft(big.NewInt(32).Bytes(), 32)...)
	// string length
	data = append(data, padLeft(big.NewInt(int64(len(ein))).Bytes(), 32)...)
	// string data (padded to 32 bytes)
	data = append(data, padRight([]byte(ein), 32)...)

	return data
}

func encodeGetIssuer(addr common.Address) []byte {
	data := make([]byte, 4)
	copy(data, selGetIssuer)
	data = append(data, padLeft(addr.Bytes(), 32)...)
	return data
}

func encodeRegisterIssuer(authority common.Address, did string) []byte {
	data := make([]byte, 4)
	copy(data, selRegisterIssuer)

	// authority (address, padded to 32 bytes)
	data = append(data, padLeft(authority.Bytes(), 32)...)

	// offset to string "did"
	data = append(data, padLeft(big.NewInt(64).Bytes(), 32)...)

	// string: length + padded data
	data = append(data, padLeft(big.NewInt(int64(len(did))).Bytes(), 32)...)
	data = append(data, padRight([]byte(did), 32)...)

	return data
}

func encodeRegisterOrganization(
	ein, name, domain, didUri string,
	vcHash [32]byte,
	paymentAddr common.Address,
) []byte {
	data := make([]byte, 4)
	copy(data, selRegisterOrg)

	// 6 params: 4 strings (dynamic) + 1 bytes32 (static) + 1 address (static)
	// Head section: 6 x 32 bytes = 192 bytes for offsets/static values
	// Strings are dynamic, so their head slots hold offsets to the tail.
	// Layout: [offset_ein, offset_name, offset_domain, offset_didUri, vcHash, paymentAddr, ...tails]

	strings := []string{ein, name, domain, didUri}
	// Calculate offsets: head is 6 * 32 = 192 bytes from start of params
	tailStart := 6 * 32
	offsets := make([]int, 4)
	currentTail := tailStart
	for i, s := range strings {
		offsets[i] = currentTail
		currentTail += 32 + paddedLen(len(s)) // length word + padded string
	}

	// Write offsets for the 4 strings
	for _, off := range offsets {
		data = append(data, padLeft(big.NewInt(int64(off)).Bytes(), 32)...)
	}

	// vcHash (bytes32, static)
	data = append(data, vcHash[:]...)

	// paymentAddr (address, static)
	data = append(data, padLeft(paymentAddr.Bytes(), 32)...)

	// Tail: encode each string
	for _, s := range strings {
		data = append(data, padLeft(big.NewInt(int64(len(s))).Bytes(), 32)...)
		data = append(data, padRight([]byte(s), 32)...)
	}

	return data
}

func encodeDeactivateOrganization(ein string) []byte {
	data := make([]byte, 4)
	copy(data, selDeactivateOrg)

	// offset to string
	data = append(data, padLeft(big.NewInt(32).Bytes(), 32)...)
	data = append(data, padLeft(big.NewInt(int64(len(ein))).Bytes(), 32)...)
	data = append(data, padRight([]byte(ein), 32)...)

	return data
}

// Decoding helpers for return values.

func decodeOrganization(hexData string) (*registry.Organization, error) {
	data := common.FromHex(hexData)
	if len(data) < 320 { // minimum for the struct
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	org := &registry.Organization{}
	var err error

	// The Organization struct is returned as a tuple with dynamic types.
	// ABI encoding: first word is offset to the struct data (usually 32).
	structOffset := readUint(data[0:32])
	d := data[structOffset:]

	// Organization has 10 fields, first 4 are dynamic strings.
	// Head: [off_ein, off_name, off_domain, off_didUri, issuer, vcHash, paymentAddr, active, createdAt, updatedAt]
	if len(d) < 10*32 {
		return nil, fmt.Errorf("struct data too short")
	}

	offEIN := readUint(d[0:32])
	offName := readUint(d[32:64])
	offDomain := readUint(d[64:96])
	offDID := readUint(d[96:128])

	org.IssuerID = chain.Address(common.BytesToAddress(d[128:160]).Hex())

	var vcHash [32]byte
	copy(vcHash[:], d[160:192])
	org.VCHash = vcHash

	org.PaymentAddress = chain.Address(common.BytesToAddress(d[192:224]).Hex())
	org.Active = d[255] != 0
	org.CreatedAt = int64(binary.BigEndian.Uint64(d[280:288]))
	org.UpdatedAt = int64(binary.BigEndian.Uint64(d[312:320]))

	org.EIN, err = readABIString(d, offEIN)
	if err != nil {
		return nil, fmt.Errorf("decode ein: %w", err)
	}
	org.Name, err = readABIString(d, offName)
	if err != nil {
		return nil, fmt.Errorf("decode name: %w", err)
	}
	org.Domain, err = readABIString(d, offDomain)
	if err != nil {
		return nil, fmt.Errorf("decode domain: %w", err)
	}
	org.DIDURI, err = readABIString(d, offDID)
	if err != nil {
		return nil, fmt.Errorf("decode didUri: %w", err)
	}

	return org, nil
}

func decodeIssuer(hexData string) (*registry.Issuer, error) {
	data := common.FromHex(hexData)
	if len(data) < 64 {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	issuer := &registry.Issuer{}

	// Issuer struct: [off_to_struct]
	structOffset := readUint(data[0:32])
	d := data[structOffset:]

	// Head: [authority, off_did, active, createdAt]
	if len(d) < 4*32 {
		return nil, fmt.Errorf("struct data too short")
	}

	issuer.Authority = chain.Address(common.BytesToAddress(d[0:32]).Hex())
	offDID := readUint(d[32:64])
	issuer.Active = d[95] != 0
	issuer.CreatedAt = int64(binary.BigEndian.Uint64(d[120:128]))

	var err error
	issuer.DID, err = readABIString(d, offDID)
	if err != nil {
		return nil, fmt.Errorf("decode did: %w", err)
	}

	return issuer, nil
}

// Utility functions.

func padLeft(b []byte, size int) []byte {
	if len(b) >= size {
		return b[:size]
	}
	pad := make([]byte, size)
	copy(pad[size-len(b):], b)
	return pad
}

func padRight(b []byte, multiple int) []byte {
	padded := len(b)
	if padded%multiple != 0 {
		padded = padded + (multiple - padded%multiple)
	}
	result := make([]byte, padded)
	copy(result, b)
	return result
}

func paddedLen(n int) int {
	if n%32 == 0 {
		if n == 0 {
			return 32
		}
		return n
	}
	return n + (32 - n%32)
}

func readUint(b []byte) uint64 {
	if len(b) < 32 {
		return 0
	}
	n := new(big.Int).SetBytes(b)
	return n.Uint64()
}

func readABIString(data []byte, offset uint64) (string, error) {
	if offset+32 > uint64(len(data)) {
		return "", fmt.Errorf("string offset %d out of bounds (data len %d)", offset, len(data))
	}
	strLen := readUint(data[offset : offset+32])
	start := offset + 32
	end := start + strLen
	if end > uint64(len(data)) {
		return "", fmt.Errorf("string data out of bounds")
	}
	return strings.TrimRight(string(data[start:end]), "\x00"), nil
}
