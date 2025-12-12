package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const userAgent = "plugin-smoke-test/1.0"

var log = logrus.New()

// Shared HTTP client for all requests
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// doRequest executes an HTTP request with standard headers
func doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)
	return httpClient.Do(req)
}

// doGet performs a GET request with standard headers
func doGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return doRequest(req)
}

// doPost performs a POST request with standard headers
func doPost(url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return doRequest(req)
}

type Plugin struct {
	ID             string `yaml:"id"`
	Title          string `yaml:"title"`
	ServerEndpoint string `yaml:"server_endpoint"`
}

type ProposedYAML struct {
	Plugins []Plugin `yaml:"plugins"`
}

type RecipeSpec struct {
	PluginID   string `json:"plugin_id"`
	PluginName string `json:"plugin_name"`
}

type ReshareResponse struct {
	KeyShare string `json:"key_share"`
}

func main() {
	log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		FullTimestamp:    false,
	})

	if len(os.Args) < 2 {
		log.Fatal("Usage: plugin-smoke-test <proposed.yaml|plugin-url>")
	}

	arg := os.Args[1]

	// Check if it's a URL or file path
	if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		// Single plugin URL
		pluginID := ""
		pluginTitle := ""
		if len(os.Args) >= 3 {
			pluginID = os.Args[2]
		}
		if len(os.Args) >= 4 {
			pluginTitle = os.Args[3]
		}

		passed := testPlugin(arg, pluginID, pluginTitle)
		if !passed {
			os.Exit(1)
		}
		return
	}

	// Test all plugins from YAML file
	data, err := os.ReadFile(arg)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	var proposed ProposedYAML
	if err := yaml.Unmarshal(data, &proposed); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	log.Infof("Testing %d plugins from %s", len(proposed.Plugins), arg)

	concurrentRuns := 20
	var wg sync.WaitGroup
	var passed, failed atomic.Int32
	sem := make(chan struct{}, concurrentRuns)

	wg.Add(len(proposed.Plugins))
	for _, plugin := range proposed.Plugins {
		go func(p Plugin) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if testPlugin(p.ServerEndpoint, p.ID, p.Title) {
				passed.Add(1)
			} else {
				failed.Add(1)
			}
		}(plugin)
	}
	wg.Wait()

	log.Infof("Summary: %d passed, %d failed", passed.Load(), failed.Load())

	if failed.Load() > 0 {
		os.Exit(1)
	}
}

func testPlugin(url, expectedID, expectedTitle string) bool {
	entry := log.WithFields(logrus.Fields{
		"url":  url,
		"id":   expectedID,
		"name": expectedTitle,
	})

	entry.Info("Testing plugin")

	allPassed := true

	// Test 1: Recipe Specification
	recipeURL := strings.TrimSuffix(url, "/") + "/plugin/recipe-specification"
	spec, err := testRecipeSpecification(recipeURL, expectedID, expectedTitle)
	if err != nil {
		entry.WithField("test", "recipe-specification").Errorf("Failed: %v", err)
		entry.Error("Recipe specification failed - skipping remaining tests")
		return false
	}

	entry.WithField("test", "recipe-specification").WithFields(logrus.Fields{
		"plugin_id":   spec.PluginID,
		"plugin_name": spec.PluginName,
	}).Info("Success")

	// From here on, spec is guaranteed to be non-nil
	pluginID := spec.PluginID
	if pluginID == "" {
		entry.Error("Plugin ID is empty - cannot test remaining endpoints")
		return false
	}

	// Test 2: Vault Exist
	if err := testVaultExist(url, pluginID); err != nil {
		entry.WithField("test", "vault-exist").Errorf("Failed: %v", err)
		allPassed = false
	} else {
		entry.WithField("test", "vault-exist").Info("Success")
	}

	// Test 3: Vault Get
	if err := testVaultGet(url, pluginID); err != nil {
		entry.WithField("test", "vault-get").Errorf("Failed: %v", err)
		allPassed = false
	} else {
		entry.WithField("test", "vault-get").Info("Success")
	}

	// Test 4: Vault Delete
	if err := testVaultDelete(url, pluginID); err != nil {
		entry.WithField("test", "vault-delete").Errorf("Failed: %v", err)
		allPassed = false
	} else {
		entry.WithField("test", "vault-delete").Info("Success")
	}

	// Test 5: Vault Reshare
	if err := testVaultReshare(url, pluginID); err != nil {
		entry.WithField("test", "vault-reshare").Errorf("Failed: %v", err)
		allPassed = false
	} else {
		entry.WithField("test", "vault-reshare").Info("Success")
	}

	// Test 6: Create Policy
	if err := testCreatePolicy(url, pluginID); err != nil {
		entry.WithField("test", "create-policy").Errorf("Failed: %v", err)
		allPassed = false
	} else {
		entry.WithField("test", "create-policy").Info("Success")
	}

	// Test 7: Update Policy
	if err := testUpdatePolicy(url, pluginID); err != nil {
		entry.WithField("test", "update-policy").Errorf("Failed: %v", err)
		allPassed = false
	} else {
		entry.WithField("test", "update-policy").Info("Success")
	}

	// Test 8: Delete Policy
	if err := testDeletePolicy(url); err != nil {
		entry.WithField("test", "delete-policy").Errorf("Failed: %v", err)
		allPassed = false
	} else {
		entry.WithField("test", "delete-policy").Info("Success")
	}

	if allPassed {
		entry.Info("All tests passed")
	} else {
		entry.Error("Some tests failed")
	}

	return allPassed
}

