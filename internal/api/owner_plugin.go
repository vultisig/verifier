package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/clientutil"
	"github.com/vultisig/verifier/internal/storage/postgres"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

const (
	msgCannotRemoveLastOwner     = "cannot remove last owner"
	msgInvalidECDSAPublicKey     = "invalid public key: ECDSA public key expected (found in vault info)"
	msgImageUploadFailed         = "failed to upload image"
	msgImageDeleteFailed         = "failed to delete image"
	msgImageNotFound             = "image not found"
	msgInvalidImageFormat        = "invalid image format, only JPEG, PNG, and GIF are allowed"
	msgImageTooLarge             = "image too large, maximum size is 5MB"
	msgAssetStorageNotConfigured = "asset storage not configured"

	maxImageSize     = 5 * 1024 * 1024 // 5MB
	maxGalleryImages = 5
)

var msgTooManyImages = fmt.Sprintf("maximum of %d gallery images reached, delete existing images to upload new ones", maxGalleryImages)

func (s *Server) ListOwnedPlugins(c echo.Context) error {
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	pluginIDs, err := s.db.GetPluginsByOwner(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugins by owner")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgInternalError))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]any{
		"plugin_ids": pluginIDs,
	}))
}

func (s *Server) AddCoOwner(c echo.Context) error {
	pluginID := types.PluginID(c.Param("pluginId"))
	callerPubKey, ok := c.Get("vault_public_key").(string)
	if !ok || callerPubKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	var req struct {
		PublicKey string `json:"public_key"`
	}
	err := c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidRequestFormat))
	}
	if !clientutil.IsValidSecp256k1PublicKey(req.PublicKey) {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidECDSAPublicKey))
	}

	err = s.db.AddOwner(c.Request().Context(), pluginID, req.PublicKey, itypes.PluginOwnerAddedViaOwnerAPI, callerPubKey)
	if err != nil {
		s.logger.WithError(err).Error("failed to add co-owner")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgInternalError))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{"message": "owner added"}))
}

func (s *Server) RemoveCoOwner(c echo.Context) error {
	pluginID := types.PluginID(c.Param("pluginId"))
	publicKey := c.Param("publicKey")
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPublicKeyFormat))
	}

	err := s.db.DeactivateOwnerIfNotLast(c.Request().Context(), pluginID, publicKey)
	if err != nil {
		if errors.Is(err, postgres.ErrWouldRemoveLastOwner) {
			return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgCannotRemoveLastOwner))
		}
		s.logger.WithError(err).Error("failed to remove co-owner")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgInternalError))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{"message": "owner removed"}))
}

func (s *Server) UploadPluginLogo(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/logo/%s%s", pluginID, hashStr, ext)

	oldImage, err := s.db.GetPluginImageByType(c.Request().Context(), pluginID, itypes.PluginImageTypeLogo)
	if err != nil {
		s.logger.WithError(err).Error("failed to get old logo")
	}

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload logo to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	record, err := s.db.ReplacePluginImage(c.Request().Context(), pluginID, itypes.PluginImageTypeLogo, s3Key, publicKey)
	if err != nil {
		delErr := s.assetStorage.Delete(c.Request().Context(), s3Key)
		if delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded logo from S3")
		}
		s.logger.WithError(err).Error("failed to update plugin logo in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	if oldImage != nil && oldImage.S3Path != s3Key {
		delErr := s.assetStorage.Delete(c.Request().Context(), oldImage.S3Path)
		if delErr != nil {
			s.logger.WithError(delErr).Warn("failed to delete old logo from S3")
		}
	}

	logoURL := s.assetStorage.GetPublicURL(s3Key)
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"logo_url": logoURL,
		"s3_key":   s3Key,
		"image_id": record.ID.String(),
	}))
}

func (s *Server) UploadPluginThumbnail(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/thumbnail/%s%s", pluginID, hashStr, ext)

	oldImage, err := s.db.GetPluginImageByType(c.Request().Context(), pluginID, itypes.PluginImageTypeThumbnail)
	if err != nil {
		s.logger.WithError(err).Error("failed to get old thumbnail")
	}

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload thumbnail to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	record, err := s.db.ReplacePluginImage(c.Request().Context(), pluginID, itypes.PluginImageTypeThumbnail, s3Key, publicKey)
	if err != nil {
		delErr := s.assetStorage.Delete(c.Request().Context(), s3Key)
		if delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded thumbnail from S3")
		}
		s.logger.WithError(err).Error("failed to update plugin thumbnail in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	if oldImage != nil && oldImage.S3Path != s3Key {
		delErr := s.assetStorage.Delete(c.Request().Context(), oldImage.S3Path)
		if delErr != nil {
			s.logger.WithError(delErr).Warn("failed to delete old thumbnail from S3")
		}
	}

	thumbnailURL := s.assetStorage.GetPublicURL(s3Key)
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"thumbnail_url": thumbnailURL,
		"s3_key":        s3Key,
		"image_id":      record.ID.String(),
	}))
}

