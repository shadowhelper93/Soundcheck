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
		CanonicalName: "claude-sonnet-5-5",
		Aliases:       []string{"claude-sonnet-5.5", "sonnet-5.5", "sonnet"},
		ProviderIDs: map[string]string{
			"claude":     "claude-sonnet-5-5",
			"opencode":   "anthropic/claude-sonnet-5-5",
			"openrouter": "anthropic/claude-sonnet-5.5",
			"bedrock":    "anthropic.claude-sonnet-5-5",
		},
	},
	{
		CanonicalName: "claude-opus-5-5",
		Aliases:       []string{"claude-opus-5.5", "opus-5.5", "opus"},
		ProviderIDs: map[string]string{
			"claude":     "claude-opus-5-5",
			"opencode":   "anthropic/claude-opus-5-5",
			"openrouter": "anthropic/claude-opus-5.5",
			"bedrock":    "anthropic.claude-opus-5-5",
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