func testRecipeSpecification(url, expectedID, expectedTitle string) (*RecipeSpec, error) {
	resp, err := doGet(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}

	// Validate required fields
	pluginID, ok := result["plugin_id"].(string)
	if !ok || pluginID == "" {
		return nil, fmt.Errorf("missing or invalid plugin_id")
	}

	pluginName, ok := result["plugin_name"].(string)
	if !ok || pluginName == "" {
		return nil, fmt.Errorf("missing or invalid plugin_name")
	}

	if _, ok := result["requirements"]; !ok {
		return nil, fmt.Errorf("missing requirements field")
	}

	if _, ok := result["configuration"]; !ok {
		return nil, fmt.Errorf("missing configuration field")
	}

	if _, ok := result["configuration_example"]; !ok {
		return nil, fmt.Errorf("missing configuration_example field")
	}

	if sr, ok := result["supported_resources"]; !ok {
		return nil, fmt.Errorf("missing supported_resources field")
	} else if _, isArray := sr.([]interface{}); !isArray {
		return nil, fmt.Errorf("supported_resources must be an array")
	}

	if perms, ok := result["permissions"]; !ok {
		return nil, fmt.Errorf("missing permissions field")
	} else if _, isArray := perms.([]interface{}); !isArray {
		return nil, fmt.Errorf("permissions must be an array")
	}

	// Validate expected values if provided
	if expectedID != "" && pluginID != expectedID {
		return nil, fmt.Errorf("plugin_id mismatch: expected %s, got %s", expectedID, pluginID)
	}

	if expectedTitle != "" && pluginName != expectedTitle {
		return nil, fmt.Errorf("plugin_name mismatch: expected %s, got %s", expectedTitle, pluginName)
	}

	spec := &RecipeSpec{
		PluginID:   pluginID,
		PluginName: pluginName,
	}

	return spec, nil
}

func testVaultExist(baseURL, pluginID string) error {
	// Use a dummy public key that won't exist
	dummyKey := "0x0000000000000000000000000000000000000000000000000000000000000000"
	url := fmt.Sprintf("%s/vault/exist/%s/%s", strings.TrimSuffix(baseURL, "/"), pluginID, dummyKey)

	resp, err := doGet(url)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Accept 200, 400, 401, or 404 as valid responses
	if resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusNotFound {
		// Validate JSON response if body is non-empty
		if len(body) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("invalid response JSON: %v", err)
			}
		}
		return nil
	}

	return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
}

