package config

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

type YAMLPricing struct {
	ID        string  `yaml:"id"`
	Type      string  `yaml:"type"`
	Frequency *string `yaml:"frequency,omitempty"`
	Amount    uint64  `yaml:"amount"`
	Asset     string  `yaml:"asset"`
	Metric    string  `yaml:"metric"`
	PluginID  string  `yaml:"plugin_id"`
}

type YAMLPlugin struct {
	ID             string         `yaml:"id"`
	Title          string         `yaml:"title"`
	Description    string         `yaml:"description"`
	ServerEndpoint string         `yaml:"server_endpoint"`
	Category       string         `yaml:"category"`
	Pricing        []YAMLPricing  `yaml:"pricing"`
}

type YAMLData struct {
	Plugins []YAMLPlugin `yaml:"plugins"`
}

type PluginData struct {
	plugins map[types.PluginID]itypes.Plugin
}

func LoadPluginData(filePath string) (*PluginData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var yamlData YAMLData
	err = yaml.Unmarshal(data, &yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	pluginData := &PluginData{
		plugins: make(map[types.PluginID]itypes.Plugin),
	}

	now := time.Now()
	for _, yp := range yamlData.Plugins {
		plugin := itypes.Plugin{
			ID:             types.PluginID(yp.ID),
			Title:          yp.Title,
			Description:    yp.Description,
			ServerEndpoint: yp.ServerEndpoint,
			Category:       itypes.PluginCategory(yp.Category),
			CreatedAt:      now,
			UpdatedAt:      now,
			Pricing:        make([]types.Pricing, 0, len(yp.Pricing)),
		}

		for _, ypr := range yp.Pricing {
			pricingID, err := uuid.Parse(ypr.ID)
			if err != nil {
				return nil, fmt.Errorf("invalid pricing ID %s: %w", ypr.ID, err)
			}

			var frequency *types.PricingFrequency
			if ypr.Frequency != nil {
				freq := types.PricingFrequency(*ypr.Frequency)
				frequency = &freq
			}

			pricing := types.Pricing{
				ID:        pricingID,
				Type:      types.PricingType(ypr.Type),
				Frequency: frequency,
				Amount:    ypr.Amount,
				Asset:     types.PricingAsset(ypr.Asset),
				Metric:    types.PricingMetric(ypr.Metric),
				PluginID:  types.PluginID(ypr.PluginID),
				CreatedAt: now,
				UpdatedAt: now,
			}
			plugin.Pricing = append(plugin.Pricing, pricing)
		}

		pluginData.plugins[plugin.ID] = plugin
	}

	return pluginData, nil
}

func (pd *PluginData) FindPluginById(id types.PluginID) (*itypes.Plugin, error) {
	plugin, exists := pd.plugins[id]
	if !exists {
		return nil, fmt.Errorf("plugin not found")
	}
	return &plugin, nil
}

func (pd *PluginData) FindPlugins(filters itypes.PluginFilters, take int, skip int, sort string) (*itypes.PluginsPaginatedList, error) {
	allPlugins := make([]itypes.Plugin, 0, len(pd.plugins))

	for _, plugin := range pd.plugins {
		allPlugins = append(allPlugins, plugin)
	}

	filtered := make([]itypes.Plugin, 0, len(allPlugins))
	for _, plugin := range allPlugins {
		if filters.Term != nil {
			if !containsIgnoreCase(plugin.Title, *filters.Term) && !containsIgnoreCase(plugin.Description, *filters.Term) {
				continue
			}
		}

		if filters.CategoryID != nil && string(plugin.Category) != *filters.CategoryID {
			continue
		}

		filtered = append(filtered, plugin)
	}

	totalCount := len(filtered)

	if skip >= len(filtered) {
		return &itypes.PluginsPaginatedList{
			Plugins:    []itypes.Plugin{},
			TotalCount: totalCount,
		}, nil
	}

	end := skip + take
	if end > len(filtered) {
		end = len(filtered)
	}

	result := filtered[skip:end]

	return &itypes.PluginsPaginatedList{
		Plugins:    result,
		TotalCount: totalCount,
	}, nil
}

func containsIgnoreCase(s, substr string) bool {
	sLower := []rune(s)
	substrLower := []rune(substr)

	for i := range sLower {
		if sLower[i] >= 'A' && sLower[i] <= 'Z' {
			sLower[i] = sLower[i] + 32
		}
	}

	for i := range substrLower {
		if substrLower[i] >= 'A' && substrLower[i] <= 'Z' {
			substrLower[i] = substrLower[i] + 32
		}
	}

	return contains(string(sLower), string(substrLower))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
