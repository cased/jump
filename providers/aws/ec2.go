package aws

import (
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	jump "github.com/cased/jump/types/v1alpha"
)

// The EC2 auto-discovery Provider queries AWS for running EC2 instances.
//
// Filters
//
// The EC2 Provider accepts the following filters:
//
// - region: The AWS region to query. Defaults to the current region.
//
// In addition to the above filter keys, the EC2 Provider also accepts all keys that are valid for
// `ec2.DescribeInstanceInput.Filters`, documentation on which is available at https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#DescribeInstancesInput.
//
// Sorting
//
// The EC2 Provider supports sorting by the following keys:
//
// - launchTime
//
//
// Annotations
//
// The EC2 Provider appends the following annotations to each Prompt:
//
// - launchTime: The EC2 instance launch time.
type EC2 struct {
	EC2Interface EC2Interface
	STSInterface STSInterface
}

type EC2ProviderConfig struct {
	EC2Interface EC2Interface
	STSInterface STSInterface
}

func (provider *EC2) Initialize(providerConfig interface{}) {
	if providerConfig != nil {
		typedConfig := providerConfig.(EC2ProviderConfig)
		provider.EC2Interface = typedConfig.EC2Interface
		provider.STSInterface = typedConfig.STSInterface
	}
}

func (provider *EC2) Discover(queries []*jump.PromptQuery) ([]*jump.Prompt, error) {
	var prompts []*jump.Prompt
	for _, query := range queries {
		queryPrompts, err := provider.Query(query)
		if err != nil {
			log.Println(err)
			continue
		}
		prompts = append(prompts, queryPrompts...)
	}
	return prompts, nil
}

func (provider *EC2) Query(query *jump.PromptQuery) ([]*jump.Prompt, error) {

	regionSession, err := GetAWSSession(query.Filters["region"])
	if err != nil {
		return nil, err
	}

	// AWS EC2 endpoints
	if provider.EC2Interface == nil {
		provider.EC2Interface = ec2.New(regionSession)
	}

	var filters []*ec2.Filter
	for key, value := range query.Filters {
		if key == "region" {
			continue
		}
		filters = append(filters, &ec2.Filter{
			Name:   aws.String(key),
			Values: []*string{aws.String(value)},
		})
	}
	input := &ec2.DescribeInstancesInput{Filters: filters}
	di, err := provider.EC2Interface.DescribeInstances(input)
	if err != nil {
		return nil, err
	}

	var prompts []*jump.Prompt

	for _, reservation := range di.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State != nil && *instance.State.Name == "running" {
				prompt := &jump.Prompt{
					Kind:     "host",
					Name:     *instance.InstanceId,
					Hostname: *instance.PrivateDnsName,
					Annotations: map[string]string{
						"launchTime": instance.LaunchTime.Format(time.RFC3339),
					},
				}
				prompts = append(prompts, provider.decoratePromptWithQuery(prompt, query))

			}
		}
	}

	switch query.SortBy {
	case "launchTime":
		sort.SliceStable(prompts, func(i, j int) bool {
			if query.SortOrder == "desc" {
				return prompts[i].Annotations["launchTime"] > prompts[j].Annotations["launchTime"]
			} else {
				return prompts[i].Annotations["launchTime"] < prompts[j].Annotations["launchTime"]
			}
		})
	}

	if query.Limit != 0 && len(prompts) > query.Limit {
		prompts = prompts[:query.Limit]
	}

	return prompts, nil
}

func (provider *EC2) decoratePromptWithQuery(prompt *jump.Prompt, query *jump.PromptQuery) *jump.Prompt {
	decoratedPrompt := prompt.DecorateWithQuery(query)
	if len(query.Filters) > 0 {
		if decoratedPrompt.Labels == nil {
			decoratedPrompt.Labels = map[string]string{}
		}
		for key, value := range query.Filters {
			decoratedPrompt.Labels[key] = value
		}
	}
	decoratedPrompt.Provider = "ec2"
	return decoratedPrompt
}
