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

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/logo/%s%s", pluginID, hashStr, ext)

	oldLogoKey, _, _, err := s.db.GetPluginS3Keys(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get old S3 keys")
	}

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload logo to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	logoURL := s.assetStorage.GetPublicURL(s3Key)
	err = s.db.UpdatePluginLogo(c.Request().Context(), pluginID, logoURL, s3Key)
	if err != nil {
		if delErr := s.assetStorage.Delete(c.Request().Context(), s3Key); delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded logo from S3")
		}
		s.logger.WithError(err).Error("failed to update plugin logo in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	if oldLogoKey != "" && oldLogoKey != s3Key {
		if delErr := s.assetStorage.Delete(c.Request().Context(), oldLogoKey); delErr != nil {
			s.logger.WithError(delErr).Warn("failed to delete old logo from S3")
		}
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"logo_url": logoURL,
		"s3_key":   s3Key,
	}))
}

func (s *Server) UploadPluginThumbnail(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/thumbnail/%s%s", pluginID, hashStr, ext)

	_, oldThumbKey, _, err := s.db.GetPluginS3Keys(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get old S3 keys")
	}

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload thumbnail to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	thumbnailURL := s.assetStorage.GetPublicURL(s3Key)
	err = s.db.UpdatePluginThumbnail(c.Request().Context(), pluginID, thumbnailURL, s3Key)
	if err != nil {
		if delErr := s.assetStorage.Delete(c.Request().Context(), s3Key); delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded thumbnail from S3")
		}
		s.logger.WithError(err).Error("failed to update plugin thumbnail in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	if oldThumbKey != "" && oldThumbKey != s3Key {
		if delErr := s.assetStorage.Delete(c.Request().Context(), oldThumbKey); delErr != nil {
			s.logger.WithError(delErr).Warn("failed to delete old thumbnail from S3")
		}
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"thumbnail_url": thumbnailURL,
		"s3_key":        s3Key,
	}))
}

func (s *Server) AddPluginImage(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))

	_, _, images, err := s.db.GetPluginS3Keys(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin images")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	if len(images) >= maxGalleryImages {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgTooManyImages))
	}

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	imageID := uuid.New().String()
	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/gallery/%s%s", pluginID, imageID, ext)

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload image to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	imageURL := s.assetStorage.GetPublicURL(s3Key)

	sortOrder := len(images)
	if so := c.QueryParam("sort_order"); so != "" {
		if parsed, err := strconv.Atoi(so); err == nil {
			sortOrder = parsed
		}
	}
	var zIndex int
	if zi := c.QueryParam("z_index"); zi != "" {
		if parsed, err := strconv.Atoi(zi); err == nil {
			zIndex = parsed
		}
	}

	newImage := itypes.PluginImage{
		ID:        imageID,
		URL:       imageURL,
		S3Key:     s3Key,
		Caption:   c.QueryParam("caption"),
		AltText:   c.QueryParam("alt_text"),
		SortOrder: sortOrder,
		ZIndex:    zIndex,
	}
	images = append(images, newImage)

	err = s.db.UpdatePluginImages(c.Request().Context(), pluginID, images)
	if err != nil {
		if delErr := s.assetStorage.Delete(c.Request().Context(), s3Key); delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded image from S3")
		}
		s.logger.WithError(err).Error("failed to update plugin images in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]any{
		"image": newImage,
	}))
}

func (s *Server) UpdatePluginImage(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))
	imageID := c.Param("imageId")

	data, contentType, err := s.readUploadedImage(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(err.Error()))
	}

	_, _, images, err := s.db.GetPluginS3Keys(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin images")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	var oldS3Key string
	var imageIndex int = -1
	for i, img := range images {
		if img.ID == imageID {
			oldS3Key = img.S3Key
			imageIndex = i
			break
		}
	}

	if imageIndex == -1 {
		return c.JSON(http.StatusNotFound, NewErrorResponseWithMessage(msgImageNotFound))
	}

	ext := getExtensionForContentType(contentType)
	s3Key := fmt.Sprintf("plugins/%s/gallery/%s%s", pluginID, imageID, ext)

	err = s.assetStorage.Upload(c.Request().Context(), s3Key, data, contentType)
	if err != nil {
		s.logger.WithError(err).Error("failed to upload image to S3")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	imageURL := s.assetStorage.GetPublicURL(s3Key)
	images[imageIndex].URL = imageURL
	images[imageIndex].S3Key = s3Key
	if caption := c.QueryParam("caption"); caption != "" {
		images[imageIndex].Caption = caption
	}
	if altText := c.QueryParam("alt_text"); altText != "" {
		images[imageIndex].AltText = altText
	}
	if so := c.QueryParam("sort_order"); so != "" {
		if parsed, err := strconv.Atoi(so); err == nil {
			images[imageIndex].SortOrder = parsed
		}
	}
	if zi := c.QueryParam("z_index"); zi != "" {
		if parsed, err := strconv.Atoi(zi); err == nil {
			images[imageIndex].ZIndex = parsed
		}
	}

	err = s.db.UpdatePluginImages(c.Request().Context(), pluginID, images)
	if err != nil {
		if delErr := s.assetStorage.Delete(c.Request().Context(), s3Key); delErr != nil {
			s.logger.WithError(delErr).Warn("failed to cleanup uploaded image from S3")
		}
		s.logger.WithError(err).Error("failed to update plugin images in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageUploadFailed))
	}

	if oldS3Key != "" && oldS3Key != s3Key {
		if delErr := s.assetStorage.Delete(c.Request().Context(), oldS3Key); delErr != nil {
			s.logger.WithError(delErr).Warn("failed to delete old image from S3")
		}
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]any{
		"image": images[imageIndex],
	}))
}

func (s *Server) DeletePluginImage(c echo.Context) error {
	if s.assetStorage == nil {
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgAssetStorageNotConfigured))
	}

	pluginID := types.PluginID(c.Param("pluginId"))
	imageID := c.Param("imageId")

	_, _, images, err := s.db.GetPluginS3Keys(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin images")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageDeleteFailed))
	}

	var s3KeyToDelete string
	var newImages []itypes.PluginImage
	for _, img := range images {
		if img.ID == imageID {
			s3KeyToDelete = img.S3Key
		} else {
			newImages = append(newImages, img)
		}
	}

	if s3KeyToDelete == "" {
		return c.JSON(http.StatusNotFound, NewErrorResponseWithMessage(msgImageNotFound))
	}

	err = s.db.UpdatePluginImages(c.Request().Context(), pluginID, newImages)
	if err != nil {
		s.logger.WithError(err).Error("failed to update plugin images in DB")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgImageDeleteFailed))
	}

	if s3KeyToDelete != "" {
		deleteErr := s.assetStorage.Delete(c.Request().Context(), s3KeyToDelete)
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
