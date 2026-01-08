package gotest

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyEndpoints(t *testing.T) {
	for i, plugin := range plugins {
		plugin := plugin
		pluginIndex := i
		t.Run(plugin.ID, func(t *testing.T) {
			t.Parallel()

			policyID := getPluginPolicyID(pluginIndex)

			t.Run("GetPolicy_HappyPath", func(t *testing.T) {
				resp, err := testClient.WithJWT(jwtToken).GET("/plugin/policy/" + policyID)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var apiResp struct {
					Data struct {
						ID       string `json:"id"`
						PluginID string `json:"plugin_id"`
						Active   bool   `json:"active"`
					} `json:"data"`
				}
				err = ReadJSONResponse(resp, &apiResp)
				require.NoError(t, err)
				assert.Equal(t, policyID, apiResp.Data.ID)
				assert.Equal(t, plugin.ID, apiResp.Data.PluginID)
				assert.True(t, apiResp.Data.Active)
			})

			t.Run("GetAllPolicies_HappyPath", func(t *testing.T) {
				resp, err := testClient.WithJWT(jwtToken).GET("/plugin/policies/" + plugin.ID)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})

			t.Run("CreatePolicy_InvalidSignature", func(t *testing.T) {
				reqBody := map[string]interface{}{
					"id":             "00000000-0000-0000-0000-000000000001",
					"public_key":     fixture.Vault.PublicKey,
					"plugin_id":      plugin.ID,
					"plugin_version": "1.0.0",
					"policy_version": 1,
					"signature":      "0x" + strings.Repeat("0", 130),
					"recipe":         "CgA=",
					"billing":        []interface{}{},
					"active":         true,
				}

				resp, err := testClient.WithJWT(jwtToken).POST("/plugin/policy", reqBody)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})

			t.Run("CreatePolicy_NoAuth", func(t *testing.T) {
				reqBody := map[string]interface{}{
					"id":             "00000000-0000-0000-0000-000000000001",
					"public_key":     fixture.Vault.PublicKey,
					"plugin_id":      plugin.ID,
					"plugin_version": "1.0.0",
					"policy_version": 1,
					"signature":      "0x" + strings.Repeat("0", 130),
					"recipe":         "CgA=",
					"billing":        []interface{}{},
					"active":         true,
				}

				resp, err := testClient.POST("/plugin/policy", reqBody)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})

			t.Run("GetPolicy_NoAuth", func(t *testing.T) {
				resp, err := testClient.GET("/plugin/policy/test-id")
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})

			t.Run("GetPolicy_InvalidID", func(t *testing.T) {
				resp, err := testClient.WithJWT(jwtToken).GET("/plugin/policy/test-id")
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})
		})
	}
}
