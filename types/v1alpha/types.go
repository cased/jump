package v1alpha

// A PromptQuery is a query for a Prompt.
type PromptQuery struct {
	Provider  string            `yaml:"provider"`            // The name of a registered Provider to use to perform this query.
	Filters   map[string]string `yaml:"filters,omitempty"`   // A map of filters. Each Provider defines its own filters.
	Limit     int               `yaml:"limit,omitempty"`     // The maximum number of results to return.
	SortBy    string            `yaml:"sortBy,omitempty"`    // The field to sort results by, passed to the Provider.
	SortOrder string            `yaml:"sortOrder,omitempty"` // The order in which to sort results, passed to the Provider.
	Prompt    *Prompt           `yaml:"prompt,omitempty"`    // A Prompt template, which can be used to give all returned results a common name, description, etc.
}

// A Prompt represents an interactive command line, and can represent the initial shell presented by an SSH connection to a host OR the interactive session presented by a command run on that host.
type Prompt struct {
	Hostname          string            `json:"hostname" yaml:"hostname,omitempty"`                             // The hostname to establish an SSH connection to. Use only for display purposes if IpAddress is provided.
	Username          string            `json:"username,omitempty" yaml:"username,omitempty"`                   // The username to use to establish an SSH connection to the host.
	IpAddress         string            `json:"ipAddress,omitempty" yaml:"ipAddress,omitempty"`                 // Optional: the IP address to establish an SSH connection to.
	Port              string            `json:"port,omitempty" yaml:"port,omitempty"`                           // Optional: the port to use when establishing an SSH connection.
	Name              string            `json:"name,omitempty" yaml:"name,omitempty"`                           // A descriptive name for this Prompt.
	Description       string            `json:"description,omitempty" yaml:"description,omitempty"`             // A longer description of this Prompt. Use this field to explain common use cases.
	JumpCommand       string            `json:"jumpCommand,omitempty" yaml:"jumpCommand,omitempty"`             // Optional: the command to run after establishing an SSH connection to the host
	ShellCommand      string            `json:"shellCommand,omitempty" yaml:"shellCommand,omitempty"`           // Optional: the command to run on the host or container to start the interactive session.
	Kind              string            `json:"kind,omitempty" yaml:"kind,omitempty"`                           // The kind of prompt. Currently valid values are "host" and "container".
	Provider          string            `json:"provider,omitempty" yaml:"provider,omitempty"`                   // The name of the provider that created this prompt.
	Labels            map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`                       // Optional: a map of key-value pairs containing additional information about this Prompt. The Cased Shell Dashboard may in the future provide functionality to filter Prompts by label values.
	Annotations       map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`             // Optional: a map of key-value pairs containing additional information about this Prompt. The Cased Shell Dashboard may in the future display these values alongside the Prompt, but is not expected to use them for filtering.
	Principals        []string          `json:"principals,omitempty" yaml:"principals,omitempty"`               // Optional: a list of users and groups that should have access to this Prompt. The Cased Shell Dashboard will use this information to conditionally display this Prompt.
	Featured          bool              `json:"featured,omitempty" yaml:"featured,omitempty"`                   // Optional: whether this Prompt should be featured in the Cased Shell Dashboard.
	PromptForKey      bool              `json:"promptForKey,omitempty" yaml:"promptForKey,omitempty"`           // Set to true to tell the Cased Shell Dashboard to prompt the user for a key.
	PromptForUsername bool              `json:"promptForUsername,omitempty" yaml:"promptForUsername,omitempty"` // Set to true to tell the Cased Shell Dashboard to prompt the user for a username.
	ProxyJumpSelector map[string]string `json:"proxyJumpSelector,omitempty" yaml:"proxyJumpSelector,omitempty"` // Optional: a map of key-value pairs matching the labels on an existing prompt. If a matching prompt is found, connections to the prompt containing the ProxyHostJump attribute will be proxied via the matching prompt, similar to SSH's `ProxyJump` option.

	// TODO combine JumpCommand and ShellCommand into a single InitialCommand when serializing to JSON
	// InitialCommand    string            `json:"initialCommand,omitempty" yaml:"initialCommand,omitempty"`
}

// A Provider turns a list of PromptQueries into a list of Prompts.
type Provider interface {
	Initialize(interface{})
	Discover([]*PromptQuery) ([]*Prompt, error)
}

// An AutoDiscoveryManifest is a JSON-encoded list of Prompts.
// The Cased Shell application reads this manifest and uses it to display a list of Prompts to the user.
type AutoDiscoveryManifest struct {
	Prompts []*Prompt `json:"prompts"`
}

// Merges any fields provided here with fields returned by their respective searches.
// Each Provider is expected to call this function at the right time for their use case.
func (p *Prompt) DecorateWithQuery(query *PromptQuery) *Prompt {
	if query.Prompt != nil {
		if query.Prompt.Hostname != "" {
			p.Hostname = query.Prompt.Hostname
		}
		if query.Prompt.IpAddress != "" {
			p.IpAddress = query.Prompt.IpAddress
		}
		if query.Prompt.Port != "" {
			p.Port = query.Prompt.Port
		}
		if query.Prompt.Name != "" {
			p.Name = query.Prompt.Name
		}
		if query.Prompt.Description != "" {
			p.Description = query.Prompt.Description
		}
		if query.Prompt.Username != "" {
			p.Username = query.Prompt.Username
		}
		if query.Prompt.JumpCommand != "" {
			p.JumpCommand = query.Prompt.JumpCommand
		}
		if query.Prompt.ShellCommand != "" {
			p.ShellCommand = query.Prompt.ShellCommand
		}
		if query.Prompt.Kind != "" {
			p.Kind = query.Prompt.Kind
		}
		if query.Prompt.Labels != nil {
			p.Labels = query.Prompt.Labels
		}
		if query.Prompt.Principals != nil {
			p.Principals = query.Prompt.Principals
		}
		p.Featured = query.Prompt.Featured
		p.PromptForKey = query.Prompt.PromptForKey
		p.PromptForUsername = query.Prompt.PromptForUsername
		if query.Prompt.ProxyJumpSelector != nil {
			p.ProxyJumpSelector = query.Prompt.ProxyJumpSelector
		}
	}
	return p
}
