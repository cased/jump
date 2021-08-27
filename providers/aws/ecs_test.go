package aws_test

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	aws_provider "github.com/cased/jump/providers/aws"
	jump "github.com/cased/jump/types/v1alpha"
	"github.com/kylelemons/godebug/pretty"
	"gopkg.in/yaml.v2"
)

type MockECS struct {
	Queries      []*jump.PromptQuery
	CurrentQuery *jump.PromptQuery

	TestListTasksInput                  func(*ecs.ListTasksInput)
	TestDescribeTasksInput              func(*ecs.DescribeTasksInput)
	TestListContainerInstancesInput     func(*ecs.ListContainerInstancesInput)
	TestDescribeContainerInstancesInput func(*ecs.DescribeContainerInstancesInput)

	ListTasksOutput                  *ecs.ListTasksOutput
	DescribeTasksOutput              *ecs.DescribeTasksOutput
	ListContainerInstancesOutput     *ecs.ListContainerInstancesOutput
	DescribeContainerInstancesOutput *ecs.DescribeContainerInstancesOutput

	ListTasksFunc                  func(*jump.PromptQuery, *ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
	DescribeTasksFunc              func(*jump.PromptQuery, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	ListContainerInstancesFunc     func(*jump.PromptQuery, *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error)
	DescribeContainerInstancesFunc func(*jump.PromptQuery, *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error)
}

func (m *MockECS) ListTasks(input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	// ORDERING DEPENDENT LOGIC ALERT
	// Call the first query that calls this function the current query
	m.CurrentQuery, m.Queries = m.Queries[0], m.Queries[1:]

	if m.TestListTasksInput != nil {
		m.TestListTasksInput(input)
	}
	if m.ListTasksFunc != nil {
		return m.ListTasksFunc(m.CurrentQuery, input)
	}
	return m.ListTasksOutput, nil
}

func (m *MockECS) DescribeTasks(input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	if m.TestDescribeTasksInput != nil {
		m.TestDescribeTasksInput(input)
	}
	if m.DescribeTasksFunc != nil {
		return m.DescribeTasksFunc(m.CurrentQuery, input)
	}
	return m.DescribeTasksOutput, nil
}

func (m *MockECS) ListContainerInstances(input *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error) {
	if m.TestListContainerInstancesInput != nil {
		m.TestListContainerInstancesInput(input)
	}
	if m.ListContainerInstancesFunc != nil {
		return m.ListContainerInstancesFunc(m.CurrentQuery, input)
	}
	return m.ListContainerInstancesOutput, nil
}

func (m *MockECS) DescribeContainerInstances(input *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error) {
	if m.TestDescribeContainerInstancesInput != nil {
		m.TestDescribeContainerInstancesInput(input)
	}
	if m.DescribeContainerInstancesFunc != nil {
		return m.DescribeContainerInstancesFunc(m.CurrentQuery, input)
	}
	return m.DescribeContainerInstancesOutput, nil
}

func TestECSProvider(t *testing.T) {

	type ecsTest struct {
		Name        string
		YamlPath    string
		MockEC2     *MockEC2
		MockECS     *MockECS
		WantPrompts []*jump.Prompt
	}

	tests := []ecsTest{
		{
			Name:     "Default ECS cluster with one container and no filters",
			YamlPath: "testdata/ecs_test_default.yml",
			WantPrompts: []*jump.Prompt{
				{
					Hostname:    "12345678.example.com",
					Name:        "example-service/example-container-name",
					JumpCommand: "docker exec -it $(docker ps --filter \"label=com.amazonaws.ecs.container-name=example-container-name\" --filter \"label=com.amazonaws.ecs.task-arn=arn:aws:ecs:us-east-1:123456789012:task/example-task-id\" -q | head -n1)",
					Kind:        "container",
					Provider:    "ecs",
					Description: "Default container debug shell",
					Annotations: map[string]string{
						"startedAt": "2015-03-26 19:54:00 +0000 UTC",
					}},
			},
			MockEC2: &MockEC2{
				DescribeInstancesFunc: func(query *jump.PromptQuery, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					return &ec2.DescribeInstancesOutput{
						Reservations: []*ec2.Reservation{
							{
								Instances: []*ec2.Instance{
									{
										InstanceId:     aws.String("i-12345678"),
										PrivateDnsName: aws.String("12345678.example.com"),
										Tags: []*ec2.Tag{
											{
												Key:   aws.String("Name"),
												Value: aws.String("test"),
											},
										},
									},
								},
							},
						},
					}, nil
				},
			},
			MockECS: &MockECS{
				TestListTasksInput: func(input *ecs.ListTasksInput) {
					if input.Cluster != nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
				},
				ListTasksOutput: &ecs.ListTasksOutput{
					TaskArns: []*string{
						aws.String("arn:aws:ecs:us-east-1:123456789012:task/example-task-id"),
					},
				},
				TestDescribeTasksInput: func(input *ecs.DescribeTasksInput) {
					if input.Cluster != nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
				},
				DescribeTasksOutput: &ecs.DescribeTasksOutput{
					Tasks: []*ecs.Task{
						{
							LastStatus:           aws.String("RUNNING"),
							TaskArn:              aws.String("arn:aws:ecs:us-east-1:123456789012:task/example-task-id"),
							ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
							Group:                aws.String("example-service"),
							StartedAt: aws.Time(
								time.Date(2015, time.March, 26, 19, 54, 0, 0, time.UTC),
							),
							Containers: []*ecs.Container{
								{
									Name:         aws.String("example-container-name"),
									TaskArn:      aws.String("arn:aws:ecs:us-east-1:123456789012:task/example-task-id"),
									ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/example-container-id"),
								},
							},
						},
					},
				},
				TestListContainerInstancesInput: func(input *ecs.ListContainerInstancesInput) {
					if input.Cluster != nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
				},
				ListContainerInstancesOutput: &ecs.ListContainerInstancesOutput{
					ContainerInstanceArns: []*string{
						aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
					},
				},
				TestDescribeContainerInstancesInput: func(input *ecs.DescribeContainerInstancesInput) {
					if input.Cluster != nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
				},
				DescribeContainerInstancesOutput: &ecs.DescribeContainerInstancesOutput{
					ContainerInstances: []*ecs.ContainerInstance{
						{
							ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
							Ec2InstanceId:        aws.String("i-12345678"),
							AgentConnected:       aws.Bool(true),
							PendingTasksCount:    aws.Int64(1),
							RunningTasksCount:    aws.Int64(1),
							Attributes: []*ecs.Attribute{
								{
									Name:  aws.String("example-attribute-name"),
									Value: aws.String("example-attribute-value"),
								},
							},
						},
					},
				},
			},
		},
		{
			Name:     "Multiple queries, complex filters",
			YamlPath: "testdata/ecs_test_complex_filters.yml",
			WantPrompts: []*jump.Prompt{
				{
					Hostname:     "12345678.test-cluster.us-west-1.example.com",
					JumpCommand:  "docker exec -it $(docker ps --filter \"label=com.amazonaws.ecs.container-name=test-container-name\" --filter \"label=com.amazonaws.ecs.task-arn=arn:aws:ecs:us-east-1:123456789012:task/test-task-id\" -q | head -n1)",
					ShellCommand: "./bin/rails console",
					Kind:         "container",
					Provider:     "ecs",
					Name:         "Test Rails Console",
					Description:  "Use to perform exploratory debugging on the test cluster",
					Labels: map[string]string{
						"region":      "us-west-1",
						"cluster":     "test-cluster",
						"environment": "test",
						"task-group":  "test-service",
					},
					Annotations: map[string]string{
						"startedAt": "2015-03-26 19:54:00 +0000 UTC",
					},
				},
				{
					Hostname:     "12345678.prod-cluster.us-west-2.example.com",
					JumpCommand:  "docker exec -it $(docker ps --filter \"label=com.amazonaws.ecs.container-name=prod-container-name\" --filter \"label=com.amazonaws.ecs.task-arn=arn:aws:ecs:us-east-1:123456789012:task/prod-task-id-2\" -q | head -n1)",
					ShellCommand: "./bin/rails console",
					Kind:         "container",
					Provider:     "ecs",
					Name:         "Production Rails Console",
					Description:  "Use to perform exploratory debugging on the production cluster",
					Labels: map[string]string{
						"region":         "us-west-2",
						"cluster":        "prod-cluster",
						"container-name": "prod-container-name",
						"environment":    "prod",
					},
					Annotations: map[string]string{
						"startedAt": "2021-03-26 19:54:00 +0000 UTC",
					},
				},
			},
			MockEC2: &MockEC2{
				DescribeInstancesFunc: func(query *jump.PromptQuery, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					return &ec2.DescribeInstancesOutput{
						Reservations: []*ec2.Reservation{
							{
								Instances: []*ec2.Instance{
									{
										InstanceId:     aws.String("i-12345678"),
										PrivateDnsName: aws.String(fmt.Sprintf("12345678.%s.%s.example.com", query.Filters["cluster"], query.Filters["region"])),
										Tags: []*ec2.Tag{
											{
												Key:   aws.String("Name"),
												Value: aws.String(query.Filters["cluster"]),
											},
										},
									},
								},
							},
						},
					}, nil
				},
			},
			MockECS: &MockECS{
				ListTasksFunc: func(query *jump.PromptQuery, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
					if input.Cluster == nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
					got := *input.Cluster
					want := query.Filters["cluster"]
					if got != want {
						t.Errorf("Got %s, wanted %s", got, want)
					}
					switch query.Filters["cluster"] {
					case "test-cluster":
						return &ecs.ListTasksOutput{
							TaskArns: []*string{
								aws.String("arn:aws:ecs:us-east-1:123456789012:task/test-task-id"),
								aws.String("arn:aws:ecs:us-east-1:123456789012:task/test-different-task-group"),
							},
						}, nil
					case "prod-cluster":
						return &ecs.ListTasksOutput{
							TaskArns: []*string{
								aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-task-id"),
								aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-id-2"),
							},
						}, nil
					default:
						return nil, nil
					}
				},
				DescribeTasksFunc: func(query *jump.PromptQuery, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
					if input.Cluster == nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
					got := *input.Cluster
					want := query.Filters["cluster"]
					if got != want {
						t.Errorf("Got %s, wanted %s", got, want)
					}
					switch query.Filters["cluster"] {
					case "test-cluster":
						return &ecs.DescribeTasksOutput{
							Tasks: []*ecs.Task{
								{
									LastStatus:           aws.String("RUNNING"),
									TaskArn:              aws.String("arn:aws:ecs:us-east-1:123456789012:task/test-task-id"),
									ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
									Group:                aws.String("test-service"),
									StartedAt: aws.Time(
										time.Date(2015, time.March, 26, 19, 54, 0, 0, time.UTC),
									),
									Containers: []*ecs.Container{
										{
											Name:         aws.String("test-container-name"),
											TaskArn:      aws.String("arn:aws:ecs:us-east-1:123456789012:task/test-task-id"),
											ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/test-container-id"),
										},
									},
								},
								{
									LastStatus:           aws.String("RUNNING"),
									TaskArn:              aws.String("arn:aws:ecs:us-east-1:123456789012:task/test-different-task-group"),
									ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
									Group:                aws.String("test-different-service"),
									StartedAt: aws.Time(
										time.Date(2015, time.March, 26, 19, 54, 0, 0, time.UTC),
									),
									Containers: []*ecs.Container{
										{
											Name:         aws.String("test-different-name"),
											TaskArn:      aws.String("arn:aws:ecs:us-east-1:123456789012:task/test-different-task-group"),
											ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/test-different-container-id"),
										},
									},
								},
							},
						}, nil
					case "prod-cluster":
						return &ecs.DescribeTasksOutput{
							Tasks: []*ecs.Task{
								{
									LastStatus:           aws.String("RUNNING"),
									TaskArn:              aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-task-id"),
									ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
									Group:                aws.String("prod-service"),
									StartedAt: aws.Time(
										time.Date(2015, time.March, 26, 19, 54, 0, 0, time.UTC),
									),
									Containers: []*ecs.Container{
										{
											Name:         aws.String("prod-container-name"),
											TaskArn:      aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-task-id"),
											ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/prod-container-id"),
										},
										{
											Name:         aws.String("prod-sidecar-name"),
											TaskArn:      aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-task-id"),
											ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/prod-sidecar-id"),
										},
									},
								},
								{
									LastStatus:           aws.String("RUNNING"),
									TaskArn:              aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-task-id-2"),
									ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
									Group:                aws.String("prod-service"),
									StartedAt: aws.Time(
										time.Date(2021, time.March, 26, 19, 54, 0, 0, time.UTC),
									),
									Containers: []*ecs.Container{
										{
											Name:         aws.String("prod-container-name"),
											TaskArn:      aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-task-id-2"),
											ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/prod-container-id-2"),
										},
										{
											Name:         aws.String("prod-sidecar-name"),
											TaskArn:      aws.String("arn:aws:ecs:us-east-1:123456789012:task/prod-task-id-2"),
											ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/prod-sidecar-id-2"),
										},
									},
								},
							},
						}, nil
					default:
						return nil, nil
					}
				},
				TestListContainerInstancesInput: func(input *ecs.ListContainerInstancesInput) {
					if input.Cluster == nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
					got := *input.Cluster
					if !strings.Contains(got, "-cluster") {
						t.Errorf("Got %s, wanted %s", got, "*-cluster")
					}
				},
				ListContainerInstancesOutput: &ecs.ListContainerInstancesOutput{
					ContainerInstanceArns: []*string{
						aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
					},
				},
				TestDescribeContainerInstancesInput: func(input *ecs.DescribeContainerInstancesInput) {
					if input.Cluster == nil {
						t.Errorf("Expected no cluster filter, got %s", *input.Cluster)
					}
					got := *input.Cluster
					if !strings.Contains(got, "-cluster") {
						t.Errorf("Got %s, wanted %s", got, "*-cluster")
					}
				},
				DescribeContainerInstancesOutput: &ecs.DescribeContainerInstancesOutput{
					ContainerInstances: []*ecs.ContainerInstance{
						{
							ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container-instance/example-container-instance-id"),
							Ec2InstanceId:        aws.String("i-12345678"),
							AgentConnected:       aws.Bool(true),
							PendingTasksCount:    aws.Int64(1),
							RunningTasksCount:    aws.Int64(1),
							Attributes: []*ecs.Attribute{
								{
									Name:  aws.String("example-attribute-name"),
									Value: aws.String("example-attribute-value"),
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		provider := &aws_provider.ECS{}
		t.Run(test.Name, func(t *testing.T) {
			c := &jump.AutoDiscoveryConfig{}
			configFile, err := ioutil.ReadFile(test.YamlPath)
			if err != nil {
				t.Error(err)
			}

			err = yaml.Unmarshal([]byte(configFile), c)
			if err != nil {
				t.Error(err)
			}
			test.MockEC2.Queries = c.Queries
			provider.EC2Interface = test.MockEC2
			test.MockECS.Queries = c.Queries
			provider.ECSInterface = test.MockECS
			provider.STSInterface = &aws_provider.MockSTS{}
			got, err := provider.Discover(c.Queries)
			if err != nil {
				t.Error(err)
			}
			if len(got) != len(test.WantPrompts) {
				t.Fatalf("Got %d, wanted %d", len(got), len(test.WantPrompts))
			}

			for i := range got {
				equal := reflect.DeepEqual(got[i], test.WantPrompts[i])
				if !equal {
					t.Error(pretty.Compare(got[i], test.WantPrompts[i]))
				}
			}
		})
	}

}
