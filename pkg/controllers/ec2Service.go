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

	tsr := ec2.LaunchTemplateTagSpecificationRequest{ResourceType: aws.String("instance"), Tags: tags}
	tagSpecificationRequest := []*ec2.LaunchTemplateTagSpecificationRequest{&tsr}

	templateRequest := &ec2.RequestLaunchTemplateData{
		IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileSpecificationRequest{
			Name: aws.String(configOptions.IamInstanceProfile),
		},
		ImageId:           aws.String(configOptions.AmiID),
		InstanceType:      aws.String(configOptions.InstanceType),
		KeyName:           aws.String(configOptions.KeyName),
		SecurityGroupIds:  configOptions.SecurityGroups,
		UserData:          aws.String(configOptions.UserData),
		TagSpecifications: tagSpecificationRequest,
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
