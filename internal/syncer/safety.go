package syncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/vultisig/verifier/plugin/safety"
	"github.com/vultisig/verifier/types"
)

const safetyEndpoint = "/plugin/safety"

func (s *Syncer) SyncSafetyToPlugin(ctx context.Context, pluginID types.PluginID, flags []safety.ControlFlag) error {
	flagBytes, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("failed to marshal flags: %w", err)
	}

	serverInfo, err := s.getServerInfo(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("failed to get server info: %w", err)
	}

	url := serverInfo.Addr + safetyEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(flagBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serverInfo.ApiKey)

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to sync safety to plugin(%s): %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to sync safety to plugin(%s): status %d, body: %s", url, resp.StatusCode, string(body))
	}

	return nil
}
