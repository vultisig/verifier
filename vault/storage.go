package vault

import vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"

type Storage interface {
	GetVault(publicKeyEcdsa string) (*vaultType.Vault, error)
	SaveVault(file string, encodedVaultContent string) error
}
