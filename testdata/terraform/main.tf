provider "aws" {
  region                     = "us-east-1"
  access_key                 = "anything"
  secret_key                 = "goes"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    ec2 = "http://localstack:4566"
    sts = "http://localstack:4566"
    iam = "http://localstack:4566"
  }
}

data "aws_iam_policy_document" "ecs-tasks-policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "jump-test-role" {
  name               = "jump-test-role"
  path               = "/"
  assume_role_policy = data.aws_iam_policy_document.ecs-tasks-policy.json
}

# Create a few fake instances

resource "aws_instance" "jump-test-1" {
  ami           = "ami-0d57c0143330e1fa7"
  instance_type = "t2.micro"

  tags = {
    Name                        = "jump-test-1"
    "aws:autoscaling:groupName" = "jump-test-1"
  }
}

resource "aws_instance" "jump-test-2" {
  ami           = "ami-0d57c0143330e1fa7"
  instance_type = "t2.micro"

  tags = {
    Name                        = "jump-test-2"
    "aws:autoscaling:groupName" = "jump-test-2"
  }
}