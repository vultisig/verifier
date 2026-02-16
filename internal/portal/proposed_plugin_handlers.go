package portal

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	itypes "github.com/vultisig/verifier/internal/types"
)

const (
	maxProposedImageBytes    = 2 * 1024 * 1024
	maxActiveProposalsPerDev = 5
	maxMediaImagesProposal   = 7
	errPluginIDTaken         = "plugin_id is already taken"
)

var (
	proposalValidate     = validator.New()
	pluginIDRegex        = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	errResponseCommitted = errors.New("response already committed")
)

func sanitizeValidationError(err error) string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return "validation failed"
	}
	var msgs []string
	for _, fe := range ve {
		field := strings.ToLower(fe.Field())
		switch fe.Tag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("%s is required", field))
		case "email":
			msgs = append(msgs, fmt.Sprintf("%s must be a valid email", field))
		case "max":
			msgs = append(msgs, fmt.Sprintf("%s exceeds maximum length", field))
		case "min":
			msgs = append(msgs, fmt.Sprintf("%s is too short", field))
		case "oneof":
			msgs = append(msgs, fmt.Sprintf("%s has invalid value", field))
		default:
			msgs = append(msgs, fmt.Sprintf("%s is invalid", field))
		}
	}
	return strings.Join(msgs, "; ")
}

const (
	constraintProposedPluginPK       = "proposed_plugins_pkey"
	constraintProposedImageOneLogo   = "proposed_plugin_images_one_logo"
	constraintProposedImageOneBanner = "proposed_plugin_images_one_banner"
	constraintProposedImageOneThumb  = "proposed_plugin_images_one_thumbnail"
)

type dbConstraintViolation int

const (
	violationUnknown dbConstraintViolation = iota
	violationPluginIDTaken
	violationDuplicateLogo
	violationDuplicateBanner
	violationDuplicateThumbnail
)

func classifyProposalConstraintViolation(err error) dbConstraintViolation {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return violationUnknown
	}
	if pgErr.Code != pgerrcode.UniqueViolation {
		return violationUnknown
	}
	switch pgErr.ConstraintName {
	case constraintProposedPluginPK:
		return violationPluginIDTaken
	case constraintProposedImageOneLogo:
		return violationDuplicateLogo
	case constraintProposedImageOneBanner:
		return violationDuplicateBanner
	case constraintProposedImageOneThumb:
		return violationDuplicateThumbnail
	default:
		return violationUnknown
	}
}

func (s *Server) buildImageResponses(images []itypes.ProposedPluginImage) []ProposedPluginImageResponse {
	responses := make([]ProposedPluginImageResponse, len(images))
	for i, img := range images {
		responses[i] = ProposedPluginImageResponse{
			ID:          img.ID.String(),
			Type:        string(img.ImageType),
			URL:         s.assetStorage.GetPublicURL(img.S3Path),
			ContentType: img.ContentType,
			Filename:    img.Filename,
			ImageOrder:  img.ImageOrder,
		}
	}
	return responses
}

func buildProposedPluginResponse(p itypes.ProposedPlugin, images []ProposedPluginImageResponse) ProposedPluginResponse {
	var pricingModelStr *string
	if p.PricingModel != nil {
		pm := string(*p.PricingModel)
		pricingModelStr = &pm
	}
	return ProposedPluginResponse{
		PluginID:         p.PluginID,
		Title:            p.Title,
		ShortDescription: p.ShortDescription,
		ServerEndpoint:   p.ServerEndpoint,
		Category:         string(p.Category),
		SupportedChains:  p.SupportedChains,
		PricingModel:     pricingModelStr,
		ContactEmail:     p.ContactEmail,
		Notes:            p.Notes,
		Status:           string(p.Status),
		Images:           images,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

func (s *Server) requireApprover(c echo.Context) (string, error) {
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		if err := c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"}); err != nil {
			return "", err
		}
		return "", errResponseCommitted
	}

	ctx := c.Request().Context()
	isApprover, dbErr := s.queries.IsListingApprover(ctx, address)
	if dbErr != nil {
		s.logger.WithError(dbErr).Error("failed to check approver status")
		if err := c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"}); err != nil {
			return "", err
		}
		return "", errResponseCommitted
	}
	if !isApprover {
		if err := c.JSON(http.StatusForbidden, map[string]string{"error": "admin access required"}); err != nil {
			return "", err
		}
		return "", errResponseCommitted
	}
	return address, nil
}

