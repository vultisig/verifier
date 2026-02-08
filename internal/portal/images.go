package portal

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/storage/postgres/queries"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

const (
	defaultMaxImageBytes  = 5 * 1024 * 1024
	defaultMaxMediaImages = 10
	maxFilenameLength     = 255
	asciiSpace            = 32
	asciiDEL              = 127
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
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "access denied"})
	}

	includeHidden := owner.Role != queries.PluginOwnerRoleViewer
	includeDeleted := false
	if c.QueryParam("include_deleted") == "true" {
		if owner.Role != queries.PluginOwnerRoleAdmin {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can view deleted images"})
		}
		includeDeleted = true
	}

	var imageTypeFilter *itypes.PluginImageType
	if it := c.QueryParam("image_type"); it != "" {
		t := itypes.PluginImageType(it)
		if _, ok := itypes.ImageTypeConstraints[t]; !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid image type"})
		}
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
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
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
	if ext == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
	}
	s3Key := fmt.Sprintf("plugins/%s/%s/%s%s", pluginID, string(imageType), imageID.String(), ext)

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
		maxMedia := s.cfg.MaxMediaImagesPerPlugin
		if maxMedia == 0 {
			maxMedia = defaultMaxMediaImages
		}
		count, err := s.db.CountVisibleMediaImages(ctx, tx, types.PluginID(pluginID))
		if err != nil {
			s.logger.Errorf("failed to count media images: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		if count >= maxMedia {
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

	expiry := s.cfg.PresignedURLExpiry
	if expiry == 0 {
		expiry = 15 * time.Minute
	}

	uploadURL, err := s.assetStorage.PresignPut(ctx, s3Key, req.ContentType, expiry)
	if err != nil {
		s.logger.Errorf("failed to presign put: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
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
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
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

	markPendingDeleted := func() {
		if _, delErr := s.db.SoftDeletePluginImage(ctx, types.PluginID(pluginID), imageID); delErr != nil {
			s.logger.Warnf("failed to soft-delete pending image %s: %v", imageID, delErr)
		}
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
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "storage temporarily unavailable, please retry"})
	}
	if meta == nil {
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file not found in storage"})
	}

	if meta.ContentType == "" || !itypes.AllowedContentTypes[meta.ContentType] || meta.ContentType != img.ContentType {
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid content type"})
	}

	maxSize := s.cfg.MaxImageSizeBytes
	if maxSize == 0 {
		maxSize = defaultMaxImageBytes
	}
	if meta.ContentLength > maxSize {
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "image file too large"})
	}

	reader, err := s.assetStorage.GetObject(ctx, img.S3Path)
	if err != nil {
		s.logger.Errorf("failed to get object: %v", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "storage temporarily unavailable, please retry"})
	}
	defer reader.Close()

	limitedReader := io.LimitReader(reader, maxSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		s.logger.Errorf("failed to read object: %v", err)
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "storage temporarily unavailable, please retry"})
	}

	if int64(len(data)) > maxSize {
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "image file too large"})
	}

	info, parseErr := ParseImageInfo(data)
	if parseErr != nil {
		s.logger.Warnf("failed to parse image info for %s: %v", imageID, parseErr)
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse image"})
	}

	if info.MIME != img.ContentType {
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file content does not match declared content type"})
	}

	constraints, ok := itypes.ImageTypeConstraints[img.ImageType]
	if !ok {
		s.logger.Errorf("unknown image type %q for image %s", img.ImageType, imageID)
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid image type"})
	}
	if info.Width > constraints.MaxWidth || info.Height > constraints.MaxHeight {
		markPendingDeleted()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image exceeds maximum dimensions of %dx%d", constraints.MaxWidth, constraints.MaxHeight)})
	}

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

	if img.ImageType == itypes.PluginImageTypeMedia {
		maxMedia := s.cfg.MaxMediaImagesPerPlugin
		if maxMedia == 0 {
			maxMedia = defaultMaxMediaImages
		}
		count, err := s.db.CountVisibleMediaImages(ctx, tx, types.PluginID(pluginID))
		if err != nil {
			s.logger.Errorf("failed to count media images: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		if count >= maxMedia {
			if _, err := s.db.SoftDeletePluginImageTx(ctx, tx, types.PluginID(pluginID), imageID); err != nil {
				s.logger.Warnf("failed to soft-delete pending image %s: %v", imageID, err)
			}
			return c.JSON(http.StatusConflict, map[string]string{"error": "maximum media images limit exceeded"})
		}
	}

	confirmedImg, err := s.db.ConfirmPluginImage(ctx, tx, types.PluginID(pluginID), imageID)
	if err != nil {
		s.logger.Errorf("failed to confirm plugin image: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Errorf("failed to commit transaction: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, toImageResponse(*confirmedImg, s.assetStorage.GetPublicURL(confirmedImg.S3Path)))
}

func (s *Server) UpdatePluginImage(c echo.Context) error {
	ctx := c.Request().Context()
	pluginID := c.Param("id")
	imageIDStr := c.Param("imageId")
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
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
		if err != nil {
			s.logger.Errorf("failed to get plugin image: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		}
		if img == nil {
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
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
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
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	owner, err := s.queries.GetPluginOwnerWithRole(ctx, &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
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

	err = s.db.LockPluginForUpdate(ctx, tx, types.PluginID(pluginID))
	if err != nil {
		s.logger.Errorf("failed to lock plugin: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	err = s.db.ReorderMediaImages(ctx, tx, types.PluginID(pluginID), imageIDs)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "duplicate") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "not media") {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": errMsg})
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
		if r < asciiSpace || r == asciiDEL {
			continue
		}
		clean.WriteRune(r)
	}
	result := clean.String()

	if len(result) > maxFilenameLength {
		truncateAt := maxFilenameLength
		for truncateAt > 0 && !utf8.RuneStart(result[truncateAt]) {
			truncateAt--
		}
		result = result[:truncateAt]
	}

	result = strings.TrimSpace(result)
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
