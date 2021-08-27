package v1alpha_test

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cased/jump/providers"
	"github.com/cased/jump/providers/aws"
	"github.com/cased/jump/providers/static"
	jump "github.com/cased/jump/types/v1alpha"
	"github.com/kylelemons/godebug/pretty"
)

func init() {
	aws.DefaultMetadataInterface = &aws.MockEC2Metadata{}
	aws.DefaultSTSInterface = &aws.MockSTS{}
}

func TestConfigRoundTrip(t *testing.T) {
	type test struct {
		Name         string
		ConfigPaths  []string
		ManifestPath string
		EC2Interface aws.EC2Interface
		ECSInterface aws.ECSInterface
	}
	tests := []test{
		{
			Name:         "example",
			ConfigPaths:  []string{"testdata/example_config.yaml", "testdata/example_config2.yaml", "testdata/empty.yaml"},
			ManifestPath: "testdata/example_manifest.json",
			EC2Interface: mockEC2twoInstance,
			ECSInterface: mockECSoneContainer,
		},
		{
			Name:         "terraform-defaults",
			ConfigPaths:  []string{"testdata/terraform-default-queries.json"},
			ManifestPath: "testdata/terraform-default-manifest.json",
			EC2Interface: mockEC2twoInstance,
			ECSInterface: mockECSoneContainer,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.Name, func(t *testing.T) {
			jump.RegisterProvider("static", &static.Static{}, nil)
			jump.RegisterProvider("ec2", &aws.EC2{}, aws.EC2ProviderConfig{
				EC2Interface: testCase.EC2Interface,
				STSInterface: &aws.MockSTS{},
			})
			jump.RegisterProvider("ecs", &aws.ECS{}, aws.ECSProviderConfig{
				EC2Interface: testCase.EC2Interface,
				ECSInterface: testCase.ECSInterface,
				STSInterface: &aws.MockSTS{},
			})

			config, err := jump.LoadAutoDiscoveryConfigFromPaths(testCase.ConfigPaths)
			if err != nil {
				t.Fatal(err)
			}
			prompts, err := config.DiscoverPrompts()
			if err != nil {
				t.Fatal(err)
			}
			err = jump.WriteAutoDiscoveryManifestToPath(prompts, testCase.ManifestPath+".generated")
			if err != nil {
				t.Fatal(err)
			}
			got, err := ioutil.ReadFile(testCase.ManifestPath + ".generated")
			if err != nil {
				t.Fatal(err)
			}
			err = os.Remove(testCase.ManifestPath + ".generated")
			if err != nil {
				t.Fatal(err)
			}
			want, err := ioutil.ReadFile(testCase.ManifestPath)
			if err != nil {
				err := ioutil.WriteFile(testCase.ManifestPath, got, 0644)
				if err != nil {
					t.Fatal(err)
				}
				t.Fatalf("%s written, please manually validate", testCase.ManifestPath)
			}
			if string(want) != string(got) {
				t.Fatal(pretty.Compare(string(want), string(got)))
			}
		})
	}
}

func TestConfigConfigValidProviders(t *testing.T) {
	providers.Register()
	_, err := jump.LoadAutoDiscoveryConfigFromPaths([]string{"testdata/example_invalid.yaml"})
	if err == nil {
		t.Error(errors.New("Expected error due to invalid provider"))
	}
}

// TODO validate each query against provider before running?
