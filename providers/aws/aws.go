package aws

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sts"
)

type EC2Interface interface {
	DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
}

type ECSInterface interface {
	ListTasks(input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
	DescribeTasks(input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	ListContainerInstances(input *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error)
	DescribeContainerInstances(input *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error)
}

type STSInterface interface {
	GetCallerIdentity(input *sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error)
}

type MockSTS struct {
}

func (s *MockSTS) GetCallerIdentity(input *sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error) {
	return &sts.GetCallerIdentityOutput{
		Account: aws.String("123456789012"),
		Arn:     aws.String("arn:aws:sts::123456789012:assumed-role/jump-role/jump-role-20190402T185959Z"),
		UserId:  aws.String("jump-role"),
	}, nil
}

type EC2MetadataInterface interface {
	Region() (string, error)
}

type MockEC2Metadata struct {
}

func (m *MockEC2Metadata) Region() (string, error) {
	return "us-notexist-1", nil
}

var regionSessions map[string]*session.Session
var defaultRegion string
var DefaultMetadataInterface EC2MetadataInterface
var DefaultSTSInterface STSInterface

func init() {
	regionSessions = make(map[string]*session.Session)
}

func getRegion() (region string, err error) {
	if region = os.Getenv("AWS_REGION"); region != "" {
		return region, nil
	}
	if region = os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region, nil
	}
	svc := DefaultMetadataInterface
	if svc == nil {
		svc = ec2metadata.New(session.Must(session.NewSession()), aws.NewConfig())
	}
	region, err = svc.Region()
	if err == nil && region != "" {
		return region, nil
	}
	return "", fmt.Errorf("could not load region from query, AWS_DEFAULT_REGION, AWS_REGION, or EC2 metadata api: %w", err)
}

func GetAWSSession(region string) (*session.Session, error) {
	if regionSessions == nil {
		regionSessions = make(map[string]*session.Session)
	}

	if region == "" && defaultRegion == "" {
		defaultRegion, err := getRegion()
		if err != nil {
			return nil, err
		}
		region = defaultRegion
	}

	if regionSessions[region] == nil {
		var regionSession *session.Session
		var err error

		regionSession, err = session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Config: aws.Config{
				Region:                        aws.String(region),
				CredentialsChainVerboseErrors: aws.Bool(true),
				Endpoint:                      aws.String(os.Getenv("AWS_ENDPOINT")),
			},
		})
		if err != nil {
			return nil, err
		}

		svc := DefaultSTSInterface
		if svc == nil {
			svc = sts.New(regionSession)
		}

		result, err := svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			return nil, err
		}
		if os.Getenv("LOG_LEVEL") == "debug" {
			log.Printf("[aws] Authenticated as %s in %s\n", *result.Arn, region)
		}

		regionSessions[region] = regionSession
	}
	return regionSessions[region], nil
}
