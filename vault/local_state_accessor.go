package vault

import (
	"fmt"

	vaultType "github.com/vultisig/commondata/go/vultisig/vault/v1"
)

type LocalStateAccessorImp struct {
	Vault *vaultType.Vault
	cache map[string]string
}

// NewLocalStateAccessorImp creates a new instance of LocalStateAccessorImp
func NewLocalStateAccessorImp(vault *vaultType.Vault) *LocalStateAccessorImp {
	localStateAccessor := &LocalStateAccessorImp{
		Vault: vault,
		cache: make(map[string]string),
	}
	return localStateAccessor
}

func (l *LocalStateAccessorImp) GetLocalState(pubKey string) (string, error) {
	if l.Vault != nil {
		for _, item := range l.Vault.KeyShares {
			if item.PublicKey == pubKey {
				return item.Keyshare, nil
			}
		}
		return "", fmt.Errorf("%s keyshare does not exist", pubKey)
	}
	return l.cache[pubKey], nil
}

func (l *LocalStateAccessorImp) SaveLocalState(pubKey, localState string) error {
	l.cache[pubKey] = localState
	return nil
}

func (l *LocalStateAccessorImp) GetLocalCacheState(pubKey string) (string, error) {
	return l.cache[pubKey], nil
}
