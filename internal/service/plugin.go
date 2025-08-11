package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/plugin/libhttp"

	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
)

type Plugin interface {
	GetPluginWithRating(ctx context.Context, pluginId string) (*types.PluginWithRatings, error)
	CreatePluginReviewWithRating(ctx context.Context, reviewDto types.ReviewCreateDto, pluginId string) (*types.ReviewDto, error)
	GetPluginRecipeSpecification(ctx context.Context, pluginID string) (*rtypes.RecipeSchema, error)
	GetPluginRecipeSpecificationSuggest(
		ctx context.Context,
		pluginID string,
		configuration map[string]any,
	) (*rtypes.PolicySuggest, error)
}

type PluginServiceStorage interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error

	FindRatingByPluginId(ctx context.Context, dbTx pgx.Tx, pluginId string) ([]types.PluginRatingDto, error)
	CreateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string) error
	UpdateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, reviewRating int) error
	ChangeRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, oldRating int, newRating int) error

	CreateReview(ctx context.Context, dbTx pgx.Tx, reviewDto types.ReviewCreateDto, pluginId string) (string, error)
	UpdateReview(ctx context.Context, dbTx pgx.Tx, reviewId string, reviewDto types.ReviewCreateDto) error
	FindReviews(ctx context.Context, pluginId string, take int, skip int, sort string) (types.ReviewsDto, error)
	FindReviewById(ctx context.Context, db pgx.Tx, id string) (*types.ReviewDto, error)
	FindReviewByUserAndPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, userAddress string) (*types.ReviewDto, error)

	FindPluginById(ctx context.Context, dbTx pgx.Tx, id ptypes.PluginID) (*types.Plugin, error)
}

type PluginService struct {
	db     PluginServiceStorage
	redis  *storage.RedisStorage
	logger *logrus.Logger
}

func NewPluginService(db PluginServiceStorage, redis *storage.RedisStorage, logger *logrus.Logger) (*PluginService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	return &PluginService{
		db:     db,
		redis:  redis,
		logger: logger,
	}, nil
}

func (s *PluginService) GetPluginWithRating(ctx context.Context, pluginId string) (*types.PluginWithRatings, error) {
	var pluginWithRatings *types.PluginWithRatings
	err := s.db.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error

		// Find plugin
		plugin, err := s.db.FindPluginById(ctx, tx, ptypes.PluginID(pluginId))
		if err != nil {
			return fmt.Errorf("failed to get plugin: %w", err)
		}

		// Find rating
		var rating []types.PluginRatingDto
		rating, err = s.db.FindRatingByPluginId(ctx, tx, pluginId)
		if err != nil {
			return fmt.Errorf("failed to get rating: %w", err)
		}

		// Create response with ratings
		pluginWithRatings = &types.PluginWithRatings{
			Plugin:  *plugin,
			Ratings: rating,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return pluginWithRatings, nil
}

func (s *PluginService) CreatePluginReviewWithRating(ctx context.Context, reviewDto types.ReviewCreateDto, pluginId string) (*types.ReviewDto, error) {
	// Normalize address to lowercase to prevent case-sensitive duplicates
	reviewDto.Address = strings.ToLower(reviewDto.Address)

	var review *types.ReviewDto
	err := s.db.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		var reviewId string

		// Check if user already has a review for this plugin
		existingReview, err := s.db.FindReviewByUserAndPlugin(ctx, tx, pluginId, reviewDto.Address)
		if err != nil {
			s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: FindReviewByUserAndPlugin failed for plugin %s", pluginId)
			return fmt.Errorf("failed to check existing review: %w", err)
		}

		if existingReview != nil {
			// Update existing review
			s.logger.Infof("PluginService.CreatePluginReviewWithRating: Updating existing review %s for plugin %s", existingReview.ID, pluginId)
			err = s.db.UpdateReview(ctx, tx, existingReview.ID, reviewDto)
			if err != nil {
				s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: UpdateReview failed for plugin %s", pluginId)
				return fmt.Errorf("failed to update review: %w", err)
			}
			reviewId = existingReview.ID

			// Update rating counts (change from old rating to new rating)
			if existingReview.Rating != reviewDto.Rating {
				err = s.db.ChangeRatingForPlugin(ctx, tx, pluginId, existingReview.Rating, reviewDto.Rating)
				if err != nil {
					s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: ChangeRatingForPlugin failed for plugin %s", pluginId)
					return fmt.Errorf("failed to change rating: %w", err)
				}
			}
		} else {
			// Create new review
			s.logger.Infof("PluginService.CreatePluginReviewWithRating: Creating new review for plugin %s", pluginId)
			reviewId, err = s.db.CreateReview(ctx, tx, reviewDto, pluginId)
			if err != nil {
				s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: CreateReview failed for plugin %s", pluginId)
				return fmt.Errorf("failed to create review: %w", err)
			}

			// Add new rating
			err = s.db.UpdateRatingForPlugin(ctx, tx, pluginId, reviewDto.Rating)
			if err != nil {
				s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: UpdateRatingForPlugin failed for plugin %s", pluginId)
				return fmt.Errorf("failed to update rating: %w", err)
			}
		}

		// Find the final review
		review, err = s.db.FindReviewById(ctx, tx, reviewId)
		if err != nil {
			s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: FindReviewById failed for review %s", reviewId)
			return fmt.Errorf("failed to get review: %w", err)
		}

		// Find updated ratings
		rating, err := s.db.FindRatingByPluginId(ctx, tx, pluginId)
		if err != nil {
			s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: FindRatingByPluginId failed for plugin %s", pluginId)
			return fmt.Errorf("failed to get rating: %w", err)
		}

		review.Ratings = rating
		return nil
	})
	if err != nil {
		s.logger.WithError(err).Errorf("PluginService.CreatePluginReviewWithRating: Transaction failed for plugin %s", pluginId)
		return nil, err
	}

	return review, nil
}

