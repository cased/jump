package aws

import (
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	jump "github.com/cased/jump/types/v1alpha"
)

// The ECS auto-discovery Provider queries ECS for running containers on EC2 instances and constructs the `docker exec` arguments necessary to run a command inside those containers.
//
// Filters
//
// The ECS Provider accepts the following filters:
//
// - region: The AWS region to query. Defaults to the current region.
// - cluster: The ECS cluster to query. Defaults to the 'default cluster'.
// - task-group: The name of the ECS Task Group.
// - container-name: The name of a running Container.
//
// Sorting
//
// The ECS Provider supports sorting by the following keys:
//
// - startedAt
//
// Annotations
//
// The ECS Provider appends the following annotations to each Prompt:
//
// - startedAt: The time the container was started.
type ECS struct {
	cache *ecsCache

	EC2Interface EC2Interface
	ECSInterface ECSInterface
	STSInterface STSInterface
}
type ecsCache struct {
	ec2InstancePrivateDnsNames map[string]string
	taskContainerArns          map[string]string
}

type ECSProviderConfig struct {
	EC2Interface EC2Interface
	ECSInterface ECSInterface
	STSInterface STSInterface
}

func (provider *ECS) Initialize(providerConfig interface{}) {
	if providerConfig != nil {
		typedConfig := providerConfig.(ECSProviderConfig)
		provider.EC2Interface = typedConfig.EC2Interface
		provider.ECSInterface = typedConfig.ECSInterface
		provider.STSInterface = typedConfig.STSInterface
	}
}

