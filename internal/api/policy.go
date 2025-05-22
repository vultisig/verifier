package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	v1 "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/mobile-tss-lib/tss"

	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/types"
	ptypes "github.com/vultisig/verifier/types"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{
		Message: message,
	}
}
func (s *Server) CreatePluginPolicy(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}
	if policy.ID.String() == "" {
		policy.ID = uuid.New()
	}
	// TODO: validate if the policy is correct
	if !s.verifyPolicySignature(policy, false) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid policy signature"))
	}

	newPolicy, err := s.policyService.CreatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.Errorf("failed to create plugin policy: %s", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create policy"))
	}

	return c.JSON(http.StatusOK, newPolicy)
}
func (s *Server) getVault(publicKeyECDSA string) (*v1.Vault, error) {
	if len(s.cfg.EncryptionSecret) == 0 {
		return nil, fmt.Errorf("no encryption secret")
	}
	fileName := common.GetVaultBackupFilename(publicKeyECDSA)
	vaultContent, err := s.vaultStorage.GetVault(fileName)
	if err != nil {
		s.logger.WithError(err).Error("fail to get vault")
		return nil, fmt.Errorf("failed to get vault")
	}
	if vaultContent == nil {
		s.logger.Error("vault not found")
		return nil, fmt.Errorf("vault not found")
	}

	v, err := common.DecryptVaultFromBackup(s.cfg.EncryptionSecret, vaultContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault,err: %w", err)
	}
	return v, nil
}

func (s *Server) verifyPolicySignature(policy types.PluginPolicy, update bool) bool {
	msgHex, err := policyToMessageHex(policy, update)
	if err != nil {
		s.logger.Errorf("failed to convert policy to message hex: %s", err)
		return false
	}
	messageBytes, err := hex.DecodeString(msgHex)
	if err != nil {
		s.logger.WithError(err).Error("failed to decode message bytes")
		return false
	}
	signatureBytes, err := hex.DecodeString(strings.TrimPrefix(policy.Signature, "0x"))
	if err != nil {
		s.logger.WithError(err).Error("failed to decode signature bytes")
		return false
	}
	vault, err := s.getVault(policy.PublicKey)
	if err != nil {
		s.logger.WithError(err).Error("fail to get vault")
		return false
	}

	derivedPublicKey, err := tss.GetDerivedPubKey(vault.PublicKeyEcdsa, vault.HexChainCode, common.Ethereum.GetDerivePath(), false)
	if err != nil {
		s.logger.WithError(err).Error("failed to get derived public key")
		return false
	}

	isVerified, err := sigutil.VerifyPolicySignature(derivedPublicKey, messageBytes, signatureBytes)
	if err != nil {
		s.logger.Errorf("failed to verify signature: %s", err)
		return false
	}
	return isVerified
}

func policyToMessageHex(policy types.PluginPolicy, isUpdate bool) (string, error) {
	if !isUpdate {
		policy.ID = uuid.Nil
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
		return c.JSON(http.StatusForbidden, NewErrorResponse("Invalid policy signature"))
	}

	updatedPolicy, err := s.policyService.UpdatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.Errorf("failed to update plugin policy: %s", err)
		return c.JSON(http.StatusInternalServerError,
			NewErrorResponse(fmt.Sprintf("failed to update policy: %s", policy.ID)))
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
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid policy ID"))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.Errorf("failed to parse policy ID: %s", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid policy ID"))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		s.logger.Errorf("failed to get plugin policy: %s", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("fail to delete policy"))
	}
	// This is because we have different signature stored in the database.
	policy.Signature = reqBody.Signature

	if !s.verifyPolicySignature(policy, true) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid policy"))
	}

	if err := s.policyService.DeletePolicy(c.Request().Context(), policyUUID, policy.PluginID, reqBody.Signature); err != nil {
		s.logger.Errorf("failed to delete plugin policy: %s", err)

		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to delete policy"))
	}

	return c.NoContent(http.StatusOK)
}
func (s *Server) GetPluginPolicyById(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid policy ID"))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.Errorf("failed to parse policy ID: %s", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid policy ID"))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		s.logger.Errorf("failed to get plugin policy: %s,id:%s", err, policyUUID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get policy"))
	}
	return c.JSON(http.StatusOK, policy)
}

func (s *Server) GetAllPluginPolicies(c echo.Context) error {
	publicKey := c.Request().Header.Get("public_key")
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to get policies"))
	}

	pluginID := c.Request().Header.Get("plugin_id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to get policies"))
	}

	skip, err := strconv.Atoi(c.QueryParam("skip"))

	if err != nil {
		skip = 0
	}

	take, err := strconv.Atoi(c.QueryParam("take"))

	if err != nil {
		take = 20
	}

	if take > 100 {
		take = 100
	}

	policies, err := s.policyService.GetPluginPolicies(c.Request().Context(), publicKey, ptypes.PluginID(pluginID), take, skip)
	if err != nil {
		s.logger.WithError(err).Error(fmt.Sprintf("Failed to get policies for public_key: %s", publicKey))
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(fmt.Sprintf("failed to get policies for public_key: %s", publicKey)))
	}

	return c.JSON(http.StatusOK, policies)
}
