package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/plugin/libhttp"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/internal/util"
	ptypes "github.com/vultisig/verifier/types"
)

// PluginSkills represents the skills returned from a plugin's /skills endpoint.
type PluginSkills struct {
	PluginID string `json:"plugin_id"`
	SkillsMD string `json:"skills_md"`
}

type Plugin interface {
	GetPluginWithRating(ctx context.Context, pluginId string) (*types.PluginWithRatings, error)
	CreatePluginReviewWithRating(ctx context.Context, reviewDto types.ReviewCreateDto, pluginId string) (*types.ReviewDto, error)
	GetPluginRecipeSpecification(ctx context.Context, pluginID string) (*rtypes.RecipeSchema, error)
	GetPluginRecipeSpecificationSuggest(
		ctx context.Context,
		pluginID string,
		configuration map[string]any,
	) (*rtypes.PolicySuggest, error)
	GetPluginRecipeFunctions(ctx context.Context, pluginID string) (types.RecipeFunctions, error)
	GetPluginTitlesByIDs(ctx context.Context, ids []string) (map[string]string, error)
	GetPluginSkills(ctx context.Context, pluginID string) (*PluginSkills, error)
}

type PluginServiceStorage interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error

	FindRatingByPluginId(ctx context.Context, dbTx pgx.Tx, pluginId string) ([]types.PluginRatingDto, error)
	FindAvgRatingByPluginID(ctx context.Context, pluginId string) (types.PluginAvgRatingDto, error)
	CreateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string) error
	UpdateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, reviewRating int) error
	ChangeRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, oldRating int, newRating int) error

	CreateReview(ctx context.Context, dbTx pgx.Tx, reviewDto types.ReviewCreateDto, pluginId string) (string, error)
	UpdateReview(ctx context.Context, dbTx pgx.Tx, reviewId string, reviewDto types.ReviewCreateDto) error
	FindReviews(ctx context.Context, pluginId string, take int, skip int, sort string) (types.ReviewsDto, error)
	FindReviewById(ctx context.Context, db pgx.Tx, id string) (*types.ReviewDto, error)
	FindReviewByUserAndPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, userAddress string) (*types.ReviewDto, error)

	FindPluginById(ctx context.Context, dbTx pgx.Tx, id ptypes.PluginID) (*types.Plugin, error)
	GetAPIKeyByPluginId(ctx context.Context, pluginId string) (*types.APIKey, error)
	GetPluginTitlesByIDs(ctx context.Context, ids []string) (map[string]string, error)
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

func (s *PluginService) GetPluginTitlesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	return s.db.GetPluginTitlesByIDs(ctx, ids)
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
	keyInfo, err := s.db.GetAPIKeyByPluginId(ctx, pluginID)
	if err != nil || keyInfo == nil {
		return nil, fmt.Errorf("failed to find plugin server info: %w", err)
	}

	// Call plugin server endpoint
	recipeSpec, err := s.fetchRecipeSpecificationFromPlugin(ctx, plugin.ServerEndpoint, keyInfo.ApiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe specification from plugin: %w", err)
	}

	// Cache for 2 hours
	if s.redis != nil {
		specBytes, err := json.Marshal(recipeSpec)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to serialize recipe spec for caching")
			return recipeSpec, nil
		}
		err = s.redis.Set(ctx, cacheKey, string(specBytes), 2*time.Hour)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to cache recipe spec for plugin %s", pluginID)
			return recipeSpec, nil
		}
		s.logger.Debugf("[GetPluginRecipeSpecification] Cached recipe spec for plugin %s", pluginID)
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
	keyInfo, err := s.db.GetAPIKeyByPluginId(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to find plugin server info: %w", err)
	}

	type req struct {
		Configuration map[string]any `json:"configuration"`
	}

	policySuggestStr, err := libhttp.Call[string](
		ctx,
		http.MethodPost,
		plugin.ServerEndpoint+"/plugin/recipe-specification/suggest",
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + keyInfo.ApiKey,
		},
		req{
			Configuration: configuration,
		},
		map[string]string{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe spec: %w", err)
	}
	var policySuggest rtypes.PolicySuggest
	err = protojson.Unmarshal([]byte(policySuggestStr), &policySuggest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal recipe spec: %w", err)
	}
	return &policySuggest, nil
}

