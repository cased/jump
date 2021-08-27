package static_test

import (
	"testing"

	"github.com/cased/jump/providers/static"
	jump "github.com/cased/jump/types/v1alpha"
	"gopkg.in/yaml.v2"
)

func TestStaticProvider(t *testing.T) {
	// TODO test tables etc
	config := `
queries:
  - provider: static
    prompt:
      hostname: example.com
      username: example
`
	c := &jump.AutoDiscoveryConfig{}
	err := yaml.Unmarshal([]byte(config), c)
	if err != nil {
		t.Error(err)
	}

	static := &static.Static{}
	got, err := static.Discover(c.Queries)
	if err != nil {
		t.Error(err)
	}
	if len(got) != 1 {
		t.Errorf("Expected 1 result, got %d", len(got))
	}
	if got[0].Hostname != "example.com" {
		t.Error("Expected example.com, got", got[0].Hostname)
	}
	if got[0].Username != "example" {
		t.Error("Expected example, got", got[0].Username)
	}
}
