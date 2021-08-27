package v1alpha

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

type AutoDiscoveryConfig struct {
	Queries []*PromptQuery `yaml:"queries"`
}

// A map of registered providers.
var Providers map[string]Provider

func init() {
	Providers = make(map[string]Provider)
}

// Register a Provider.
func RegisterProvider(providerName string, provider Provider, providerConfig interface{}) {
	provider.Initialize(providerConfig)
	Providers[providerName] = provider
}

func LoadAutoDiscoveryConfigFromPaths(paths []string) (*AutoDiscoveryConfig, error) {
	mergedConfig := &AutoDiscoveryConfig{}
	for _, path := range paths {
		config := &AutoDiscoveryConfig{}
		yamlFile, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if strings.Contains(path, ".json") {
			err = json.Unmarshal(yamlFile, config)
		} else {
			err = yaml.Unmarshal(yamlFile, config)
		}
		if err != nil {
			return nil, err
		}
		mergedConfig.Queries = append(mergedConfig.Queries, config.Queries...)
	}

	// Ensure all queries have a provider
	_, err := mergedConfig.queriesByProvider()
	if err != nil {
		return nil, err
	}

	return mergedConfig, nil
}

func (config *AutoDiscoveryConfig) queriesByProvider() (map[string][]*PromptQuery, error) {
	ret := make(map[string][]*PromptQuery)
	for _, query := range config.Queries {
		if Providers[query.Provider] != nil {
			if ret[query.Provider] == nil {
				ret[query.Provider] = make([]*PromptQuery, 0)
			}
			ret[query.Provider] = append(ret[query.Provider], query)
		} else {
			return nil, fmt.Errorf("unknown provider %s", query.Provider)
		}
	}
	return ret, nil
}

// Dispatches PromptQueries to each registered Provider. Returns a list of Prompts.
func (config *AutoDiscoveryConfig) DiscoverPrompts() ([]*Prompt, error) {
	prompts := make([]*Prompt, 0)
	queries, err := config.queriesByProvider()
	if err != nil {
		return nil, err
	}
	for providerName, providerQueries := range queries {
		provider := Providers[providerName]
		providerPrompts, err := provider.Discover(providerQueries)
		if err != nil {
			log.Println(err)
			continue
		}
		prompts = append(prompts, providerPrompts...)
	}

	return prompts, nil
}

func WriteAutoDiscoveryManifestToPath(prompts []*Prompt, path string) error {
	sort.SliceStable(prompts, func(i, j int) bool {
		return prompts[i].Provider < prompts[j].Provider
	})

	manifest := &AutoDiscoveryManifest{
		Prompts: prompts,
	}
	file, err := json.MarshalIndent(manifest, "", " ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, file, 0644)
	if err != nil {
		return err
	}
	return nil
}
