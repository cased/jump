package providers_test

import (
	"testing"

	jump "github.com/cased/jump/types/v1alpha"
)

type ExampleProvider struct {
}

func (provider *ExampleProvider) Discover(queries []*jump.PromptQuery) ([]*jump.Prompt, error) {
	var prompts []*jump.Prompt
	prompts = append(prompts, &jump.Prompt{
		Hostname: "example.com",
		Username: "user",
		Provider: "example",
	})
	return prompts, nil
}

func TestExampleProvider(t *testing.T) {
	provider := &ExampleProvider{}
	prompts, err := provider.Discover([]*jump.PromptQuery{})
	if err != nil {
		t.Error(err)
	}
	if len(prompts) != 1 {
		t.Errorf("Expected 1 result, got %d", len(prompts))
	}
}
