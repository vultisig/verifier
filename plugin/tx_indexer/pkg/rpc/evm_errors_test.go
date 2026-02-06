package rpc

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestErrorRegistrySize(t *testing.T) {
	size := ErrorRegistrySize()
	require.Greater(t, size, 0, "error registry should not be empty")
}

func TestDecodeEVMRevert_ErrorString(t *testing.T) {
	// Error(string) with message "insufficient balance"
	// Selector: 0x08c379a0
	// ABI-encoded: offset (32) + length (20) + "insufficient balance" padded
	data := common.FromHex("0x08c379a0" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000014" +
		"696e73756666696369656e742062616c616e6365000000000000000000000000")

	msg, ok := DecodeEVMRevert(data)
	require.True(t, ok)
	require.Equal(t, "insufficient balance", msg)
}

func TestDecodeEVMRevert_Panic(t *testing.T) {
	// Accepts either geth's abi.UnpackRevert output or our formatPanicCode fallback
	tests := []struct {
		name         string
		code         string
		acceptedMsgs []string
	}{
		{"assertion_failed", "01", []string{"assert(false)", "panic: assertion failed"}},
		{"arithmetic_overflow", "11", []string{"arithmetic underflow or overflow", "panic: arithmetic overflow"}},
		{"division_by_zero", "12", []string{"division or modulo by zero", "panic: division by zero"}},
		{"array_out_of_bounds", "32", []string{"out-of-bounds access of an array or bytesN", "panic: array index out of bounds"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := common.FromHex("0x4e487b71" +
				"00000000000000000000000000000000000000000000000000000000000000" + tc.code)

			msg, ok := DecodeEVMRevert(data)
			require.True(t, ok)
			require.Contains(t, tc.acceptedMsgs, msg)
		})
	}
}

func TestDecodeEVMRevert_TooShort(t *testing.T) {
	data := common.FromHex("0x08c3")
	msg, ok := DecodeEVMRevert(data)
	require.False(t, ok)
	require.Empty(t, msg)
}

func TestDecodeEVMRevert_UnknownSelector(t *testing.T) {
	data := common.FromHex("0xdeadbeef" +
		"0000000000000000000000000000000000000000000000000000000000000000")

	msg, ok := DecodeEVMRevert(data)
	require.False(t, ok)
	require.Empty(t, msg)
}

func TestDecodeStandard_EmptyMessage(t *testing.T) {
	// Error(string) with empty message
	data := common.FromHex("0x08c379a0" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000000")

	msg, ok := decodeStandard(data)
	require.False(t, ok)
	require.Empty(t, msg)
}

func TestFormatPanicCode_UnknownCode(t *testing.T) {
	msg := formatPanicCode(0xff)
	require.Equal(t, "panic: code 0xff", msg)
}

func TestDecodeEVMRevert_CustomError(t *testing.T) {
	if ErrorRegistrySize() == 0 {
		t.Skip("error registry is empty")
	}

	for selector, errDef := range errorRegistry {
		if len(errDef.Inputs) == 0 {
			data := selector[:]
			msg, ok := DecodeEVMRevert(data)
			require.True(t, ok, "should decode custom error %s", errDef.Name)
			require.Equal(t, errDef.Name, msg)
			return
		}
	}

	t.Skip("no zero-input custom errors found in registry")
}

func TestFormatErrorWithArgs(t *testing.T) {
	tests := []struct {
		name     string
		errName  string
		args     []interface{}
		expected string
	}{
		{"no_args", "SomeError", nil, "SomeError"},
		{"single_string", "SomeError", []interface{}{"hello"}, "SomeError(hello)"},
		{"multiple_args", "SomeError", []interface{}{"hello", uint64(42)}, "SomeError(hello, 42)"},
		{"bytes_arg", "SomeError", []interface{}{[]byte{0xde, 0xad}}, "SomeError(0xdead)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatErrorWithArgs(tc.errName, tc.args)
			require.Equal(t, tc.expected, result)
		})
	}
}
