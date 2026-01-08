package gotest

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginSignerEndpoints(t *testing.T) {
	for i, plugin := range plugins {
		plugin := plugin
		pluginIndex := i
		t.Run(plugin.ID, func(t *testing.T) {
			if pluginIndex > 0 {
				time.Sleep(2 * time.Second)
			}

			apiKey := getPluginAPIKey(plugin.ID)
			policyID := getPluginPolicyID(pluginIndex)

			t.Run("Sign_NoAPIKey", func(t *testing.T) {
				reqBody := map[string]interface{}{
					"plugin_id":  plugin.ID,
					"public_key": fixture.Vault.PublicKey,
					"policy_id":  policyID,
					"messages":   []interface{}{},
				}

				resp, err := testClient.POST("/plugin-signer/sign", reqBody)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})

			t.Run("Sign_InvalidAPIKey", func(t *testing.T) {
				reqBody := map[string]interface{}{
					"plugin_id":  plugin.ID,
					"public_key": fixture.Vault.PublicKey,
					"policy_id":  policyID,
					"messages":   []interface{}{},
				}

				resp, err := testClient.WithAPIKey("invalid-api-key").POST("/plugin-signer/sign", reqBody)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})

			t.Run("Sign_EmptyMessages", func(t *testing.T) {
				reqBody := map[string]interface{}{
					"plugin_id":  plugin.ID,
					"public_key": fixture.Vault.PublicKey,
					"policy_id":  policyID,
					"messages":   []interface{}{},
				}

				resp, err := testClient.WithAPIKey(apiKey).POST("/plugin-signer/sign", reqBody)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})

			t.Run("Sign_ValidRequest", func(t *testing.T) {
				reqBody := map[string]interface{}{
					"plugin_id":        plugin.ID,
					"public_key":       fixture.Vault.PublicKey,
					"policy_id":        policyID,
					"transactions":     evmFixture.TxB64,
					"transaction_type": "evm",
					"messages": []map[string]interface{}{
						{
							"message":       evmFixture.MsgB64,
							"chain":         "Ethereum",
							"hash":          evmFixture.MsgSHA256B64,
							"hash_function": "SHA256",
						},
					},
				}

				resp, err := testClient.WithAPIKey(apiKey).POST("/plugin-signer/sign", reqBody)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var apiResp struct {
					Data struct {
						TaskIDs []string `json:"task_ids"`
					} `json:"data"`
				}
				err = ReadJSONResponse(resp, &apiResp)
				require.NoError(t, err)

				require.Len(t, apiResp.Data.TaskIDs, 1)
				taskID := apiResp.Data.TaskIDs[0]

				t.Run("GetSignResponse_NoAPIKey", func(t *testing.T) {
					resp, err := testClient.GET("/plugin-signer/sign/response/" + taskID)
					require.NoError(t, err)
					defer resp.Body.Close()

					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				})

				t.Run("GetSignResponse_WithAPIKey", func(t *testing.T) {
					resp, err := testClient.WithAPIKey(apiKey).GET("/plugin-signer/sign/response/" + taskID)
					require.NoError(t, err)
					defer resp.Body.Close()

					assert.True(t, resp.StatusCode >= 200, "expected any valid response")
				})
			})
		})
	}
}
