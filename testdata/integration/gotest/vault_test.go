package gotest

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultEndpoints(t *testing.T) {
	time.Sleep(2 * time.Second)

	for _, plugin := range plugins {
		plugin := plugin
		t.Run(plugin.ID, func(t *testing.T) {
			time.Sleep(500 * time.Millisecond)

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

			t.Run("GetVault_HappyPath", func(t *testing.T) {
				time.Sleep(500 * time.Millisecond)
				resp, err := testClient.WithJWT(jwtToken).GET("/vault/get/" + plugin.ID + "/" + fixture.Vault.PublicKey)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		})
	}
}
