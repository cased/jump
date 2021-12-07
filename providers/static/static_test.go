package static_test

import (
	"testing"

	"github.com/cased/jump/providers/static"
	jump "github.com/cased/jump/types/v1alpha"
	"gopkg.in/yaml.v2"
)

type testCase struct {
	Config string
	Tests  func(*testing.T, *jump.Prompt)
}

func TestStaticProvider(t *testing.T) {
	testTable := []testCase{
		{
			Config: `
queries:
- provider: static
  prompt:
    name: simple example
    hostname: example.com
    username: example
`,
			Tests: func(t *testing.T, p *jump.Prompt) {
				want := "example.com"
				if p.Hostname != want {
					t.Errorf("got %q, want %q", p.Hostname, want)
				}
				if *p.CloseTerminalOnExit != true {
					t.Errorf("got %v, want %v", p.CloseTerminalOnExit, true)
				}
			},
		},
		{
			Config: `
queries:
- provider: static
  prompt:
    name: example Command
    hostname: example.com
    username: example
    shellCommand: echo hello
    closeTerminalOnExit: false
`,
			Tests: func(t *testing.T, p *jump.Prompt) {
				want := "example"
				if p.Username != want {
					t.Errorf("got %q, want %q", p.Hostname, want)
				}
				if *p.CloseTerminalOnExit != false {
					t.Errorf("got %v, want %v", *p.CloseTerminalOnExit, false)
				}
			},
		},
	}
	for _, test := range testTable {
		c := &jump.AutoDiscoveryConfig{}
		err := yaml.Unmarshal([]byte(test.Config), c)
		if err != nil {
			t.Fatal(err)
		}

		static := &static.Static{}
		got, err := static.Discover(c.Queries)
		if err != nil {
			t.Fatal(err)
		}
		want := 1
		if len(got) != want {
			t.Fatalf("Expected %d result, got %d", want, len(got))
		}
		p := got[0]
		t.Run(p.Name, func(t *testing.T) {
			test.Tests(t, p)
		})
	}
}
