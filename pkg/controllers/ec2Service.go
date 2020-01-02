package controllers

import (
	"encoding/base64"
	"log"
	"reflect"
	"strconv"

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

	names := []*string{&name}
	input := ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: names,
	}

	log.Println("Getting launch template: ", name)
	response, err := asgSvc.DescribeLaunchTemplates(&input)
	if err != nil {
		log.Panicln("Error while getting launch template", err)
	}

	if response.LaunchTemplates == nil {
		return nil
	}

	return response.LaunchTemplates[0]
}

//GetLaunchTemplateVersion represents
func (r *Ec2Service) GetLaunchTemplateVersion(name *string, version *string) *ec2.LaunchTemplateVersion {
	asgSvc := ec2.New(&r.AwsSession)

	versions := []*string{version}
	input := ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateName: name,
		Versions:           versions,
	}

	log.Printf("Getting launch template with version: '%v', version: '%v'", *name, *version)
	response, err := asgSvc.DescribeLaunchTemplateVersions(&input)
	if err != nil {
		log.Fatal("Error while getting launch template version", err)
	}

	if response.LaunchTemplateVersions == nil {
		return nil
	}

	return response.LaunchTemplateVersions[0]
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

	templateRequest := r.getLaunchTemplateDataRequest(configOptions)

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

//UpdateLaunchTemplate represents
func (r *Ec2Service) UpdateLaunchTemplate(configOptions *apiTypes.LaunchTemplateOptions) (*ec2.LaunchTemplate, error) {
	ec2Svc := ec2.New(&r.AwsSession)

	latestVersion, err := r.CreateLaunchTemplateVersion(configOptions)
	if err != nil {
		return nil, err
	}

	input := ec2.ModifyLaunchTemplateInput{
		LaunchTemplateName: aws.String(configOptions.Name),
		DefaultVersion:     &latestVersion,
	}

	output, err := ec2Svc.ModifyLaunchTemplate(&input)
	if err != nil {
		log.Println("Failed to update Launch Template version to the latest version:", latestVersion, err)
		return nil, err
	}

	return output.LaunchTemplate, err
}

//CompareLaunchTemplateData represents
func (r *Ec2Service) CompareLaunchTemplateData(new *apiTypes.LaunchTemplateOptions, current *ec2.ResponseLaunchTemplateData) (*apiTypes.LaunchTemplateOptions, bool) {
	changed := false
	if new.AmiID != *current.ImageId {
		log.Println("AMI has changed: ", new.AmiID, *current.ImageId)
		changed = true
	}

	if new.PublicIps != *current.NetworkInterfaces[0].AssociatePublicIpAddress {
		log.Println("Public Ips setting has changed: ", new.PublicIps, *current.NetworkInterfaces[0].AssociatePublicIpAddress)
		changed = true
	}

	currentTags := make(map[string]string)
	for _, v := range current.TagSpecifications[0].Tags {
		currentTags[*v.Key] = *v.Value
	}

	if !reflect.DeepEqual(new.Tags, currentTags) {
		log.Println("Tags have changed.")
		changed = true
	}

	if new.IamInstanceProfile != *current.IamInstanceProfile.Name {
		log.Println("IamInstanceProfile has changed.")
		changed = true
	}

	if new.InstanceType != *current.InstanceType {
		log.Println("InstanceType has changed.")
		changed = true
	}

	cUserData, _ := base64.StdEncoding.DecodeString(*current.UserData)
	if new.UserData != string(cUserData) {
		log.Println("UserData has changed.")
		changed = true
	}

	if new.KeyName != *current.KeyName {
		log.Println("Key has changed.")
		changed = true
	}

	return new, changed
}

func (r *Ec2Service) getLaunchTemplateDataRequest(configOptions *apiTypes.LaunchTemplateOptions) *ec2.RequestLaunchTemplateData {
	tags := r.getEc2Tags(configOptions.Tags)

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
			VolumeSize: aws.Int64(configOptions.VolumeSize),
			VolumeType: aws.String(configOptions.VolumeType),
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

	return templateRequest
}

//CreateLaunchTemplateVersion represents
func (r *Ec2Service) CreateLaunchTemplateVersion(configOptions *apiTypes.LaunchTemplateOptions) (string, error) {
	ec2Svc := ec2.New(&r.AwsSession)
	templateRequest := r.getLaunchTemplateDataRequest(configOptions)

	launchTemplateVersionInput := ec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateName: aws.String(configOptions.Name),
		LaunchTemplateData: templateRequest,
	}

	var latestVersion string
	ltOutput, err := ec2Svc.CreateLaunchTemplateVersion(&launchTemplateVersionInput)
	if err != nil {
		log.Println("Failed to create Launch Template version.", err)
		return latestVersion, err
	}

	latestVersion = strconv.Itoa(int(*ltOutput.LaunchTemplateVersion.VersionNumber))
	log.Printf("Created Launch Template version: %v for %v", latestVersion, configOptions.Name)
	return latestVersion, nil
}

func (r *Ec2Service) getEc2Tags(tags map[string]string) []*ec2.Tag {
	ec2Tags := []*ec2.Tag{}
	for i, v := range tags {
		t := ec2.Tag{Key: aws.String(i), Value: aws.String(v)}

		ec2Tags = append(ec2Tags, &t)
	}

	return ec2Tags
}
