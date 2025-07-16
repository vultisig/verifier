package vault

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"google.golang.org/protobuf/proto"
)

func TestMPCWrapperImp_KeyshareFromBytes(t *testing.T) {
	vault, err := unpackKeyshare("testdata/vault_unencrypted.vult")
	require.Nil(t, err, "unpackKeyshare")

	ecdsa, eddsa := sortKeyshares(vault)
	require.NotNil(t, ecdsa, "ECDSA keyshare should not be nil")
	require.NotNil(t, eddsa, "EDDSA keyshare should not be nil")

	ecdsaBytes, err := base64.StdEncoding.DecodeString(ecdsa.Keyshare)
	require.Nil(t, err, "failed to decode ECDSA keyshare")

	_, err = NewMPCWrapperImp(false).KeyshareFromBytes(ecdsaBytes)
	require.Nil(t, err, "KeyshareFromBytes")

	// TODO: https://github.com/vultisig/verifier/issues/257
	//eddsaBytes, err := base64.StdEncoding.DecodeString(eddsa.Keyshare)
	//require.Nil(t, err, "failed to decode EDDSA keyshare")
	//_, err = NewMPCWrapperImp(true).KeyshareFromBytes(eddsaBytes)
	//require.Nil(t, err, "KeyshareFromBytes")
}

func unpackKeyshare(path string) (*vaultType.Vault, error) {
	dataB64, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	containerBytes, err := base64.StdEncoding.DecodeString(string(dataB64))
	if err != nil {
		return nil, fmt.Errorf("base64.StdEncoding.DecodeString: %w", err)
	}

	container := &vaultType.VaultContainer{}
	err = proto.Unmarshal(containerBytes, container)
	if err != nil {
		return nil, fmt.Errorf("proto.Unmarshal: %w", err)
	}

	vaultBytes, err := base64.StdEncoding.DecodeString(container.Vault)
	if err != nil {
		return nil, fmt.Errorf("base64.StdEncoding.DecodeString: %w", err)
	}

	vault := &vaultType.Vault{}
	err = proto.Unmarshal(vaultBytes, vault)
	if err != nil {
		return nil, fmt.Errorf("proto.Unmarshal: %w", err)
	}
	return vault, nil
}

func TestMPCWrapperImp_SignSessionFromSetup(t *testing.T) {
	vault, err := unpackKeyshare("testdata/vault_unencrypted.vult")
	require.Nil(t, err, "unpackKeyshare")

	ecdsa, _ := sortKeyshares(vault)
	ecdsaBytes, err := base64.StdEncoding.DecodeString(ecdsa.Keyshare)
	require.Nil(t, err, "failed to decode ECDSA keyshare")

	mpc := NewMPCWrapperImp(false)

	keyshare, err := mpc.KeyshareFromBytes(ecdsaBytes)
	require.Nil(t, err, "KeyshareFromBytes")

	id, err := mpc.KeyshareKeyID(keyshare)
	require.Nil(t, err, "KeyshareKeyID")

	setupMsg, err := mpc.SignSetupMsgNew(
		id,
		[]byte("m/44'/60'/0'/0/0"),
		make([]byte, 32),
		toIdsSlice([]string{
			"verifier-1",
			"payroll-plugin-0000-1",
		}),
	)
	require.Nil(t, err, "SignSetupMsgNew")

	_, err = mpc.SignSessionFromSetup(setupMsg, []byte("verifier-1"), keyshare)
	require.Nil(t, err, "SignSessionFromSetup")
}

func sortKeyshares(vault *vaultType.Vault) (*vaultType.Vault_KeyShare, *vaultType.Vault_KeyShare) {
	var (
		ecdsa *vaultType.Vault_KeyShare
		eddsa *vaultType.Vault_KeyShare
	)
	for _, keyshare := range vault.KeyShares {
		if keyshare.PublicKey == vault.PublicKeyEcdsa {
			ecdsa = keyshare
		}
		if keyshare.PublicKey == vault.PublicKeyEddsa {
			eddsa = keyshare
		}
	}
	return ecdsa, eddsa
}
