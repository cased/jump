package aws_test

import (
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	aws_provider "github.com/cased/jump/providers/aws"
	jump "github.com/cased/jump/types/v1alpha"
	"github.com/kylelemons/godebug/pretty"
	"gopkg.in/yaml.v2"
)

func init() {
	aws_provider.DefaultMetadataInterface = &aws_provider.MockEC2Metadata{}
	aws_provider.DefaultSTSInterface = &aws_provider.MockSTS{}
}

type MockEC2 struct {
	Queries      []*jump.PromptQuery
	CurrentQuery *jump.PromptQuery

	DescribeInstancesFunc func(query *jump.PromptQuery, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}

func (m *MockEC2) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	// ORDERING DEPENDENT LOGIC ALERT
	// Call the first query that calls this function the current query
	m.CurrentQuery, m.Queries = m.Queries[0], m.Queries[1:]

	return m.DescribeInstancesFunc(m.CurrentQuery, input)
}
func TestEC2Provider(t *testing.T) {

	type ec2Test struct {
		Name        string
		YamlPath    string
		MockEC2     *MockEC2
		WantPrompts []*jump.Prompt
	}

	tests := []ec2Test{
		{
			Name:     "Single instance",
			YamlPath: "testdata/ec2_test_default.yml",
			MockEC2: &MockEC2{
				DescribeInstancesFunc: func(query *jump.PromptQuery, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					return &ec2.DescribeInstancesOutput{
						Reservations: []*ec2.Reservation{
							{
								Instances: []*ec2.Instance{
									{
										InstanceId:     aws.String("i-12345678"),
										PrivateDnsName: aws.String("12345678.example.com"),
										State: &ec2.InstanceState{
											Name: aws.String("running"),
										},
										LaunchTime: aws.Time(
											time.Date(2021, time.July, 11, 0, 0, 0, 0, time.UTC),
										),
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
			WantPrompts: jump.Prompts([]*jump.Prompt{
				{
					Name:        "i-12345678",
					Description: "An EC2 instance",
					Hostname:    "12345678.example.com",
					Kind:        "host",
					Provider:    "ec2",
					Annotations: map[string]string{
						"launchTime": "2021-07-11T00:00:00Z",
					},
				},
			}),
		},
		{
			Name:     "Filters",
			YamlPath: "testdata/ec2_test_filters.yml",
			MockEC2: &MockEC2{
				DescribeInstancesFunc: func(query *jump.PromptQuery, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					wantFilters := []*ec2.Filter{
						{
							Name: aws.String("tag:Name"),
							Values: []*string{
								aws.String("*test*"),
							},
						},
					}
					equal := reflect.DeepEqual(input.Filters, wantFilters)
					if !equal {
						t.Error(pretty.Compare(input.Filters, wantFilters))
					}
					return &ec2.DescribeInstancesOutput{
						Reservations: []*ec2.Reservation{
							{
								Instances: []*ec2.Instance{
									{
										InstanceId:     aws.String("i-12345678"),
										PrivateDnsName: aws.String("12345678.example.com"),
										State: &ec2.InstanceState{
											Name: aws.String("running"),
										},
										LaunchTime: aws.Time(
											time.Date(2021, time.July, 11, 0, 0, 0, 0, time.UTC),
										),
										Tags: []*ec2.Tag{
											{
												Key:   aws.String("Name"),
												Value: aws.String("test"),
											},
										},
									},
									{
										InstanceId:     aws.String("i-9101112"),
										PrivateDnsName: aws.String("9101112.example.com"),
										State: &ec2.InstanceState{
											Name: aws.String("running"),
										},
										LaunchTime: aws.Time(
											time.Date(2020, time.July, 11, 0, 0, 0, 0, time.UTC),
										),
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
			WantPrompts: jump.Prompts([]*jump.Prompt{
				{
					Name:        "i-12345678",
					Description: "The most recently launched test instance in us-south-1",
					Hostname:    "12345678.example.com",
					Kind:        "host",
					Provider:    "ec2",
					Labels: map[string]string{
						"region":   "us-south-1",
						"tag:Name": "*test*",
					},
					Annotations: map[string]string{
						"launchTime": "2021-07-11T00:00:00Z",
					},
				},
			}),
		},
	}

	for _, test := range tests {
		provider := &aws_provider.EC2{}
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
