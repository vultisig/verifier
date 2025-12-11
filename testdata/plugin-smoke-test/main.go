package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const userAgent = "plugin-smoke-test/1.0"

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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: plugin-smoke-test <proposed.yaml|plugin-url>")
		os.Exit(1)
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
		fmt.Printf("❌ Failed to read file: %v\n", err)
		os.Exit(1)
	}

	var proposed ProposedYAML
	if err := yaml.Unmarshal(data, &proposed); err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🧪 Testing %d plugins from %s\n\n", len(proposed.Plugins), arg)

	failed := 0
	passed := 0

	for i, plugin := range proposed.Plugins {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Plugin %d/%d\n", i+1, len(proposed.Plugins))
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		if testPlugin(plugin.ServerEndpoint, plugin.ID, plugin.Title) {
			passed++
		} else {
			failed++
		}
		fmt.Println()
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Summary: %d passed, %d failed\n", passed, failed)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if failed > 0 {
		os.Exit(1)
	}
}

func testPlugin(url, expectedID, expectedTitle string) bool {
	fmt.Printf("  URL:   %s\n", url)
	if expectedID != "" {
		fmt.Printf("  ID:    %s\n", expectedID)
	}
	if expectedTitle != "" {
		fmt.Printf("  Title: %s\n", expectedTitle)
	}
	fmt.Println()

	allPassed := true

	// Test 1: Recipe Specification
	fmt.Printf("  [1/8] GET /plugin/recipe-specification ... ")
	recipeURL := strings.TrimSuffix(url, "/") + "/plugin/recipe-specification"
	spec, err := testRecipeSpecification(recipeURL, expectedID, expectedTitle)
	if err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Error: %v\n", err)
		allPassed = false
		// Without a valid spec, we can't test endpoints that depend on plugin_id
		fmt.Printf("\n  ❌ Recipe specification failed - skipping remaining tests\n")
		return allPassed
	}

	fmt.Printf("✅\n")
	fmt.Printf("        Plugin ID: %s\n", spec.PluginID)
	fmt.Printf("        Plugin Name: %s\n", spec.PluginName)

	// From here on, spec is guaranteed to be non-nil
	pluginID := spec.PluginID
	if pluginID == "" {
		fmt.Printf("\n  ❌ Plugin ID is empty - cannot test remaining endpoints\n")
		return false
	}

	// Test 2: Vault Exist
	fmt.Printf("  [2/8] GET /vault/exist/:pluginId/:publicKey ... ")
	if err := testVaultExist(url, pluginID); err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Error: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("✅\n")
	}

	// Test 3: Vault Get
	fmt.Printf("  [3/8] GET /vault/get/:pluginId/:publicKey ... ")
	if err := testVaultGet(url, pluginID); err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Error: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("✅\n")
	}

	// Test 4: Vault Delete
	fmt.Printf("  [4/8] DELETE /vault/:pluginId/:publicKey ... ")
	if err := testVaultDelete(url, pluginID); err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Note: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("✅\n")
	}

	// Test 5: Vault Reshare
	fmt.Printf("  [5/8] POST /vault/reshare ... ")
	if err := testVaultReshare(url, pluginID); err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Error: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("✅\n")
	}

	// Test 6: Create Policy
	fmt.Printf("  [6/8] POST /plugin/policy ... ")
	if err := testCreatePolicy(url, pluginID); err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Error: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("✅\n")
	}

	// Test 7: Update Policy
	fmt.Printf("  [7/8] PUT /plugin/policy ... ")
	if err := testUpdatePolicy(url, pluginID); err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Error: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("✅\n")
	}

	// Test 8: Delete Policy
	fmt.Printf("  [8/8] DELETE /plugin/policy/:policyId ... ")
	if err := testDeletePolicy(url); err != nil {
		fmt.Printf("❌\n")
		fmt.Printf("        Error: %v\n", err)
		allPassed = false
	} else {
		fmt.Printf("✅\n")
	}

	if allPassed {
		fmt.Printf("\n  ✅ All tests passed!\n")
	} else {
		fmt.Printf("\n  ❌ Some tests failed\n")
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

	// Accept either 200 (exists) or 400 (doesn't exist) as valid responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func testVaultGet(baseURL, pluginID string) error {
	dummyKey := "0x0000000000000000000000000000000000000000000000000000000000000000"
	url := fmt.Sprintf("%s/vault/get/%s/%s", strings.TrimSuffix(baseURL, "/"), pluginID, dummyKey)

	resp, err := doGet(url)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Accept 200, 400, or 404 as valid (vault likely doesn't exist)
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusBadRequest &&
		resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
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

	// Accept 200, 400, or 404 as valid - endpoint exists and behaves sanely
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusBadRequest &&
		resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func testVaultReshare(baseURL, pluginID string) error {
	url := strings.TrimSuffix(baseURL, "/") + "/vault/reshare"

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

	// Accept 200 (success) or 400 (validation error) as valid
	// We're just testing the endpoint exists and responds
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func testCreatePolicy(baseURL, pluginID string) error {
	url := strings.TrimSuffix(baseURL, "/") + "/plugin/policy"

	// Use minimal test payload - will likely fail auth/validation but tests endpoint exists
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

	// Use minimal test payload
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
	// Use a dummy policy ID
	policyID := "00000000-0000-0000-0000-000000000001"
	url := fmt.Sprintf("%s/plugin/policy/%s", strings.TrimSuffix(baseURL, "/"), policyID)

	// Send signature in request body
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
