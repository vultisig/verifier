package gotest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultEndpoints(t *testing.T) {
	for i, plugin := range plugins {
		plugin := plugin
		pluginIndex := i
		t.Run(plugin.ID, func(t *testing.T) {
			t.Parallel()

			t.Run("ReshareVault_Idempotent", func(t *testing.T) {
				sessionID := getPluginSessionID(pluginIndex)
				reqBody := map[string]interface{}{
					"name":               fixture.Vault.Name,
					"public_key":         fixture.Vault.PublicKey,
					"session_id":         sessionID,
					"hex_encryption_key": fixture.Reshare.HexEncryptionKey,
					"hex_chain_code":     fixture.Reshare.HexChainCode,
					"local_party_id":     fixture.Reshare.LocalPartyID,
					"old_parties":        fixture.Reshare.OldParties,
					"email":              fixture.Reshare.Email,
					"plugin_id":          plugin.ID,
				}

				resp, err := testClient.WithJWT(jwtToken).POST("/vault/reshare", reqBody)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var apiResp struct {
					Data string `json:"data"`
				}
				err = ReadJSONResponse(resp, &apiResp)
				require.NoError(t, err)

				assert.Equal(t, "already_exists", apiResp.Data)
			})

			t.Run("VaultExists", func(t *testing.T) {
				resp, err := testClient.WithJWT(jwtToken).GET("/vault/exist/" + plugin.ID + "/" + fixture.Vault.PublicKey)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var apiResp struct {
					Data string `json:"data"`
				}
				err = ReadJSONResponse(resp, &apiResp)
				require.NoError(t, err)

				assert.Equal(t, "ok", apiResp.Data)
			})
		})
	}
}
