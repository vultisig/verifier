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

	"github.com/vultisig/verifier/address"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/types"
	vtypes "github.com/vultisig/verifier/types"
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

	err = engine.NewEngine().ValidatePolicyWithSchema(recipe, spec)
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
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("failed to parse request"))
	}
	if policy.ID.String() == "" {
		policy.ID = uuid.New()
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("Failed to get vault public key"))
	}
	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage("Public key mismatch"))
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("Invalid policy signature"))
	}

	if err := s.validatePluginPolicy(c.Request().Context(), policy); err != nil {
		s.logger.WithError(err).Error("failed to validate plugin policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("Invalid plugin policy"))
	}

	newPolicy, err := s.policyService.CreatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to create plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to create policy"))
	}

	return c.JSON(http.StatusOK, newPolicy)
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
			return nil, errors.New("invalid policy signature")
		}
	}
	result := strings.Join(fields, delimiter)
	return []byte(result), nil
}

func (s *Server) UpdatePluginPolicyById(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("failed to parse request"))
	}

	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("Failed to get vault public key"))
	}

	oldPolicy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policy.ID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to get plugin policy, id:%s", policy.ID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get policy"))
	}

	if oldPolicy.PublicKey != publicKey || policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage("Public key mismatch"))
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage("Invalid policy signature"))
	}

	if err := s.validatePluginPolicy(c.Request().Context(), policy); err != nil {
		s.logger.WithError(err).Error("failed to validate plugin policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("Invalid plugin policy"))
	}
	updatedPolicy, err := s.policyService.UpdatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.WithError(err).Error("failed to update plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(fmt.Sprintf("failed to update policy: %s", policy.ID)))
	}

	return c.JSON(http.StatusOK, updatedPolicy)
}

func (s *Server) DeletePluginPolicyById(c echo.Context) error {
	var reqBody struct {
		Signature string `json:"signature"`
	}

	if err := c.Bind(&reqBody); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("failed to parse request"))
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("Failed to get vault public key"))
	}

	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid policy ID"))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to parse policy ID")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid policy ID"))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)

	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("fail to delete policy"))
	}

	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage("Public key mismatch"))
	}

	// This is because we have different signature stored in the database.
	policy.Signature = reqBody.Signature

	if !s.verifyPolicySignature(*policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("Invalid policy signature"))
	}

	if err := s.policyService.DeletePolicy(c.Request().Context(), policyUUID, policy.PluginID, reqBody.Signature); err != nil {
		s.logger.WithError(err).Error("failed to delete plugin policy")

		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to delete policy"))
	}

	return c.NoContent(http.StatusOK)
}
func (s *Server) GetPluginPolicyById(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("Invalid policy ID"))
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("Failed to get vault public key"))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to parse policy ID")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("Invalid policy ID"))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to get plugin policy,id:%s", policyUUID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get policy"))
	}
	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage("Public key mismatch"))
	}
	return c.JSON(http.StatusOK, policy)
}

func (s *Server) GetAllPluginPolicies(c echo.Context) error {
	includeInactiveString := c.QueryParam("includeInactive")
	var includeInactive bool = false
	if includeInactiveString == "true" {
		includeInactive = true
	}
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("Plugin ID is required"))
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("Failed to get vault public key"))
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
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get policies"))
	}

	return c.JSON(http.StatusOK, policies)
}
