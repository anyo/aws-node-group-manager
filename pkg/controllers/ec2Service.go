package controllers

import (
	"log"

	apiTypes "github.com/anyo/aws-node-group-manager/pkg/apis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

//Ec2Service represents ssm operations
type Ec2Service struct {
	AwsSession session.Session
	Region     string
}

//GetLaunchTemplate represents
func (r *Ec2Service) GetLaunchTemplate(name string) *ec2.LaunchTemplate {
	asgSvc := ec2.New(&r.AwsSession)

	input := ec2.DescribeLaunchTemplatesInput{}
	response, err := asgSvc.DescribeLaunchTemplates(&input)
	if err != nil {
		log.Fatal("Error while getting asgs", err)
	}

	if response.LaunchTemplates == nil {
		return nil
	}

	return response.LaunchTemplates[0]
}

//GetLaunchTemplates represents
func (r *Ec2Service) GetLaunchTemplates() []*ec2.LaunchTemplate {
	asgSvc := ec2.New(&r.AwsSession)

	input := ec2.DescribeLaunchTemplatesInput{}
	response, err := asgSvc.DescribeLaunchTemplates(&input)
	if err != nil {
		log.Fatal("Error while getting asgs", err)
	}

	if response.LaunchTemplates == nil {
		return nil
	}

	return response.LaunchTemplates
}

// CreateLaunchTemplate represents
func (r *Ec2Service) CreateLaunchTemplate(configOptions *apiTypes.LaunchTemplateOptions) (*ec2.LaunchTemplate, error) {
	ec2Svc := ec2.New(&r.AwsSession)

	tags := []*ec2.Tag{}
	for i, v := range configOptions.Tags {
		t := ec2.Tag{Key: aws.String(i), Value: aws.String(v)}

		tags = append(tags, &t)
	}

	instanceTags := ec2.LaunchTemplateTagSpecificationRequest{ResourceType: aws.String("instance"), Tags: tags}
	volumeTags := ec2.LaunchTemplateTagSpecificationRequest{ResourceType: aws.String("volume"), Tags: tags}
	tagSpecificationRequest := []*ec2.LaunchTemplateTagSpecificationRequest{&instanceTags, &volumeTags}

	networkInterface := ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{
		AssociatePublicIpAddress: aws.Bool(configOptions.PublicIps),
		Groups:                   configOptions.SecurityGroups,
		DeviceIndex:              aws.Int64(0),
		DeleteOnTermination:      aws.Bool(true),
	}
	networkInterfaces := []*ec2.LaunchTemplateInstanceNetworkInterfaceSpecificationRequest{&networkInterface}

	bdm := []*ec2.LaunchTemplateBlockDeviceMappingRequest{}
	deviceMapping := ec2.LaunchTemplateBlockDeviceMappingRequest{
		DeviceName: aws.String("/dev/xvda"),
		Ebs: &ec2.LaunchTemplateEbsBlockDeviceRequest{
			VolumeSize: aws.Int64(configOptions.EbsVolumeSize),
			VolumeType: aws.String("gp2"),
		},
	}

	bdm = append(bdm, &deviceMapping)

	templateRequest := &ec2.RequestLaunchTemplateData{
		BlockDeviceMappings: bdm,
		IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Name: aws.String(configOptions.IamInstanceProfile),
		},
		ImageId:           aws.String(configOptions.AmiID),
		InstanceType:      aws.String(configOptions.InstanceType),
		KeyName:           aws.String(configOptions.KeyName),
		UserData:          aws.String(configOptions.UserData),
		TagSpecifications: tagSpecificationRequest,
		NetworkInterfaces: networkInterfaces,
	}

	input := ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String(configOptions.Name),
		LaunchTemplateData: templateRequest,
	}

	response, err := ec2Svc.CreateLaunchTemplate(&input)
	if err != nil {
		log.Fatal("Error creating new launch template", err)
		return nil, err
	}

	return response.LaunchTemplate, nil
}
