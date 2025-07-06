package api

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	v1 "github.com/vultisig/commondata/go/vultisig/vault/v1"
	rtypes "github.com/vultisig/recipes/types"
	"google.golang.org/protobuf/proto"

	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/vultisig/verifier/address"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/types"
	ptypes "github.com/vultisig/verifier/types"
)

// Parses the base64 wrapped protobuf encoded recipe and validates it (TODO)
func (s *Server) validatePluginPolicy(policy types.PluginPolicy) error {
	if len(policy.Recipe) == 0 {
		return errors.New("recipe cannot be empty")
	}

	recipeBytes, err := base64.StdEncoding.DecodeString(policy.Recipe)
	if err != nil {
		return fmt.Errorf("fail to base64 decode recipe: %w", err)
	}

	var recipe rtypes.Policy
	if err := proto.Unmarshal(recipeBytes, &recipe); err != nil {
		return fmt.Errorf("fail to unmarshal recipe: %w", err)
	}

	// TODO: validate the recipe
	return nil
}

func (s *Server) CreatePluginPolicy(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "failed to parse request", err.Error()))
	}
	if policy.ID.String() == "" {
		policy.ID = uuid.New()
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
	}
	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponse(http.StatusForbidden, "Public key mismatch", ""))
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid policy signature", ""))
	}

	if err := s.validatePluginPolicy(policy); err != nil {
		s.logger.WithError(err).Error("failed to validate plugin policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid plugin policy", err.Error()))
	}

	newPolicy, err := s.policyService.CreatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.Errorf("failed to create plugin policy: %s", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to create policy", err.Error()))
	}

	return c.JSON(http.StatusOK, newPolicy)
}

func (s *Server) getVault(publicKeyECDSA, pluginId string) (*v1.Vault, error) {
	if len(s.cfg.EncryptionSecret) == 0 {
		return nil, fmt.Errorf("no encryption secret")
	}
	fileName := common.GetVaultBackupFilename(publicKeyECDSA, pluginId)
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
		s.logger.Errorf("failed to verify signature: %s", err)
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
			return nil, fmt.Errorf("invalid policy signature")
		}
	}
	result := strings.Join(fields, delimiter)
	return []byte(result), nil
}

func (s *Server) UpdatePluginPolicyById(c echo.Context) error {
	var policy types.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "failed to parse request", err.Error()))
	}

	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
	}

	oldPolicy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policy.ID)
	if err != nil {
		s.logger.Errorf("failed to get plugin policy: %s,id:%s", err, policy.ID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get policy", err.Error()))
	}

	if oldPolicy.PublicKey != publicKey || policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponse(http.StatusForbidden, "Public key mismatch", ""))
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusForbidden, NewErrorResponse(http.StatusForbidden, "Invalid policy signature", ""))
	}

	if err := s.validatePluginPolicy(policy); err != nil {
		s.logger.WithError(err).Error("failed to validate plugin policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid plugin policy", err.Error()))
	}
	updatedPolicy, err := s.policyService.UpdatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.Errorf("failed to update plugin policy: %s", err)
		return c.JSON(http.StatusInternalServerError,
			NewErrorResponse(http.StatusInternalServerError, fmt.Sprintf("failed to update policy: %s", policy.ID), err.Error()))
	}

	return c.JSON(http.StatusOK, updatedPolicy)
}

func (s *Server) DeletePluginPolicyById(c echo.Context) error {
	var reqBody struct {
		Signature string `json:"signature"`
	}

	if err := c.Bind(&reqBody); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "failed to parse request", err.Error()))
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
	}

	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "invalid policy ID", ""))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.Errorf("failed to parse policy ID: %s", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "invalid policy ID", err.Error()))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)

	if err != nil {
		s.logger.Errorf("failed to get plugin policy: %s", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "fail to delete policy", err.Error()))
	}

	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponse(http.StatusForbidden, "Public key mismatch", ""))
	}

	// This is because we have different signature stored in the database.
	policy.Signature = reqBody.Signature

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid policy", ""))
	}

	if err := s.policyService.DeletePolicy(c.Request().Context(), policyUUID, policy.PluginID, reqBody.Signature); err != nil {
		s.logger.Errorf("failed to delete plugin policy: %s", err)

		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to delete policy", err.Error()))
	}

	return c.NoContent(http.StatusOK)
}
func (s *Server) GetPluginPolicyById(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid policy ID", ""))
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.Errorf("failed to parse policy ID: %s", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid policy ID", err.Error()))
	}
	policy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		s.logger.Errorf("failed to get plugin policy: %s,id:%s", err, policyUUID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get policy", err.Error()))
	}
	if policy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponse(http.StatusForbidden, "Public key mismatch", ""))
	}
	return c.JSON(http.StatusOK, policy)
}

func (s *Server) GetAllPluginPolicies(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Plugin ID is required", ""))
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
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
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, fmt.Sprintf("failed to get policies for public_key: %s", publicKey), err.Error()))
	}

	return c.JSON(http.StatusOK, policies)
}