func (s *Server) isPluginIDAvailable(ctx context.Context, pluginID string) (bool, error) {
	existsInPlugins, err := s.db.PluginIDExistsInPlugins(ctx, pluginID)
	if err != nil {
		return false, fmt.Errorf("check plugins: %w", err)
	}
	if existsInPlugins {
		return false, nil
	}

	existsInProposals, err := s.db.PluginIDExistsInProposals(ctx, pluginID)
	if err != nil {
		return false, fmt.Errorf("check proposals: %w", err)
	}
	return !existsInProposals, nil
}

func (s *Server) ValidatePluginID(c echo.Context) error {
	pluginID := strings.TrimSpace(strings.ToLower(c.Param("id")))
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	if !pluginIDRegex.MatchString(pluginID) {
		return c.JSON(http.StatusOK, ValidatePluginIDResponse{Available: false})
	}

	ctx := c.Request().Context()

	available, err := s.isPluginIDAvailable(ctx, pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to check plugin_id availability")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, ValidatePluginIDResponse{Available: available})
}

func (s *Server) CreatePluginProposal(c echo.Context) error {
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	ctx := c.Request().Context()

	activeCount, err := s.db.CountActiveProposalsByPublicKey(ctx, address)
	if err != nil {
		s.logger.WithError(err).Error("failed to count active proposals")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if activeCount >= maxActiveProposalsPerDev {
		return c.JSON(http.StatusConflict, map[string]string{"error": fmt.Sprintf("maximum of %d active proposals reached", maxActiveProposalsPerDev)})
	}

	var req CreateProposedPluginRequest
	err = c.Bind(&req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	req.PluginID = strings.TrimSpace(strings.ToLower(req.PluginID))
	req.Title = strings.TrimSpace(req.Title)
	req.ServerEndpoint = strings.TrimRight(strings.TrimSpace(req.ServerEndpoint), "/")
	req.ContactEmail = strings.TrimSpace(strings.ToLower(req.ContactEmail))
	req.ShortDescription = strings.TrimSpace(req.ShortDescription)
	req.Notes = strings.TrimSpace(req.Notes)
	req.PricingModel = strings.TrimSpace(req.PricingModel)

	err = proposalValidate.Struct(req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": sanitizeValidationError(err)})
	}

	if !pluginIDRegex.MatchString(req.PluginID) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin_id must contain only lowercase letters, numbers, and hyphens"})
	}

	u, err := url.ParseRequestURI(req.ServerEndpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint must be a valid http(s) URL"})
	}

	available, err := s.isPluginIDAvailable(ctx, req.PluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to check plugin_id availability")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if !available {
		return c.JSON(http.StatusConflict, map[string]string{"error": errPluginIDTaken})
	}

	logoCount := 0
	bannerCount := 0
	thumbnailCount := 0
	mediaCount := 0
	for _, img := range req.Images {
		switch img.Type {
		case "logo":
			logoCount++
		case "banner":
			bannerCount++
		case "thumbnail":
			thumbnailCount++
		case "media":
			mediaCount++
		}
	}

	if logoCount != 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "exactly 1 logo image is required"})
	}
	if bannerCount > 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at most 1 banner image allowed"})
	}
	if thumbnailCount > 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at most 1 thumbnail image allowed"})
	}
	if mediaCount > maxMediaImagesProposal {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("at most %d media images allowed", maxMediaImagesProposal)})
	}

	type decodedImage struct {
		Data        []byte
		ContentType string
		ImageType   string
		Filename    string
		Info        ImageInfo
	}
	decodedImages := make([]decodedImage, 0, len(req.Images))

	for i, img := range req.Images {
		estimatedSize := base64.StdEncoding.DecodedLen(len(img.Data))
		if estimatedSize > maxProposedImageBytes {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image %d exceeds maximum size of %dMB", i+1, maxProposedImageBytes/(1024*1024))})
		}

		data, err := base64.StdEncoding.DecodeString(img.Data)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image %d: invalid base64 data", i+1)})
		}

		if len(data) > maxProposedImageBytes {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image %d exceeds maximum size of %dMB", i+1, maxProposedImageBytes/(1024*1024))})
		}

		info, err := ParseImageInfo(data)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image %d: %v", i+1, err)})
		}

		if info.MIME != img.ContentType {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image %d: content_type does not match actual image format", i+1)})
		}

		imageType := itypes.PluginImageType(img.Type)
		constraints, ok := itypes.ImageTypeConstraints[imageType]
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image %d: invalid image type", i+1)})
		}

		if info.Width > constraints.MaxWidth || info.Height > constraints.MaxHeight {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("image %d: exceeds maximum dimensions of %dx%d", i+1, constraints.MaxWidth, constraints.MaxHeight)})
		}

		filename := sanitizeFilename(img.Filename)
		if filename == "" || filename == "file" {
			filename = fmt.Sprintf("%s%s", img.Type, contentTypeToExt(img.ContentType))
		}

		decodedImages = append(decodedImages, decodedImage{
			Data:        data,
			ContentType: img.ContentType,
			ImageType:   img.Type,
			Filename:    filename,
			Info:        info,
		})
	}

	type uploadedImage struct {
		S3Path      string
		ImageType   string
		ContentType string
		Filename    string
		ID          uuid.UUID
		Order       int
	}
	uploadedImages := make([]uploadedImage, 0, len(decodedImages))
	mediaOrder := 0

	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, uploaded := range uploadedImages {
			if err := s.assetStorage.Delete(cleanupCtx, uploaded.S3Path); err != nil {
				s.logger.WithError(err).Warnf("failed to delete S3 object %s during cleanup", uploaded.S3Path)
			}
		}
	}

	for _, img := range decodedImages {
		imageID := uuid.New()
		ext := contentTypeToExt(img.ContentType)
		s3Path := fmt.Sprintf("proposals/%s/%s/%s%s", req.PluginID, img.ImageType, imageID.String(), ext)

		order := 0
		if img.ImageType == "media" {
			order = mediaOrder
			mediaOrder++
		}

		uploadedImages = append(uploadedImages, uploadedImage{
			S3Path:      s3Path,
			ImageType:   img.ImageType,
			ContentType: img.ContentType,
			Filename:    img.Filename,
			ID:          imageID,
			Order:       order,
		})

		err := s.assetStorage.Upload(ctx, s3Path, img.Data, img.ContentType)
		if err != nil {
			s.logger.WithError(err).Error("failed to upload image to S3")
			cleanup()
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to upload images"})
		}
	}

	var pricingModel *itypes.ProposedPluginPricing
	if req.PricingModel != "" {
		pm := itypes.ProposedPluginPricing(req.PricingModel)
		pricingModel = &pm
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to begin transaction")
		cleanup()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	defer tx.Rollback(ctx)

	proposal, err := s.db.CreateProposedPlugin(ctx, tx, itypes.ProposedPluginCreateParams{
		PluginID:         req.PluginID,
		PublicKey:        address,
		Title:            req.Title,
		ShortDescription: req.ShortDescription,
		ServerEndpoint:   req.ServerEndpoint,
		Category:         itypes.PluginCategory(req.Category),
		SupportedChains:  req.SupportedChains,
		PricingModel:     pricingModel,
		ContactEmail:     req.ContactEmail,
		Notes:            req.Notes,
	})
	if err != nil {
		cleanup()
		violation := classifyProposalConstraintViolation(err)
		if violation == violationPluginIDTaken {
			s.logger.WithFields(logrus.Fields{"plugin_id": req.PluginID}).Info("plugin_id collision")
			return c.JSON(http.StatusConflict, map[string]string{"error": errPluginIDTaken})
		}
		s.logger.WithError(err).Error("failed to create proposed plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	imageResponses := make([]ProposedPluginImageResponse, 0, len(uploadedImages))
	for _, img := range uploadedImages {
		_, err := s.db.CreateProposedPluginImage(ctx, tx, itypes.ProposedPluginImageCreateParams{
			ID:                  img.ID,
			PluginID:            req.PluginID,
			ImageType:           itypes.PluginImageType(img.ImageType),
			S3Path:              img.S3Path,
			ImageOrder:          img.Order,
			UploadedByPublicKey: address,
			ContentType:         img.ContentType,
			Filename:            img.Filename,
		})
		if err != nil {
			cleanup()
			violation := classifyProposalConstraintViolation(err)
			switch violation {
			case violationDuplicateLogo:
				return c.JSON(http.StatusConflict, map[string]string{"error": "duplicate logo image"})
			case violationDuplicateBanner:
				return c.JSON(http.StatusConflict, map[string]string{"error": "duplicate banner image"})
			case violationDuplicateThumbnail:
				return c.JSON(http.StatusConflict, map[string]string{"error": "duplicate thumbnail image"})
			default:
				s.logger.WithError(err).Error("failed to create proposed plugin image")
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save image metadata"})
			}
		}

		imageResponses = append(imageResponses, ProposedPluginImageResponse{
			ID:          img.ID.String(),
			Type:        img.ImageType,
			URL:         s.assetStorage.GetPublicURL(img.S3Path),
			ContentType: img.ContentType,
			Filename:    img.Filename,
			ImageOrder:  img.Order,
		})
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to commit transaction")
		cleanup()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id": req.PluginID,
		"owner":     address,
		"images":    len(uploadedImages),
	}).Info("proposed plugin created with images")

	// TODO: migrate to Redis/Asynq queue for reliability
	s.emailService.SendProposalNotificationAsync(req.PluginID, req.Title, req.ContactEmail)

	return c.JSON(http.StatusCreated, buildProposedPluginResponse(*proposal, imageResponses))
}

func (s *Server) GetMyPluginProposal(c echo.Context) error {
	pluginID := strings.TrimSpace(strings.ToLower(c.Param("id")))
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	ctx := c.Request().Context()

	proposal, err := s.db.GetProposedPluginByOwner(ctx, address, pluginID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "proposed plugin not found"})
		}
		s.logger.WithError(err).Error("failed to get proposed plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	images, err := s.db.ListProposedPluginImages(ctx, pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to list proposed plugin images")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, buildProposedPluginResponse(*proposal, s.buildImageResponses(images)))
}

