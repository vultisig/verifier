package portal

import (
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/storage/postgres/queries"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type GetUploadURLRequest struct {
	ImageType   string `json:"image_type"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
}

type GetUploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	ImageID   string `json:"image_id"`
	S3Path    string `json:"s3_path"`
	ExpiresAt string `json:"expires_at"`
}

type UpdateImageRequest struct {
	Visible    *bool `json:"visible,omitempty"`
	ImageOrder *int  `json:"image_order,omitempty"`
}

type ReorderImagesRequest struct {
	ImageIDs []string `json:"image_ids"`
}

type PluginImageResponse struct {
	ImageID     string `json:"image_id"`
	PluginID    string `json:"plugin_id"`
	ImageType   string `json:"image_type"`
	URL         string `json:"url"`
	S3Path      string `json:"s3_path"`
	ImageOrder  int    `json:"image_order"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	Visible     bool   `json:"visible"`
	Deleted     bool   `json:"deleted"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (s *Server) ListPluginImages(c echo.Context) error {
	ctx := c.Request().Context()
	pluginID := c.Param("id")
	address := c.Get("address").(string)

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  queries.PluginID(pluginID),
		PublicKey: address,
	})
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
	}

	includeHidden := owner.Role != queries.PluginOwnerRoleViewer
	includeDeleted := c.QueryParam("include_deleted") == "true" && owner.Role == queries.PluginOwnerRoleAdmin

	var imageTypeFilter *itypes.PluginImageType
	if it := c.QueryParam("image_type"); it != "" {
		t := itypes.PluginImageType(it)
		imageTypeFilter = &t
	}

	images, err := s.db.ListPluginImages(ctx, postgres.ListPluginImagesParams{
		PluginID:       types.PluginID(pluginID),
		ImageType:      imageTypeFilter,
		IncludeHidden:  includeHidden,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		s.logger.Errorf("failed to list plugin images: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	resp := make([]PluginImageResponse, len(images))
	for i, img := range images {
		resp[i] = toImageResponse(img, s.assetStorage.GetPublicURL(img.S3Path))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"images": resp})
}

func (s *Server) GetImageUploadURL(c echo.Context) error {
	ctx := c.Request().Context()
	pluginID := c.Param("id")
	address := c.Get("address").(string)

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  queries.PluginID(pluginID),
		PublicKey: address,
	})
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
	}
	if owner.Role == queries.PluginOwnerRoleViewer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "viewers cannot upload images"})
	}

	var req GetUploadURLRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request format"})
	}

	if req.Filename == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "filename is required"})
	}

	if !itypes.AllowedContentTypes[req.ContentType] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
	}

	imageType := itypes.PluginImageType(req.ImageType)
	if _, ok := itypes.ImageTypeConstraints[imageType]; !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid image type"})
	}

	filename := sanitizeFilename(req.Filename)

	imageID := uuid.New()
	ext := contentTypeToExt(req.ContentType)
	s3Key := fmt.Sprintf("plugins/%s/%s/%s%s", pluginID, req.ImageType, imageID.String(), ext)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Errorf("failed to begin transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	defer tx.Rollback(ctx)

	err = s.db.LockPluginForUpdate(ctx, tx, types.PluginID(pluginID))
	if err != nil {
		s.logger.Errorf("failed to lock plugin: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	if imageType == itypes.PluginImageTypeMedia {
		count, err := s.db.CountVisibleMediaImages(ctx, tx, types.PluginID(pluginID))
		if err != nil {
			s.logger.Errorf("failed to count media images: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		if count >= s.cfg.MaxMediaImagesPerPlugin {
			return c.JSON(http.StatusConflict, map[string]string{"error": "maximum media images limit exceeded"})
		}
	}

	nextOrder := 0
	if imageType == itypes.PluginImageTypeMedia {
		nextOrder, err = s.db.GetNextMediaOrderTx(ctx, tx, types.PluginID(pluginID))
		if err != nil {
			s.logger.Errorf("failed to get next media order: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
	}

	_, err = s.db.CreatePendingPluginImage(ctx, tx, itypes.PluginImageCreateParams{
		ID:                  imageID,
		PluginID:            types.PluginID(pluginID),
		ImageType:           imageType,
		S3Path:              s3Key,
		ImageOrder:          nextOrder,
		UploadedByPublicKey: address,
		ContentType:         req.ContentType,
		Filename:            filename,
		Visible:             false,
	})
	if err != nil {
		s.logger.Errorf("failed to create pending image: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Errorf("failed to commit transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	expiry := s.cfg.PresignedURLExpiry
	if expiry == 0 {
		expiry = 15 * time.Minute
	}

	uploadURL, err := s.assetStorage.PresignPut(ctx, s3Key, req.ContentType, expiry)
	if err != nil {
		s.logger.Errorf("failed to presign put: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	expiresAt := time.Now().Add(expiry).UTC().Format(time.RFC3339)

	return c.JSON(http.StatusOK, GetUploadURLResponse{
		UploadURL: uploadURL,
		ImageID:   imageID.String(),
		S3Path:    s3Key,
		ExpiresAt: expiresAt,
	})
}

func (s *Server) ConfirmImageUpload(c echo.Context) error {
	ctx := c.Request().Context()
	pluginID := c.Param("id")
	imageIDStr := c.Param("imageId")
	address := c.Get("address").(string)

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  queries.PluginID(pluginID),
		PublicKey: address,
	})
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
	}
	if owner.Role == queries.PluginOwnerRoleViewer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "viewers cannot confirm uploads"})
	}

	imageID, err := uuid.Parse(imageIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid image id"})
	}

	img, err := s.db.GetPluginImageByID(ctx, types.PluginID(pluginID), imageID)
	if err != nil {
		s.logger.Errorf("failed to get plugin image: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if img == nil || img.Visible || img.Deleted {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "pending image not found"})
	}

	meta, err := s.assetStorage.HeadObject(ctx, img.S3Path)
	if err != nil {
		s.logger.Errorf("failed to head object: %v", err)
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file not found in storage"})
	}
	if meta == nil {
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file not found in storage"})
	}

	if meta.ContentType == "" || !itypes.AllowedContentTypes[meta.ContentType] || meta.ContentType != img.ContentType {
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
	}

	maxSize := s.cfg.MaxImageSizeBytes
	if maxSize == 0 {
		maxSize = 5 * 1024 * 1024
	}
	if meta.ContentLength > maxSize {
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "image file too large"})
	}

	rangeEnd := int64(65535)
	data, err := s.assetStorage.GetObjectRange(ctx, img.S3Path, 0, rangeEnd)
	if err != nil {
		s.logger.Errorf("failed to get object range: %v", err)
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse image dimensions"})
	}

	detectedType := DetectContentType(data)
	if detectedType != img.ContentType {
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file content does not match declared content type"})
	}

	width, height, err := ParseImageDimensions(data, detectedType)
	if err != nil && detectedType == "image/jpeg" {
		rangeEnd = int64(262143)
		data, err = s.assetStorage.GetObjectRange(ctx, img.S3Path, 0, rangeEnd)
		if err == nil {
			width, height, err = ParseImageDimensions(data, detectedType)
		}
	}
	if err != nil {
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse image dimensions"})
	}

	constraints := itypes.ImageTypeConstraints[img.ImageType]
	if width > constraints.MaxWidth || height > constraints.MaxHeight {
		s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image exceeds maximum dimensions of %dx%d", constraints.MaxWidth, constraints.MaxHeight)})
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Errorf("failed to begin transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	defer tx.Rollback(ctx)

	if img.ImageType == itypes.PluginImageTypeMedia {
		err = s.db.LockPluginForUpdate(ctx, tx, types.PluginID(pluginID))
		if err != nil {
			s.logger.Errorf("failed to lock plugin: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}

		count, err := s.db.CountVisibleMediaImages(ctx, tx, types.PluginID(pluginID))
		if err != nil {
			s.logger.Errorf("failed to count media images: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		if count >= s.cfg.MaxMediaImagesPerPlugin {
			s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
			return c.JSON(http.StatusConflict, map[string]string{"error": "maximum media images limit exceeded"})
		}
	}

	err = s.db.ConfirmPluginImage(ctx, tx, types.PluginID(pluginID), imageID)
	if err != nil {
		s.logger.Errorf("failed to confirm plugin image: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Errorf("failed to commit transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	img, _ = s.db.GetPluginImageByID(ctx, types.PluginID(pluginID), imageID)
	if img == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, toImageResponse(*img, s.assetStorage.GetPublicURL(img.S3Path)))
}

func (s *Server) UpdatePluginImage(c echo.Context) error {
	ctx := c.Request().Context()
	pluginID := c.Param("id")
	imageIDStr := c.Param("imageId")
	address := c.Get("address").(string)

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  queries.PluginID(pluginID),
		PublicKey: address,
	})
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
	}
	if owner.Role == queries.PluginOwnerRoleViewer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "viewers cannot update images"})
	}

	imageID, err := uuid.Parse(imageIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid image id"})
	}

	var req UpdateImageRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request format"})
	}

	if req.Visible == nil && req.ImageOrder == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no fields to update"})
	}

	if req.ImageOrder != nil {
		img, err := s.db.GetPluginImageByID(ctx, types.PluginID(pluginID), imageID)
		if err != nil || img == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "image not found"})
		}
		if img.ImageType != itypes.PluginImageTypeMedia {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "image order only valid for media type"})
		}
		if *req.ImageOrder < 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "image order cannot be negative"})
		}
	}

	updated, err := s.db.UpdatePluginImage(ctx, types.PluginID(pluginID), imageID, req.Visible, req.ImageOrder)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "image not found"})
		}
		s.logger.Errorf("failed to update plugin image: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, toImageResponse(*updated, s.assetStorage.GetPublicURL(updated.S3Path)))
}

func (s *Server) DeletePluginImage(c echo.Context) error {
	ctx := c.Request().Context()
	pluginID := c.Param("id")
	imageIDStr := c.Param("imageId")
	address := c.Get("address").(string)

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  queries.PluginID(pluginID),
		PublicKey: address,
	})
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
	}
	if owner.Role == queries.PluginOwnerRoleViewer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "viewers cannot delete images"})
	}

	imageID, err := uuid.Parse(imageIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid image id"})
	}

	_, err = s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "image not found"})
		}
		s.logger.Errorf("failed to delete plugin image: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "image deleted"})
}

func (s *Server) ReorderPluginImages(c echo.Context) error {
	ctx := c.Request().Context()
	pluginID := c.Param("id")
	address := c.Get("address").(string)

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  queries.PluginID(pluginID),
		PublicKey: address,
	})
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
	}
	if owner.Role == queries.PluginOwnerRoleViewer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "viewers cannot reorder images"})
	}

	var req ReorderImagesRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request format"})
	}

	if len(req.ImageIDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "image ids required"})
	}

	imageIDs := make([]uuid.UUID, len(req.ImageIDs))
	for i, idStr := range req.ImageIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid image id format"})
		}
		imageIDs[i] = id
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Errorf("failed to begin transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	defer tx.Rollback(ctx)

	err = s.db.ReorderMediaImages(ctx, tx, types.PluginID(pluginID), imageIDs)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		s.logger.Errorf("failed to reorder media images: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Errorf("failed to commit transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "images reordered"})
}

func toImageResponse(img itypes.PluginImageRecord, url string) PluginImageResponse {
	return PluginImageResponse{
		ImageID:     img.ID.String(),
		PluginID:    string(img.PluginID),
		ImageType:   string(img.ImageType),
		URL:         url,
		S3Path:      img.S3Path,
		ImageOrder:  img.ImageOrder,
		ContentType: img.ContentType,
		Filename:    img.Filename,
		Visible:     img.Visible,
		Deleted:     img.Deleted,
		CreatedAt:   img.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   img.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func sanitizeFilename(filename string) string {
	normalized := strings.ReplaceAll(filename, "\\", "/")
	base := path.Base(normalized)

	var clean strings.Builder
	for _, r := range base {
		if r < 32 || r == 127 {
			continue
		}
		clean.WriteRune(r)
	}
	result := clean.String()

	if len(result) > 255 {
		result = result[:255]
	}

	if result == "" || result == "." || result == ".." {
		result = "file"
	}

	return result
}

func contentTypeToExt(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