// GetPluginRecipeFunctions fetches recipe functions from plugin server with caching
func (s *PluginService) GetPluginRecipeFunctions(ctx context.Context, pluginID string) (types.RecipeFunctions, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("recipe_functions:%s", pluginID)

	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			cachedSpec := types.RecipeFunctions{}
			if err := json.Unmarshal([]byte(cached), &cachedSpec); err == nil {
				s.logger.Debugf("[GetPluginRecipeFunctions] Cache hit for plugin %s\n", pluginID)
				return cachedSpec, nil
			}
		}
	}

	// Get plugin from database to get server endpoint
	recipeSpec, err := s.GetPluginRecipeSpecification(ctx, pluginID)
	if err != nil {
		return types.RecipeFunctions{}, fmt.Errorf("failed to get recipe specification: %w", err)
	}

	uniqueFunctions := make(map[string]struct{})
	for _, resource := range recipeSpec.SupportedResources {
		f := resource.ResourcePath.FunctionId
		if f != "" {
			uniqueFunctions[f] = struct{}{}
		}
	}
	funcs := make([]string, 0, len(uniqueFunctions))
	for f := range uniqueFunctions {
		funcs = append(funcs, f)
	}
	sort.Strings(funcs)
	plugin, err := s.db.FindPluginById(ctx, nil, ptypes.PluginID(pluginID))
	if err != nil {
		return types.RecipeFunctions{}, fmt.Errorf("failed to find plugin: %w", err)
	}
	// if plugin is not free
	if len(plugin.Pricing) > 0 {
		funcs = append(funcs, "Fee deduction authorization")
	}
	// all plugins should have Vault balance visibility
	funcs = append(funcs, "Vault balance visibility")

	recipeFuncs := types.RecipeFunctions{
		ID:        pluginID,
		Functions: funcs,
	}
	// Cache for 2 hours
	if s.redis != nil {
		bytes, err := json.Marshal(recipeFuncs)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to serialize recipe functions for caching")
			return recipeFuncs, nil
		}
		err = s.redis.Set(ctx, cacheKey, string(bytes), 2*time.Hour)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to cache recipe functions for plugin %s", pluginID)
			return recipeFuncs, nil
		}
		s.logger.Debugf("[GetPluginRecipeFunctions] Cached recipe functions for plugin %s", pluginID)
	}

	return recipeFuncs, nil
}

// Helper method to call plugin server
func (s *PluginService) fetchRecipeSpecificationFromPlugin(ctx context.Context, serverEndpoint, token string) (*rtypes.RecipeSchema, error) {
	url := fmt.Sprintf("%s/plugin/recipe-specification", strings.TrimSuffix(serverEndpoint, "/"))

	s.logger.Debugf("[fetchRecipeSpecificationFromPlugin] Calling plugin endpoint: %s\n", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+token)

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
	if err := util.ValidateRecipeSchema(*recipeSpec); err != nil {
		return nil, fmt.Errorf("invalid recipe schema: %w", err)
	}

	return recipeSpec, nil
}

// GetPluginSkills fetches skills from plugin server with caching
func (s *PluginService) GetPluginSkills(ctx context.Context, pluginID string) (*PluginSkills, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("plugin_skills:%s", pluginID)

	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			cachedSkills := &PluginSkills{}
			if err := json.Unmarshal([]byte(cached), cachedSkills); err == nil {
				s.logger.Debugf("[GetPluginSkills] Cache hit for plugin %s", pluginID)
				return cachedSkills, nil
			}
		}
	}

	// Get plugin from database to get server endpoint
	plugin, err := s.db.FindPluginById(ctx, nil, ptypes.PluginID(pluginID))
	if err != nil {
		return nil, fmt.Errorf("failed to find plugin: %w", err)
	}
	keyInfo, err := s.db.GetAPIKeyByPluginId(ctx, pluginID)
	if err != nil || keyInfo == nil {
		return nil, fmt.Errorf("failed to find plugin server info: %w", err)
	}

	// Call plugin server endpoint
	skills, err := s.fetchSkillsFromPlugin(ctx, plugin.ServerEndpoint, keyInfo.ApiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills from plugin: %w", err)
	}

	// Cache for 2 hours
	if s.redis != nil {
		skillsBytes, err := json.Marshal(skills)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to serialize skills for caching")
			return skills, nil
		}
		err = s.redis.Set(ctx, cacheKey, string(skillsBytes), 2*time.Hour)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to cache skills for plugin %s", pluginID)
			return skills, nil
		}
		s.logger.Debugf("[GetPluginSkills] Cached skills for plugin %s", pluginID)
	}

	return skills, nil
}

func (s *PluginService) fetchSkillsFromPlugin(ctx context.Context, serverEndpoint, token string) (*PluginSkills, error) {
	url := fmt.Sprintf("%s/skills", strings.TrimSuffix(serverEndpoint, "/"))

	s.logger.Debugf("[fetchSkillsFromPlugin] Calling plugin endpoint: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call plugin endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plugin endpoint returned status %d", resp.StatusCode)
	}

	skills := &PluginSkills{}
	if err := json.NewDecoder(resp.Body).Decode(skills); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return skills, nil
}
