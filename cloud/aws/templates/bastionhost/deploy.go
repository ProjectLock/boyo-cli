package bastionhost

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

const (
	Name        = "Bastion"
	Description = "Single EC2 instance (t3.micro) Security Group & VPC. Highly recommend updating the security group access."

	al2023Param = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
	sgName      = "boyo-bastion-sg"
)

func getLatestAMI(ctx context.Context, cfg aws.Config) (string, error) {
	ssmClient := ssm.NewFromConfig(cfg)

	out, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name: aws.String(al2023Param),
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest AMI from SSM: %w", err)
	}

	return aws.ToString(out.Parameter.Value), nil
}

func getOrCreateVpc(ctx context.Context, client *ec2.Client) (string, error) {
	vpcsOut, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to describe VPCs: %w", err)
	}

	if len(vpcsOut.Vpcs) > 0 {
		for _, vpc := range vpcsOut.Vpcs {
			if aws.ToBool(vpc.IsDefault) {
				return aws.ToString(vpc.VpcId), nil
			}
		}
		return aws.ToString(vpcsOut.Vpcs[0].VpcId), nil
	}

	fmt.Println("No VPC found in this region. Creating a Default VPC...")
	createOut, err := client.CreateDefaultVpc(ctx, &ec2.CreateDefaultVpcInput{})
	if err != nil {
		return "", fmt.Errorf("failed to create default VPC: %w", err)
	}

	vpcID := aws.ToString(createOut.Vpc.VpcId)
	fmt.Printf("Default VPC created (%s)\n", vpcID)
	return vpcID, nil
}

func getOrCreateSecurityGroup(ctx context.Context, client *ec2.Client, vpcID string) (string, error) {
	descOut, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{sgName},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err == nil && len(descOut.SecurityGroups) > 0 {
		existingID := aws.ToString(descOut.SecurityGroups[0].GroupId)
		fmt.Printf("Using existing Security Group '%s' (%s)\n", sgName, existingID)
		return existingID, nil
	}

	fmt.Printf("Creating Security Group '%s' in VPC %s...\n", sgName, vpcID)
	createOut, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(sgName),
		Description: aws.String("Allow inbound ssh traffic"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create security group: %w", err)
	}

	sgID := aws.ToString(createOut.GroupId)

	_, err = client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []types.IpPermission{
			// Port 22 (SSH / EC2 Instance Connect)
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Allow inbound SSH from anywhere"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize ingress rules: %w", err)
	}

	fmt.Printf("Authorized inbound port 22 for Security Group (%s)\n", sgID)
	return sgID, nil
}

func Deploy(region string) error {
	ctx := context.Background()

	fmt.Printf("Loading AWS config for region '%s'...\n", region)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	fmt.Println("Looking up latest Amazon Linux 2023 AMI ID...")
	amiID, err := getLatestAMI(ctx, cfg)
	if err != nil {
		return err
	}

	vpcID, err := getOrCreateVpc(ctx, ec2Client)
	if err != nil {
		return err
	}

	sgID, err := getOrCreateSecurityGroup(ctx, ec2Client, vpcID)
	if err != nil {
		return err
	}

	userDataScript := `#!/bin/bash
dnf update -y
`
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userDataScript))

	fmt.Println("Launching EC2 Bastion instance...")

	// 5. Launch Instance
	input := &ec2.RunInstancesInput{
		ImageId:          aws.String(amiID),
		InstanceType:     types.InstanceTypeT3Micro,
		MinCount:         aws.Int32(1),
		MaxCount:         aws.Int32(1),
		UserData:         aws.String(encodedUserData),
		SecurityGroupIds: []string{sgID},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("boyo-bastion"),
					},
					{
						Key:   aws.String("ManagedBy"),
						Value: aws.String("boyo-cli"),
					},
				},
			},
		},
	}

	result, err := ec2Client.RunInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to launch EC2 instance: %w", err)
	}

	if len(result.Instances) > 0 {
		instanceID := aws.ToString(result.Instances[0].InstanceId)

		fmt.Println("Waiting for instance to reach 'running' state and get assigned a Public IP...")

		// 1. Wait until the instance is running (times out after 5 minutes)
		waiter := ec2.NewInstanceRunningWaiter(ec2Client)
		err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		}, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("failed waiting for instance to run: %w", err)
		}

		// 2. Fetch updated details (including Public IP)
		descResult, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		})
		if err != nil || len(descResult.Reservations) == 0 || len(descResult.Reservations[0].Instances) == 0 {
			return fmt.Errorf("failed to fetch running instance details: %w", err)
		}

		runningInstance := descResult.Reservations[0].Instances[0]
		publicIP := aws.ToString(runningInstance.PublicIpAddress)

		fmt.Printf("EC2 Bastion host launched successfully!\n")
		fmt.Printf("Instance ID: %s\n", instanceID)
		fmt.Printf("VPC ID: %s\n", vpcID)
		fmt.Printf("Security Group: %s\n", sgID)
		fmt.Printf("Bastion host will be available at: %s\n", publicIP)
	} else {
		return fmt.Errorf("no instances were created ")
	}

	return nil
}
