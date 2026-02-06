package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	abi_embed "github.com/vultisig/recipes/chain/evm/abi"
)

var (
	errorRegistry map[[4]byte]abi.Error
	initOnce      sync.Once
)

func initRegistry() {
	errorRegistry = make(map[[4]byte]abi.Error)

	entries, err := abi_embed.Dir.ReadDir(".")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := abi_embed.Dir.ReadFile(entry.Name())
		if err != nil {
			continue
		}
		parsed, err := abi.JSON(bytes.NewReader(data))
		if err != nil {
			continue
		}
		for _, errDef := range parsed.Errors {
			var selector [4]byte
			copy(selector[:], errDef.ID[:4])
			if _, exists := errorRegistry[selector]; !exists {
				errorRegistry[selector] = errDef
			}
		}
	}
}

func DecodeEVMRevert(data []byte) (string, bool) {
	initOnce.Do(initRegistry)

	if len(data) < 4 {
		return "", false
	}

	msg, ok := decodeStandard(data)
	if ok {
		return msg, true
	}

	msg, ok = decodeCustom(data)
	if ok {
		return msg, true
	}

	return "", false
}

var (
	panicSelector = [4]byte{0x4e, 0x48, 0x7b, 0x71}
	errorSelector = [4]byte{0x08, 0xc3, 0x79, 0xa0}
)

func decodeStandard(data []byte) (string, bool) {
	msg, err := abi.UnpackRevert(data)
	if err == nil && msg != "" {
		return msg, true
	}

	if len(data) >= 36 {
		var selector [4]byte
		copy(selector[:], data[:4])
		if selector == panicSelector {
			code := new(big.Int).SetBytes(data[4:36]).Uint64()
			return formatPanicCode(code), true
		}
	}

	return "", false
}

func formatPanicCode(code uint64) string {
	switch code {
	case 0x00:
		return "panic: generic/compiler inserted"
	case 0x01:
		return "panic: assertion failed"
	case 0x11:
		return "panic: arithmetic overflow"
	case 0x12:
		return "panic: division by zero"
	case 0x21:
		return "panic: invalid enum value"
	case 0x22:
		return "panic: storage byte array encoding error"
	case 0x31:
		return "panic: pop on empty array"
	case 0x32:
		return "panic: array index out of bounds"
	case 0x41:
		return "panic: memory allocation overflow"
	case 0x51:
		return "panic: zero initialized variable"
	default:
		return fmt.Sprintf("panic: code 0x%x", code)
	}
}

func decodeCustom(data []byte) (string, bool) {
	if len(data) < 4 {
		return "", false
	}

	var selector [4]byte
	copy(selector[:], data[:4])

	errDef, ok := errorRegistry[selector]
	if !ok {
		return "", false
	}

	if len(data) == 4 {
		return errDef.Name, true
	}

	values, err := errDef.Inputs.Unpack(data[4:])
	if err != nil {
		return errDef.Name, true
	}

	return formatErrorWithArgs(errDef.Name, values), true
}

func formatErrorWithArgs(name string, args []interface{}) string {
	if len(args) == 0 {
		return name
	}

	parts := make([]string, 0, len(args))
	for _, arg := range args {
		switch v := arg.(type) {
		case []byte:
			parts = append(parts, fmt.Sprintf("0x%x", v))
		case common.Address:
			parts = append(parts, v.Hex())
		case [20]byte:
			parts = append(parts, common.Address(v).Hex())
		default:
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
	}

	return fmt.Sprintf("%s(%s)", name, strings.Join(parts, ", "))
}

type dataErrorInterface interface {
	ErrorData() interface{}
}

func extractRevertBytesFromError(err error) []byte {
	if err == nil {
		return nil
	}

	var de dataErrorInterface
	if errors.As(err, &de) {
		errData := de.ErrorData()
		if errData != nil {
			switch v := errData.(type) {
			case []byte:
				return v
			case string:
				if strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X") {
					return common.FromHex(v)
				}
				return nil
			}
		}
	}

	errStr := err.Error()
	idx := strings.Index(errStr, "0x")
	if idx >= 0 {
		hexPart := errStr[idx:]
		endIdx := strings.IndexAny(hexPart, " \n\t\"'")
		if endIdx > 0 {
			hexPart = hexPart[:endIdx]
		}
		decoded := common.FromHex(hexPart)
		if len(decoded) >= 4 {
			var sel [4]byte
			copy(sel[:], decoded[:4])
			if sel == errorSelector || sel == panicSelector {
				return decoded
			}
			initOnce.Do(initRegistry)
			if _, ok := errorRegistry[sel]; ok {
				return decoded
			}
		}
	}

	return nil
}

func ErrorRegistrySize() int {
	initOnce.Do(initRegistry)
	return len(errorRegistry)
}
