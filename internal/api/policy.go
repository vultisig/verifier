package api

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	v1 "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/recipes/engine"
	rtypes "github.com/vultisig/recipes/types"
	"google.golang.org/protobuf/proto"

	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/types"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/address"
	"github.com/vultisig/vultisig-go/common"
)

func (s *Server) validatePluginPolicy(ctx context.Context, policy types.PluginPolicy) error {
	if len(policy.Recipe) == 0 {
		return errors.New("recipe cannot be empty")
	}

	recipeBytes, err := base64.StdEncoding.DecodeString(policy.Recipe)
	if err != nil {
		return fmt.Errorf("fail to base64 decode recipe: %w", err)
	}

	recipe := &rtypes.Policy{}
	err = proto.Unmarshal(recipeBytes, recipe)
	if err != nil {
		return fmt.Errorf("fail to unmarshal recipe: %w", err)
	}

	spec, err := s.pluginService.GetPluginRecipeSpecification(ctx, policy.PluginID.String())
	if err != nil {
		return fmt.Errorf("failed to get plugin recipe specification: %w", err)
	}

	ngn, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("fail to create engine: %w", err)
	}
	err = ngn.ValidatePolicyWithSchema(recipe, spec)
	if err != nil {
		return fmt.Errorf("failed to validate plugin policy: %w", err)
	}
	if err := policy.ParseBillingFromRecipe(); err != nil {
		return fmt.Errorf("failed to parse billing from recipe: %w", err)
	}

	return nil
}

func (s *Server) CreatePluginPolicy(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestParseFailed))
	}
	if policy.ID.String() == "" {
		policy.ID = uuid.New()
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}
	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgPublicKeyMismatch))
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPolicySignature))
	}

	if err := s.validatePluginPolicy(c.Request().Context(), policy); err != nil {
		s.logger.WithError(err).Error("failed to validate plugin policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPluginPolicy))
	}

	newPolicy, err := s.policyService.CreatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to create plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPolicyCreateFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, newPolicy))
}

func (s *Server) getVault(publicKeyECDSA, pluginId string) (*v1.Vault, error) {

	fileName := common.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	vaultContent, err := s.vaultStorage.GetVault(fileName)
	if err != nil {
		return nil, errors.New("failed to get vault")
	}
	if vaultContent == nil {
		return nil, errors.New("vault not found")
	}

	v, err := common.DecryptVaultFromBackup(s.cfg.EncryptionSecret, vaultContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault,err: %w", err)
	}
	return v, nil
}

func (s *Server) verifyPolicySignature(policy types.PluginPolicy) bool {
	messageBytes, err := policyToMessageHex(policy)
	if err != nil {
		s.logger.WithError(err).Error("failed to convert policy to message hex")
		return false
	}
	signatureBytes, err := hex.DecodeString(strings.TrimPrefix(policy.Signature, "0x"))
	if err != nil {
		s.logger.WithError(err).Error("failed to decode signature bytes")
		return false
	}
	vault, err := s.getVault(policy.PublicKey, policy.PluginID.String())
	if err != nil {
		s.logger.WithError(err).Error("fail to get vault")
		return false
	}

	ethAddress, _, _, err := address.GetAddress(vault.PublicKeyEcdsa, vault.HexChainCode, common.Ethereum)
	if err != nil {
		s.logger.WithError(err).Error("failed to get Ethereum address")
		return false
	}

	success, err := sigutil.VerifyEthAddressSignature(ecommon.HexToAddress(ethAddress), messageBytes, signatureBytes)
	if err != nil {
		s.logger.WithError(err).Error("failed to verify signature")
		return false
	}
	return success
}

func policyToMessageHex(policy types.PluginPolicy) ([]byte, error) {
	delimiter := "*#*"

	fields := []string{
		policy.Recipe,
		policy.PublicKey,
		fmt.Sprintf("%d", policy.PolicyVersion),
		policy.PluginVersion,
	}

	for _, item := range fields {
		if strings.Contains(item, delimiter) {
			return nil, errors.New(msgInvalidPolicySignature)
		}
	}
	result := strings.Join(fields, delimiter)
	return []byte(result), nil
}

func (s *Server) UpdatePluginPolicyById(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestParseFailed))
	}

	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	oldPolicy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policy.ID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to get plugin policy, id:%s", policy.ID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPolicyGetFailed))
	}

	if oldPolicy.PublicKey != publicKey || policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgPublicKeyMismatch))
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgInvalidPolicySignature))
	}

	if err := s.validatePluginPolicy(c.Request().Context(), policy); err != nil {
		s.logger.WithError(err).Error("failed to validate plugin policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPluginPolicy))
	}
	updatedPolicy, err := s.policyService.UpdatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.WithError(err).Error("failed to update plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(fmt.Sprintf("failed to update policy: %s", policy.ID)))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, updatedPolicy))
}

func (s *Server) DeletePluginPolicyById(c echo.Context) error {
	var reqBody struct {
		Signature string `json:"signature"`
	}

	if err := c.Bind(&reqBody); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestParseFailed))
	}

	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		s.logger.Warn("Missing vault_public_key in context")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPolicyID))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to parse policyId")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPolicyID))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPolicyGetFailed))
	}

	if policy.PublicKey != publicKey {
		s.logger.Warn("Public key mismatch")
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgPublicKeyMismatch))
	}

	// This is because we have different signature stored in the database.
	policy.Signature = reqBody.Signature
	if !s.verifyPolicySignature(*policy) {
		s.logger.Error("Invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPolicySignature))
	}

	if err := s.policyService.DeletePolicy(c.Request().Context(), policyUUID, policy.PluginID, reqBody.Signature); err != nil {
		s.logger.WithError(err).Error("Failed to delete plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPolicyDeleteFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{"status": "deleted"}))
}

func (s *Server) GetPluginPolicyById(c echo.Context) error {
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}
	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPolicyID))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to parse policy ID")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPolicyID))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to get plugin policy, id:%s", policyUUID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPolicyGetFailed))
	}
	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgPublicKeyMismatch))
	}
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, policy))
}

func (s *Server) GetAllPluginPolicies(c echo.Context) error {
	includeInactiveString := c.QueryParam("includeInactive")
	var includeInactive bool = false
	if includeInactiveString == "true" {
		includeInactive = true
	}
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
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

	policies, err := s.policyService.GetPluginPolicies(c.Request().Context(), publicKey, vtypes.PluginID(pluginID), take, skip, includeInactive)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get policies for public_key: %s", publicKey)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPoliciesGetFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, policies))
}