// GetPluginRecipeSpecification fetches recipe specification from plugin server with caching
func (s *PluginService) GetPluginRecipeSpecification(ctx context.Context, pluginID string) (*rtypes.RecipeSchema, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("recipe_spec:%s", pluginID)

	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			cachedSpec := &rtypes.RecipeSchema{}
			if err := json.Unmarshal([]byte(cached), cachedSpec); err == nil {
				s.logger.Debugf("[GetPluginRecipeSpecification] Cache hit for plugin %s\n", pluginID)
				return cachedSpec, nil
			}
		}
	}

	// Get plugin from database to get server endpoint
	plugin, err := s.db.FindPluginById(ctx, nil, ptypes.PluginID(pluginID))
	if err != nil {
		return nil, fmt.Errorf("failed to find plugin: %w", err)
	}

	// Call plugin server endpoint
	recipeSpec, err := s.fetchRecipeSpecificationFromPlugin(ctx, plugin.ServerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe specification from plugin: %w", err)
	}

	// Cache the result for 2 hours
	if s.redis != nil {
		specBytes, _ := json.Marshal(recipeSpec)
		_ = s.redis.Set(ctx, cacheKey, string(specBytes), 2*time.Hour)
		if err := s.redis.Set(ctx, cacheKey, string(specBytes), 2*time.Hour); err != nil {
			s.logger.WithError(err).Warnf("Failed to cache recipe spec for plugin %s", pluginID)
		}
		s.logger.Debugf("[GetPluginRecipeSpecification] Cached recipe spec for plugin %s\n", pluginID)
	}

	return recipeSpec, nil
}

// GetPluginRecipeSpecificationSuggest fetches recipe suggest from plugin server with caching
func (s *PluginService) GetPluginRecipeSpecificationSuggest(
	ctx context.Context,
	pluginID string,
	configuration map[string]any,
) (*rtypes.PolicySuggest, error) {
	plugin, err := s.db.FindPluginById(ctx, nil, ptypes.PluginID(pluginID))
	if err != nil {
		return nil, fmt.Errorf("failed to find plugin: %w", err)
	}

	type req struct {
		Configuration map[string]any `json:"configuration"`
	}

	recipeSpec, err := libhttp.Call[*rtypes.PolicySuggest](
		ctx,
		http.MethodPost,
		plugin.ServerEndpoint+"/plugin/recipe-specification/suggest",
		map[string]string{
			"Content-Type": "application/json",
		},
		req{
			Configuration: configuration,
		},
		map[string]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe spec: %w", err)
	}

	return recipeSpec, nil
}

// Helper method to call plugin server
func (s *PluginService) fetchRecipeSpecificationFromPlugin(ctx context.Context, serverEndpoint string) (*rtypes.RecipeSchema, error) {
	url := fmt.Sprintf("%s/plugin/recipe-specification", strings.TrimSuffix(serverEndpoint, "/"))

	s.logger.Debugf("[fetchRecipeSpecificationFromPlugin] Calling plugin endpoint: %s\n", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call plugin endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin endpoint returned status %d", resp.StatusCode)
	}

	recipeSpec := &rtypes.RecipeSchema{}
	if err := json.NewDecoder(resp.Body).Decode(recipeSpec); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return recipeSpec, nil
}