func (provider *ECS) Discover(queries []*jump.PromptQuery) ([]*jump.Prompt, error) {
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

func (provider *ECS) Query(query *jump.PromptQuery) ([]*jump.Prompt, error) {
	cache := &ecsCache{
		ec2InstancePrivateDnsNames: make(map[string]string),
		taskContainerArns:          make(map[string]string),
	}
	provider.cache = cache

	regionSession, err := GetAWSSession(query.Filters["region"])
	if err != nil {
		return nil, err
	}

	// AWS ECS endpoints
	if provider.ECSInterface == nil {
		provider.ECSInterface = ecs.New(regionSession)
	}

	// AWS EC2 endpoints
	if provider.EC2Interface == nil {
		provider.EC2Interface = ec2.New(regionSession)
	}

	// List all ECS containers, filter by cluster if provided
	listContainerInstancesInput := &ecs.ListContainerInstancesInput{}
	if query.Filters["cluster"] != "" {
		listContainerInstancesInput.Cluster = aws.String(query.Filters["cluster"])
	}
	containers, err := provider.ECSInterface.ListContainerInstances(listContainerInstancesInput)
	if err != nil {
		return nil, err
	}

	var prompts []*jump.Prompt

	// Loop through all matching containers and build the mapping of tasks running
	// on that container instance.
	for _, containerArn := range containers.ContainerInstanceArns {
		// We only want to display RUNNING tasks, not any that are PROVISIONING or STOPPING
		listTasksInput := &ecs.ListTasksInput{
			ContainerInstance: containerArn,
			DesiredStatus:     aws.String("RUNNING"),
		}
		// Filter by cluster if provided
		if query.Filters["cluster"] != "" {
			listTasksInput.Cluster = aws.String(query.Filters["cluster"])
		}
		result, err := provider.ECSInterface.ListTasks(listTasksInput)
		if err != nil {
			return nil, err
		}

		// We need more details about the tasks than ListTasks gives.
		describeTasksInput := &ecs.DescribeTasksInput{
			Tasks: result.TaskArns,
		}
		if query.Filters["cluster"] != "" {
			describeTasksInput.Cluster = aws.String(query.Filters["cluster"])
		}
		tasks, err := provider.ECSInterface.DescribeTasks(describeTasksInput)
		if err != nil {
			return nil, err
		}

		for _, task := range tasks.Tasks {
			if *task.LastStatus != "RUNNING" {
				continue
			}

			if query.Filters["task-group"] != "" {
				if *task.Group != query.Filters["task-group"] {
					continue
				}
			}

			describeContainerInstancesInput := &ecs.DescribeContainerInstancesInput{
				ContainerInstances: []*string{aws.String(*task.ContainerInstanceArn)},
			}
			if query.Filters["cluster"] != "" {
				describeContainerInstancesInput.Cluster = aws.String(query.Filters["cluster"])
			}
			ci, err := provider.ECSInterface.DescribeContainerInstances(describeContainerInstancesInput)
			if err != nil {
				return nil, err
			}

			// Not the cleanest API, but looked at Web Inspector how AWS event does it,
			// surprisingly kind of clunky
			containerInstance := ci.ContainerInstances[0]

			if provider.cache.ec2InstancePrivateDnsNames[*containerInstance.Ec2InstanceId] == "" {
				di, err := provider.EC2Interface.DescribeInstances(&ec2.DescribeInstancesInput{
					Filters: []*ec2.Filter{
						{
							Name: aws.String("instance-id"),
							Values: []*string{
								containerInstance.Ec2InstanceId,
							},
						},
					},
				})
				if err != nil {
					return nil, err
				}

				if len(di.Reservations) == 0 {
					return nil, errors.New("could not find any reservations")
				}

				// If we cared about a particular EC2 instance, we could let the user pick
				res := di.Reservations[0]
				if len(res.Instances) == 0 {
					return nil, errors.New("could not find any instances")
				}

				i := res.Instances[0]

				provider.cache.ec2InstancePrivateDnsNames[*containerInstance.Ec2InstanceId] = *i.PrivateDnsName
			}

			for _, container := range task.Containers {
				if query.Filters["container-name"] != "" {
					if *container.Name != query.Filters["container-name"] {
						continue
					}
				}

				taskContainer := fmt.Sprintf("%s/%s", *task.Group, *container.Name)
				prompt := &jump.Prompt{
					Kind:        "container",
					Name:        fmt.Sprintf("%s/%s", *task.Group, *container.Name),
					Hostname:    provider.cache.ec2InstancePrivateDnsNames[*containerInstance.Ec2InstanceId],
					JumpCommand: fmt.Sprintf("docker exec -it $(docker ps --filter \"label=com.amazonaws.ecs.container-name=%s\" --filter \"label=com.amazonaws.ecs.task-arn=%s\" -q | head -n1)", *container.Name, *container.TaskArn),
					Annotations: map[string]string{
						"startedAt": task.StartedAt.String(),
					},
				}
				prompts = append(prompts, provider.decoratePromptWithQuery(prompt, query))
				provider.cache.taskContainerArns[taskContainer] = *container.ContainerArn
			}
		}
	}

	switch query.SortBy {
	case "startedAt":
		sort.SliceStable(prompts, func(i, j int) bool {
			if query.SortOrder == "desc" {
				return prompts[i].Annotations["startedAt"] > prompts[j].Annotations["startedAt"]
			} else {
				return prompts[i].Annotations["startedAt"] < prompts[j].Annotations["startedAt"]
			}
		})
	}

	if query.Limit != 0 && len(prompts) > query.Limit {
		prompts = prompts[:query.Limit]
	}

	return prompts, nil
}

func (provider *ECS) decoratePromptWithQuery(prompt *jump.Prompt, query *jump.PromptQuery) *jump.Prompt {
	decoratedPrompt := prompt.DecorateWithQuery(query)
	filterKeysToLabels := []string{
		"region",
		"cluster",
		"task-group",
		"container-name",
	}
	for _, filterKey := range filterKeysToLabels {
		if query.Filters[filterKey] != "" {
			decoratedPrompt.Labels[filterKey] = query.Filters[filterKey]
		}
	}
	decoratedPrompt.Provider = "ecs"
	return decoratedPrompt
}
