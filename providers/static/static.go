package static

import (
	jump "github.com/cased/jump/types/v1alpha"
)

// The Static auto-discovery Provider is a bit of an oxymoron: it performs no discovery and returns the Prompt passed to it.
// Use it to include a list of Static prompts alongside other auto-discovered entries.
//
// Filters
//
// The Static Provider ignores all provided filters.
//
// Sorting
//
// The Static Provider ignores all provided sorting options.
type Static struct {
}

func (provider *Static) Initialize(i interface{}) {
}

func (provider *Static) Discover(queries []*jump.PromptQuery) ([]*jump.Prompt, error) {
	var prompts []*jump.Prompt
	for _, query := range queries {
		prompt := &jump.Prompt{
			Provider: "static",
		}
		prompts = append(prompts, prompt.DecorateWithQuery(query))
	}
	return prompts, nil
}
