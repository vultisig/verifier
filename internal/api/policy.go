package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/types"
)

func (s *Server) CreatePluginPolicy(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}
	if policy.ID == "" {
		policy.ID = uuid.NewString()
	}
	// TODO: validate if the policy is correct
	if !s.verifyPolicySignature(policy, false) {
		s.logger.Error("invalid policy signature")
		message := map[string]interface{}{
			"message": "Authorization failed",
			"error":   "Invalid policy signature",
		}
		return c.JSON(http.StatusBadRequest, message)
	}

	newPolicy, err := s.policyService.CreatePolicy(c.Request().Context(), policy)
	if err != nil {
		err = fmt.Errorf("failed to create plugin policy: %w", err)
		message := map[string]interface{}{
			"message": "failed to create policy",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, newPolicy)
}

func (s *Server) verifyPolicySignature(policy types.PluginPolicy, update bool) bool {
	msgHex, err := policyToMessageHex(policy, update)
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to convert policy to message hex: %w", err))
		return false
	}

	msgBytes, err := hex.DecodeString(strings.TrimPrefix(msgHex, "0x"))
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to decode message bytes: %w", err))
		return false
	}

	signatureBytes, err := hex.DecodeString(strings.TrimPrefix(policy.Signature, "0x"))
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to decode signature bytes: %w", err))
		return false
	}

	isVerified, err := sigutil.VerifySignature(policy.PublicKey, policy.ChainCodeHex, policy.DerivePath, msgBytes, signatureBytes)
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to verify signature: %w", err))
		return false
	}
	return isVerified
}

func policyToMessageHex(policy types.PluginPolicy, isUpdate bool) (string, error) {
	if !isUpdate {
		policy.ID = ""
	}
	// signature is not part of the message that is signed
	policy.Signature = ""

	serializedPolicy, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("failed to serialize policy")
	}
	return hex.EncodeToString(serializedPolicy), nil
}

func (s *Server) UpdatePluginPolicyById(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if !s.verifyPolicySignature(policy, true) {
		s.logger.Error("invalid policy signature")
		message := map[string]interface{}{
			"message": "Authorization failed",
			"error":   "Invalid policy signature",
		}
		return c.JSON(http.StatusForbidden, message)
	}

	updatedPolicy, err := s.policyService.UpdatePolicy(c.Request().Context(), policy)
	if err != nil {
		err = fmt.Errorf("failed to update plugin policy: %w", err)
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to update policy: %s", policy.ID),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, updatedPolicy)
}

func (s *Server) DeletePluginPolicyById(c echo.Context) error {
	var reqBody struct {
		Signature string `json:"signature"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	policyID := c.Param("policyId")
	if policyID == "" {
		err := fmt.Errorf("policy ID is required")
		message := map[string]interface{}{
			"message": "failed to delete policy",
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, message)
	}

	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyID)
	if err != nil {
		err = fmt.Errorf("failed to get policy: %w", err)
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to get policy: %s", policyID),
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	// This is because we have different signature stored in the database.
	policy.Signature = reqBody.Signature

	if !s.verifyPolicySignature(policy, true) {
		s.logger.Error("invalid policy signature")
		message := map[string]interface{}{
			"message": "Authorization failed",
			"error":   "Invalid policy signature",
		}
		return c.JSON(http.StatusForbidden, message)
	}

	if err := s.policyService.DeletePolicy(c.Request().Context(), policyID, reqBody.Signature); err != nil {
		err = fmt.Errorf("failed to delete policy: %w", err)
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to delete policy: %s", policyID),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.NoContent(http.StatusNoContent)
}
func (s *Server) GetPolicySchema(c echo.Context) error {
	pluginType := c.Request().Header.Get("plugin_type") // this is a unique identifier; this won't be needed once the DCA and Payroll are separate services
	if pluginType == "" {
		err := fmt.Errorf("missing required header: plugin_type")
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to get policy schema for plugin: %s", pluginType),
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, message)
	}

	keyPath := filepath.Join("plugin", pluginType, "dcaPluginUiSchema.json")

	jsonData, err := os.ReadFile(keyPath)
	if err != nil {
		message := map[string]interface{}{
			"message": fmt.Sprintf("missing schema for plugin: %s", pluginType),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, message)
	}

	var data map[string]interface{}
	jsonErr := json.Unmarshal([]byte(jsonData), &data)
	if jsonErr != nil {

		message := map[string]interface{}{
			"message": fmt.Sprintf("could not unmarshal json: %s", jsonErr),
			"error":   jsonErr.Error(),
		}
		s.logger.Error(jsonErr)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, data)
}

func (s *Server) GetPluginPolicyById(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		err := fmt.Errorf("policy ID is required")
		message := map[string]interface{}{
			"message": "failed to get policy",
			"error":   err.Error(),
		}
		s.logger.Error(err)

		return c.JSON(http.StatusBadRequest, message)
	}

	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyID)
	if err != nil {
		err = fmt.Errorf("failed to get policy: %w", err)
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to get policy: %s", policyID),
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, policy)
}

func (s *Server) GetAllPluginPolicies(c echo.Context) error {
	publicKey := c.Request().Header.Get("public_key")
	if publicKey == "" {
		err := fmt.Errorf("missing required header: public_key")
		message := map[string]interface{}{
			"message": "failed to get policies",
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, message)
	}

	pluginType := c.Request().Header.Get("plugin_type")
	if pluginType == "" {
		err := fmt.Errorf("missing required header: plugin_type")
		message := map[string]interface{}{
			"message": "failed to get policies",
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, message)
	}

	policies, err := s.policyService.GetPluginPolicies(c.Request().Context(), publicKey, pluginType)
	if err != nil {
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to get policies for public_key: %s", publicKey),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, policies)
}
