package v1alpha_test

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type MockEC2 struct {
	DescribeInstancesOutput *ec2.DescribeInstancesOutput
}

func (m *MockEC2) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return m.DescribeInstancesOutput, nil
}

type MockECS struct {
	ListTasksOutput                  *ecs.ListTasksOutput
	DescribeTasksOutput              *ecs.DescribeTasksOutput
	ListContainerInstancesOutput     *ecs.ListContainerInstancesOutput
	DescribeContainerInstancesOutput *ecs.DescribeContainerInstancesOutput
}

func (m *MockECS) ListTasks(input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	return m.ListTasksOutput, nil
}
func (m *MockECS) DescribeTasks(input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	return m.DescribeTasksOutput, nil
}
func (m *MockECS) ListContainerInstances(input *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error) {
	return m.ListContainerInstancesOutput, nil
}
func (m *MockECS) DescribeContainerInstances(input *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error) {
	return m.DescribeContainerInstancesOutput, nil
}

var (
	instanceOne = &ec2.Instance{
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
	}
	instanceTwo = &ec2.Instance{
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
	}

	describeTwoInstanceOutput = &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{
				Instances: []*ec2.Instance{instanceOne, instanceTwo},
			},
		},
	}

	mockEC2twoInstance = &MockEC2{
		DescribeInstancesOutput: describeTwoInstanceOutput,
	}

	mockECSoneContainer = &MockECS{
		ListTasksOutput: &ecs.ListTasksOutput{
			TaskArns: []*string{
				aws.String("arn:aws:ecs:us-east-1:012345678910:task/01234567-0123-0123-0123-012345678910"),
			},
		},
		DescribeTasksOutput: &ecs.DescribeTasksOutput{
			Tasks: []*ecs.Task{
				{
					LastStatus:           aws.String("RUNNING"),
					TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/01234567-0123-0123-0123-012345678910"),
					ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/01234567-0123-0123-0123-012345678910"),
					Group:                aws.String("example-service"),
					StartedAt: aws.Time(
						time.Date(2015, time.March, 26, 19, 54, 0, 0, time.UTC),
					),

					Containers: []*ecs.Container{
						{
							Name:         aws.String("test"),
							TaskArn:      aws.String("arn:aws:ecs:us-east-1:012345678910:task/01234567-0123-0123-0123-012345678910"),
							ContainerArn: aws.String("arn:aws:ecs:us-east-1:123456789012:container/example-container-id"),
						},
					},
				},
			},
		},
		ListContainerInstancesOutput: &ecs.ListContainerInstancesOutput{
			ContainerInstanceArns: []*string{
				aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/01234567-0123-0123-0123-012345678910"),
			},
		},
		DescribeContainerInstancesOutput: &ecs.DescribeContainerInstancesOutput{
			ContainerInstances: []*ecs.ContainerInstance{
				{
					ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/01234567-0123-0123-0123-012345678910"),
					Ec2InstanceId:        aws.String("i-01234567"),
				},
			},
		},
	}
)
