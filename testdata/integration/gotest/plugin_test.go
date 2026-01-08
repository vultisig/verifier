package gotest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginEndpoints(t *testing.T) {
	for _, plugin := range plugins {
		plugin := plugin
		t.Run(plugin.ID, func(t *testing.T) {
			t.Parallel()

			t.Run("GetPluginDetails", func(t *testing.T) {
				resp, err := testClient.GET("/plugins/" + plugin.ID)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var apiResp struct {
					Data struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					} `json:"data"`
				}
				err = ReadJSONResponse(resp, &apiResp)
				require.NoError(t, err)

				assert.Equal(t, plugin.ID, apiResp.Data.ID)
				assert.NotEmpty(t, apiResp.Data.Title)
			})

			t.Run("GetRecipeSpecification", func(t *testing.T) {
				resp, err := testClient.GET("/plugins/" + plugin.ID + "/recipe-specification")
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var apiResp struct {
					Data struct {
						PluginID   string `json:"plugin_id"`
						PluginName string `json:"plugin_name"`
					} `json:"data"`
				}
				err = ReadJSONResponse(resp, &apiResp)
				require.NoError(t, err)

				assert.Equal(t, plugin.ID, apiResp.Data.PluginID)
				assert.NotEmpty(t, apiResp.Data.PluginName)
			})
		})
	}
}