func testVaultGet(baseURL, pluginID string) error {
	dummyKey := "0x0000000000000000000000000000000000000000000000000000000000000000"
	url := fmt.Sprintf("%s/vault/get/%s/%s", strings.TrimSuffix(baseURL, "/"), pluginID, dummyKey)

	resp, err := doGet(url)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Endpoint must respond with valid JSON structure
	// Accept 200 (vault exists with data) or 400/401/404 (vault doesn't exist or auth required)
	if resp.StatusCode == http.StatusOK {
		// Validate response is valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("invalid response JSON: %v", err)
		}
		if len(result) == 0 {
			return fmt.Errorf("response body is empty")
		}
		return nil
	}

	if resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusNotFound {
		// Validate error response is valid JSON if body present
		if len(body) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("invalid error response JSON: %v", err)
			}
		}
		return nil // Acceptable response
	}

	return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
}

func testVaultDelete(baseURL, pluginID string) error {
	dummyKey := "0x0000000000000000000000000000000000000000000000000000000000000000"
	url := fmt.Sprintf("%s/vault/%s/%s", strings.TrimSuffix(baseURL, "/"), pluginID, dummyKey)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := doRequest(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Accept 200, 400, 401, or 404 as valid - endpoint exists and behaves sanely
	if resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusNotFound {
		// Validate JSON response if body is non-empty
		if len(body) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("invalid response JSON: %v", err)
			}
		}
		return nil
	}

	return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
}

func testVaultReshare(baseURL, pluginID string) error {
	url := strings.TrimSuffix(baseURL, "/") + "/vault/reshare"

	// NOTE: This is a lightweight API contract smoke test.
	// It verifies that plugin endpoints exist and respond with proper HTTP status codes.
	// Full E2E tests (including real reshare + key_share validation via verifier+DB)
	// are planned as a separate integration test suite.

	// Use minimal valid payload
	payload := fmt.Sprintf(`{
		"name": "smoke-test-vault",
		"public_key": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"session_id": "00000000-0000-0000-0000-000000000000",
		"hex_encryption_key": "0000000000000000000000000000000000000000000000000000000000000000",
		"hex_chain_code": "0000000000000000000000000000000000000000000000000000000000000000",
		"local_party_id": "test-party",
		"old_parties": ["party1"],
		"email": "test@example.com",
		"plugin_id": "%s"
	}`, pluginID)
	resp, err := doPost(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Accept 200, 400, 401, or 404 as valid - endpoint exists and responds
	if resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusNotFound {
		// Validate JSON response if body is non-empty
		if len(body) > 0 {
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("invalid response JSON: %v", err)
			}
		}
		return nil
	}

	return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
}

func testCreatePolicy(baseURL, pluginID string) error {
	url := strings.TrimSuffix(baseURL, "/") + "/plugin/policy"

	// Use valid test payload
	payload := fmt.Sprintf(`{
		"id": "00000000-0000-0000-0000-000000000001",
		"public_key": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"plugin_id": "%s",
		"plugin_version": "1.0.0",
		"policy_version": 1,
		"signature": "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"recipe": "CgA=",
		"billing": [],
		"active": true
	}`, pluginID)
	resp, err := doPost(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Accept 200, 400 (validation), or 401 (auth) as valid responses
	// We're just checking the endpoint exists
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusBadRequest &&
		resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func testUpdatePolicy(baseURL, pluginID string) error {
	url := strings.TrimSuffix(baseURL, "/") + "/plugin/policy"

	// Use valid test payload
	payload := fmt.Sprintf(`{
		"id": "00000000-0000-0000-0000-000000000001",
		"public_key": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"plugin_id": "%s",
		"plugin_version": "1.0.0",
		"policy_version": 2,
		"signature": "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"recipe": "CgA=",
		"billing": [],
		"active": true
	}`, pluginID)

	req, err := http.NewRequest("PUT", url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := doRequest(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Accept 200, 400 (validation), 401 (auth), or 404 (not found) as valid
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusBadRequest &&
		resp.StatusCode != http.StatusUnauthorized &&
		resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func testDeletePolicy(baseURL string) error {
	// Use a test policy ID
	policyID := "00000000-0000-0000-0000-000000000001"
	url := fmt.Sprintf("%s/plugin/policy/%s", strings.TrimSuffix(baseURL, "/"), policyID)

	payload := `{"signature": "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}`
	req, err := http.NewRequest("DELETE", url, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := doRequest(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Accept 200, 400 (validation), 401 (auth), or 404 (not found) as valid
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusBadRequest &&
		resp.StatusCode != http.StatusUnauthorized &&
		resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
