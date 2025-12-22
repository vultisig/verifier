package types

type PluginCategory string

const (
	PluginCategoryAIAgent PluginCategory = "ai-agent"
	PluginCategoryApp     PluginCategory = "app"
)

// IsValid checks if the plugin category is valid
func (pc PluginCategory) IsValid() bool {
	switch pc {
	case PluginCategoryAIAgent, PluginCategoryApp:
		return true
	}
	return false
}

// String returns the string representation of the plugin category
func (pc PluginCategory) String() string {
	switch pc {
	case PluginCategoryAIAgent:
		return "AI Agent"
	case PluginCategoryApp:
		return "App"
	}
	return ""
}
