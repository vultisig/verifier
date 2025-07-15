package vault

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"google.golang.org/protobuf/proto"
)

func TestMPCWrapperImp_KeyshareFromBytes(t *testing.T) {
	dataB64, err := os.ReadFile("testdata/vault_unencrypted.vult")
	require.Nil(t, err, "failed to read keyshare file")

	containerBytes, err := base64.StdEncoding.DecodeString(string(dataB64))
	require.Nil(t, err, "failed to decode base64 container")

	container := &vaultType.VaultContainer{}
	err = proto.Unmarshal(containerBytes, container)
	require.Nil(t, err, "failed to unmarshal container")

	vaultBytes, err := base64.StdEncoding.DecodeString(container.Vault)
	require.Nil(t, err, "failed to decode base64 vault")

	vault := &vaultType.Vault{}
	err = proto.Unmarshal(vaultBytes, vault)
	require.Nil(t, err, "failed to unmarshal vault")

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
