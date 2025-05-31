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
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
)

type Plugin interface {
	GetPluginWithRating(ctx context.Context, pluginId string) (*types.Plugin, error)
	CreatePluginReviewWithRating(ctx context.Context, reviewDto types.ReviewCreateDto, pluginId string) (*types.ReviewDto, error)
	GetPluginRecipeSpecification(ctx context.Context, pluginID string) (interface{}, error)
}

type PluginServiceStorage interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error

	FindRatingByPluginId(ctx context.Context, dbTx pgx.Tx, pluginId string) ([]types.PluginRatingDto, error)
	CreateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string) error
	UpdateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, reviewRating int) error

	CreateReview(ctx context.Context, dbTx pgx.Tx, reviewDto types.ReviewCreateDto, pluginId string) (string, error)
	FindReviews(ctx context.Context, pluginId string, take int, skip int, sort string) (types.ReviewsDto, error)
	FindReviewById(ctx context.Context, db pgx.Tx, id string) (*types.ReviewDto, error)

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

func (s *PluginService) GetPluginWithRating(ctx context.Context, pluginId string) (*types.Plugin, error) {
	var plugin *types.Plugin
	err := s.db.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error

		// Find plugin
		plugin, err = s.db.FindPluginById(ctx, tx, ptypes.PluginID(pluginId))
		if err != nil {
			return fmt.Errorf("failed to get plugin: %w", err)
		}

		// Find rating
		// TODO: restore ratings with a custom type
		// var rating []types.PluginRatingDto
		// rating, err = s.db.FindRatingByPluginId(ctx, tx, pluginId)
		// if err != nil {
		// 	return fmt.Errorf("failed to get rating: %w", err)
		// }

		return nil
	})
	if err != nil {
		return nil, err
	}
	return plugin, nil
}

func (s *PluginService) CreatePluginReviewWithRating(ctx context.Context, reviewDto types.ReviewCreateDto, pluginId string) (*types.ReviewDto, error) {
	var review *types.ReviewDto
	err := s.db.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		// Insert review
		reviewId, err := s.db.CreateReview(ctx, tx, reviewDto, pluginId)
		if err != nil {
			return fmt.Errorf("failed to create review: %w", err)
		}

		// Update rating
		err = s.db.UpdateRatingForPlugin(ctx, tx, pluginId, reviewDto.Rating)
		if err != nil {
			return fmt.Errorf("failed to update rating: %w", err)
		}

		// Find review
		review, err = s.db.FindReviewById(ctx, tx, reviewId)
		if err != nil {
			return fmt.Errorf("failed to get review: %w", err)
		}

		// Find rating
		rating, err := s.db.FindRatingByPluginId(ctx, tx, pluginId)
		if err != nil {
			return fmt.Errorf("failed to get rating: %w", err)
		}

		review.Ratings = rating

		return nil
	})
	if err != nil {
		return nil, err
	}
	return review, nil
}

// GetPluginRecipeSpecification fetches recipe specification from plugin server with caching
func (s *PluginService) GetPluginRecipeSpecification(ctx context.Context, pluginID string) (interface{}, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("recipe_spec:%s", pluginID)

	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var cachedSpec interface{}
			if err := json.Unmarshal([]byte(cached), &cachedSpec); err == nil {
				fmt.Printf("[GetPluginRecipeSpecification] Cache hit for plugin %s\n", pluginID)
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
		fmt.Printf("[GetPluginRecipeSpecification] Cached recipe spec for plugin %s\n", pluginID)
	}

	return recipeSpec, nil
}

// Helper method to call plugin server
func (s *PluginService) fetchRecipeSpecificationFromPlugin(ctx context.Context, serverEndpoint string) (interface{}, error) {
	url := fmt.Sprintf("%s/recipe-specification", strings.TrimSuffix(serverEndpoint, "/"))

	fmt.Printf("[fetchRecipeSpecificationFromPlugin] Calling plugin endpoint: %s\n", url)

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

	var recipeSpec interface{}
	if err := json.NewDecoder(resp.Body).Decode(&recipeSpec); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return recipeSpec, nil
}
