package providers

import (
	"github.com/cased/jump/providers/aws"
	"github.com/cased/jump/providers/static"
	jump "github.com/cased/jump/types/v1alpha"
)

// Registers all built-in providers with default configuration.
func Register() {
	jump.RegisterProvider("static", &static.Static{}, nil)
	jump.RegisterProvider("ecs", &aws.ECS{}, nil)
	jump.RegisterProvider("ec2", &aws.EC2{}, nil)
}
