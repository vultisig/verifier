package gotest

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyEndpoints(t *testing.T) {
	for _, plugin := range plugins {
		plugin := plugin
		t.Run(plugin.ID, func(t *testing.T) {
			t.Parallel()

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