func (s *Server) AddPluginImage(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	existingImages, err := s.db.GetPluginImagesByPluginIDs(c.Request().Context(), []types.PluginID{pluginID})
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin images")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	mediaCount := 0
	for _, img := range existingImages {
		if img.ImageType == itypes.PluginImageTypeMedia {
			mediaCount++
		}
	}
	if mediaCount >= maxGalleryImages {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgTooManyImages))
	}

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	imageID := uuid.New()
	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/media/%s%s", pluginID, imageID.String(), ext)

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload image to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	imageOrder, err := s.db.GetNextMediaOrder(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Warn("failed to get next media order, using 0")
		imageOrder = 0
	}
	if so := c.QueryParam("sort_order"); so != "" {
		if parsed, parseErr := strconv.Atoi(so); parseErr == nil {
			imageOrder = parsed
		}
	}

	record, err := s.db.CreatePluginImage(c.Request().Context(), itypes.PluginImageCreateParams{
		PluginID:            pluginID,
		ImageType:           itypes.PluginImageTypeMedia,
		S3Path:              s3Key,
		ImageOrder:          imageOrder,
		UploadedByPublicKey: publicKey,
	})
	if err != nil {
		delErr := s.assetStorage.Delete(c.Request().Context(), s3Key)
		if delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded image from S3")
		}
		s.logger.WithError(err).Error("failed to create plugin image in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	imageURL := s.assetStorage.GetPublicURL(s3Key)
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]any{
		"image": map[string]any{
			"id":         record.ID.String(),
			"url":        imageURL,
			"s3_key":     s3Key,
			"sort_order": record.ImageOrder,
		},
	}))
}

func (s *Server) UpdatePluginImage(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))
	imageIDStr := c.Param("imageId")
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	imageID, err := uuid.Parse(imageIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid image ID"))
	}

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	existingImages, err := s.db.GetPluginImagesByPluginIDs(c.Request().Context(), []types.PluginID{pluginID})
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin images")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	var oldImage itypes.PluginImageRecord
	found := false
	for _, img := range existingImages {
		if img.ID == imageID {
			oldImage = img
			found = true
			break
		}
	}

	if !found {
		return c.JSON(http.StatusNotFound, NewErrorResponseWithMessage(msgImageNotFound))
	}

	newImageID := uuid.New()
	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/media/%s%s", pluginID, newImageID.String(), ext)

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload image to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	imageOrder := oldImage.ImageOrder
	if so := c.QueryParam("sort_order"); so != "" {
		if parsed, parseErr := strconv.Atoi(so); parseErr == nil {
			imageOrder = parsed
		}
	}

	_, err = s.db.SoftDeletePluginImage(c.Request().Context(), pluginID, imageID)
	if err != nil {
		delErr := s.assetStorage.Delete(c.Request().Context(), s3Key)
		if delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded image from S3")
		}
		s.logger.WithError(err).Error("failed to delete old plugin image")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	record, err := s.db.CreatePluginImage(c.Request().Context(), itypes.PluginImageCreateParams{
		PluginID:            pluginID,
		ImageType:           itypes.PluginImageTypeMedia,
		S3Path:              s3Key,
		ImageOrder:          imageOrder,
		UploadedByPublicKey: publicKey,
	})
	if err != nil {
		delErr := s.assetStorage.Delete(c.Request().Context(), s3Key)
		if delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded image from S3")
		}
		s.logger.WithError(err).Error("failed to create new plugin image in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	delErr := s.assetStorage.Delete(c.Request().Context(), oldImage.S3Path)
	if delErr != nil {
		s.logger.WithError(delErr).Warn("failed to delete old image from S3")
	}

	imageURL := s.assetStorage.GetPublicURL(s3Key)
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]any{
		"image": map[string]any{
			"id":         record.ID.String(),
			"url":        imageURL,
			"s3_key":     s3Key,
			"sort_order": record.ImageOrder,
		},
	}))
}

func (s *Server) DeletePluginImage(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))
	imageIDStr := c.Param("imageId")

	imageID, err := uuid.Parse(imageIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid image ID format"))
	}

	s3Path, err := s.db.SoftDeletePluginImage(c.Request().Context(), pluginID, imageID)
	if err != nil {
		s.logger.WithError(err).Error("failed to soft delete plugin image")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageDeleteFailed))
	}

	if s3Path != "" {
		deleteErr := s.assetStorage.Delete(c.Request().Context(), s3Path)
		if deleteErr != nil {
			s.logger.WithError(deleteErr).Warn("failed to delete image from S3 (best-effort)")
		}
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{"message": "image deleted"}))
}

var (
	errImageRequired      = errors.New("image file is required")
	errImageOpenFailed    = errors.New("failed to open uploaded file")
	errImageReadFailed    = errors.New("failed to read uploaded file")
	errImageTooLarge      = errors.New(msgImageTooLarge)
	errImageInvalidFormat = errors.New(msgInvalidImageFormat)
)

func (s *Server) readUploadedImage(c echo.Context) ([]byte, string, error) {
	file, err := c.FormFile("image")
	if err != nil {
		return nil, "", errImageRequired
	}

	src, err := file.Open()
	if err != nil {
		return nil, "", errImageOpenFailed
	}
	defer src.Close()

	limitedReader := io.LimitReader(src, maxImageSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, "", errImageReadFailed
	}

	if len(data) > maxImageSize {
		return nil, "", errImageTooLarge
	}

	contentType := http.DetectContentType(data)
	if !isAllowedImageType(contentType) {
		return nil, "", errImageInvalidFormat
	}

	return data, contentType, nil
}

func isAllowedImageType(contentType string) bool {
	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
	}
	return allowed[contentType]
}

func getExtensionForContentType(contentType string) string {
	ext := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
	}
	if e, ok := ext[contentType]; ok {
		return e
	}
	return path.Ext(strings.TrimPrefix(contentType, "image/"))
}