func (s *Server) GetMyPluginProposals(c echo.Context) error {
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	ctx := c.Request().Context()

	proposals, err := s.db.ListProposedPluginsByPublicKey(ctx, address)
	if err != nil {
		s.logger.WithError(err).Error("failed to list proposed plugins")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	responses := make([]ProposedPluginResponse, len(proposals))
	for i, proposal := range proposals {
		images, err := s.db.ListProposedPluginImages(ctx, proposal.PluginID)
		if err != nil {
			s.logger.WithError(err).Warnf("failed to list images for proposal %s", proposal.PluginID)
			images = []itypes.ProposedPluginImage{}
		}
		responses[i] = buildProposedPluginResponse(proposal, s.buildImageResponses(images))
	}

	return c.JSON(http.StatusOK, responses)
}

func (s *Server) GetAllPluginProposals(c echo.Context) error {
	_, err := s.requireApprover(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()

	proposals, err := s.db.ListAllProposedPlugins(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to list all proposed plugins")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	responses := make([]ProposedPluginResponse, len(proposals))
	for i, proposal := range proposals {
		images, err := s.db.ListProposedPluginImages(ctx, proposal.PluginID)
		if err != nil {
			s.logger.WithError(err).Warnf("failed to list images for proposal %s", proposal.PluginID)
			images = []itypes.ProposedPluginImage{}
		}
		responses[i] = buildProposedPluginResponse(proposal, s.buildImageResponses(images))
	}

	return c.JSON(http.StatusOK, responses)
}

func (s *Server) GetPluginProposal(c echo.Context) error {
	pluginID := strings.TrimSpace(strings.ToLower(c.Param("id")))
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}

	_, err := s.requireApprover(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()

	proposal, err := s.db.GetProposedPlugin(ctx, pluginID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "proposed plugin not found"})
		}
		s.logger.WithError(err).Error("failed to get proposed plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	images, err := s.db.ListProposedPluginImages(ctx, pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to list proposed plugin images")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, buildProposedPluginResponse(*proposal, s.buildImageResponses(images)))
}
