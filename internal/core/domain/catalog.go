package domain

// ModelFamily defines equivalent model IDs across different CLI tools and gateways.
type ModelFamily struct {
	CanonicalName string            `json:"canonicalName" yaml:"canonicalName"`
	Aliases       []string          `json:"aliases" yaml:"aliases"`
	ProviderIDs   map[string]string `json:"providerIds" yaml:"providerIds"` // provider -> internal model identifier
}

// BuiltinModelCatalog provides built-in equivalence mappings so users don't have to define them.
var BuiltinModelCatalog = []ModelFamily{
	{
		CanonicalName: "claude-3-7-sonnet",
		Aliases:       []string{"claude-sonnet-3.7", "claude-3.7-sonnet", "sonnet-3.7"},
		ProviderIDs: map[string]string{
			"claude":     "claude-3-7-sonnet",
			"opencode":   "anthropic/claude-3-7-sonnet",
			"openrouter": "anthropic/claude-3.7-sonnet",
			"bedrock":    "anthropic.claude-3-7-sonnet-v1:0",
		},
	},
	{
		CanonicalName: "claude-3-5-sonnet",
		Aliases:       []string{"claude-sonnet-3.5", "claude-3.5-sonnet", "sonnet-3.5"},
		ProviderIDs: map[string]string{
			"claude":     "claude-3-5-sonnet-20241022",
			"opencode":   "anthropic/claude-3-5-sonnet",
			"openrouter": "anthropic/claude-3.5-sonnet",
			"bedrock":    "anthropic.claude-3-5-sonnet-v2:0",
		},
	},
	{
		CanonicalName: "gpt-4o",
		Aliases:       []string{"chatgpt-4o", "gpt4o"},
		ProviderIDs: map[string]string{
			"codex":      "gpt-4o",
			"opencode":   "openai/gpt-4o",
			"copilot":    "gpt-4o-copilot",
			"openrouter": "openai/gpt-4o",
		},
	},
	{
		CanonicalName: "gemini-2.5-pro",
		Aliases:       []string{"gemini-pro", "gemini-2.5"},
		ProviderIDs: map[string]string{
			"gemini":     "gemini-2.5-pro",
			"agy":        "gemini-2.5-pro",
			"opencode":   "google/gemini-2.5-pro",
			"openrouter": "google/gemini-2.5-pro",
		},
	},
}

// FindModelFamily resolves any alias or provider-specific model string to its canonical family.
func FindModelFamily(query string) *ModelFamily {
	for _, fam := range BuiltinModelCatalog {
		if fam.CanonicalName == query {
			return &fam
		}
		for _, alias := range fam.Aliases {
			if alias == query {
				return &fam
			}
		}
		for _, provModel := range fam.ProviderIDs {
			if provModel == query {
				return &fam
			}
		}
	}
	return nil
}
