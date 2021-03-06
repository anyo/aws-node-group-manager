package controllers

import (
	"log"
	"reflect"
	"time"

	apiTypes "github.com/anyo/aws-node-group-manager/pkg/apis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

//AsgService represents ssm operations
type AsgService struct {
	AwsSession session.Session
	Region     string
}

//GetAutoScalingGroups represents
func (r *AsgService) GetAutoScalingGroups() []*autoscaling.Group {
	asgSvc := autoscaling.New(&r.AwsSession)

	input := autoscaling.DescribeAutoScalingGroupsInput{}
	grps, err := asgSvc.DescribeAutoScalingGroups(&input)
	if err != nil {
		log.Println("Error while getting asgs", err)
		return nil
	}

	return grps.AutoScalingGroups
}

//GetLaunchConfiguration represents
func (r *AsgService) GetLaunchConfiguration(name string) *autoscaling.LaunchConfiguration {
	asgSvc := autoscaling.New(&r.AwsSession)

	input := autoscaling.DescribeLaunchConfigurationsInput{}
	response, err := asgSvc.DescribeLaunchConfigurations(&input)
	if err != nil {
		log.Println("Error while getting asgs", err)
		return nil
	}

	if response.LaunchConfigurations == nil {
		return nil
	}

	return response.LaunchConfigurations[0]
}

//GetAutoScalingGroup represents
func (r *AsgService) GetAutoScalingGroup(name string) *autoscaling.Group {
	asgSvc := autoscaling.New(&r.AwsSession)

	names := []*string{aws.String(name)}
	maxRecords := int64(1)
	input := autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: names,
		MaxRecords:            &maxRecords,
	}

	response, err := asgSvc.DescribeAutoScalingGroups(&input)
	if err != nil {
		log.Println("Error while getting asg: ", name, ", Error: ", err)
		return nil
	}

	if response.AutoScalingGroups == nil {
		return nil
	}

	return response.AutoScalingGroups[0]
}

//CreateAsgLaunchConfig represents
func (r *AsgService) CreateAsgLaunchConfig(configOptions *apiTypes.LaunchConfigurationOptions) (*autoscaling.CreateLaunchConfigurationOutput, error) {

	asgSvc := autoscaling.New(&r.AwsSession)
	bdm := []*autoscaling.BlockDeviceMapping{}
	deviceMapping := autoscaling.BlockDeviceMapping{
		DeviceName: aws.String("/dev/sda2"),
		Ebs: &autoscaling.Ebs{
			VolumeSize: aws.Int64(configOptions.EbsVolumeSize),
		},
	}

	bdm = append(bdm, &deviceMapping)

	launchConfInput := autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName:  aws.String(configOptions.NamePrefix),
		AssociatePublicIpAddress: aws.Bool(configOptions.PublicIps),
		// BlockDeviceMappings:      bdm,
		ImageId:            aws.String(configOptions.AmiID),
		InstanceType:       aws.String(configOptions.InstanceType),
		KeyName:            aws.String(configOptions.KeyName),
		SecurityGroups:     configOptions.SecurityGroups,
		UserData:           aws.String(configOptions.UserData),
		IamInstanceProfile: aws.String(configOptions.IamInstanceProfile),
	}

	output, err := asgSvc.CreateLaunchConfiguration(&launchConfInput)

	return output, err
}

//CreateAsg represents
func (r *AsgService) CreateAsg(asgOptions *apiTypes.AutoScalingGroupOptions) (*autoscaling.CreateAutoScalingGroupOutput, error) {
	asgSvc := autoscaling.New(&r.AwsSession)

	tags := []*autoscaling.Tag{}

	for i, v := range asgOptions.Tags {
		t := autoscaling.Tag{
			Key:               aws.String(i),
			PropagateAtLaunch: aws.Bool(true),
			ResourceId:        aws.String(asgOptions.Name),
			ResourceType:      aws.String("auto-scaling-group"),
			Value:             aws.String(v),
		}

		tags = append(tags, &t)
	}

	launchTemplateSpecification := autoscaling.LaunchTemplateSpecification{
		LaunchTemplateName: aws.String(asgOptions.LaunchTemplateName),
		Version:            aws.String("$Latest"),
	}

	input := autoscaling.CreateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgOptions.Name),
		VPCZoneIdentifier:    aws.String(asgOptions.Subnets),
		DesiredCapacity:      aws.Int64(asgOptions.DesiredInstances),
		MinSize:              aws.Int64(asgOptions.MinInstances),
		MaxSize:              aws.Int64(asgOptions.MaxInstances),
		Tags:                 tags,
		LaunchTemplate:       &launchTemplateSpecification,
	}

	output, err := asgSvc.CreateAutoScalingGroup(&input)

	return output, err
}

//UpdateAsg represents
func (r *AsgService) UpdateAsg(asgOptions *apiTypes.AutoScalingGroupOptions) (*autoscaling.UpdateAutoScalingGroupOutput, error) {
	asgSvc := autoscaling.New(&r.AwsSession)
	tags := []*autoscaling.Tag{}

	for i, v := range asgOptions.Tags {
		t := autoscaling.Tag{
			Key:               aws.String(i),
			PropagateAtLaunch: aws.Bool(true),
			ResourceId:        aws.String(asgOptions.Name),
			ResourceType:      aws.String("auto-scaling-group"),
			Value:             aws.String(v),
		}

		tags = append(tags, &t)
	}

	input := autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(asgOptions.Name),
		DesiredCapacity:      aws.Int64(asgOptions.DesiredInstances),
		MinSize:              aws.Int64(asgOptions.MinInstances),
		MaxSize:              aws.Int64(asgOptions.MaxInstances),
	}

	tagsInput := autoscaling.CreateOrUpdateTagsInput{
		Tags: tags,
	}

	output, err := asgSvc.UpdateAutoScalingGroup(&input)
	if err != nil {
		log.Println("Error updating ASG:", asgOptions.Name)
		return nil, err
	}

	_, tagsErr := asgSvc.CreateOrUpdateTags(&tagsInput)
	if tagsErr != nil {
		log.Println("Error updating tags for ASG:", asgOptions.Name)
		return output, err
	}

	return output, err
}

//CompareAsg represents
func (r *AsgService) CompareAsg(new *apiTypes.AutoScalingGroupOptions, current *autoscaling.Group) (bool, error) {
	if new.DesiredInstances != *current.DesiredCapacity {
		return true, nil
	}

	if new.MaxInstances != *current.MaxSize {
		return true, nil
	}

	if new.MinInstances != *current.MinSize {
		return true, nil
	}

	currentTags := make(map[string]string)
	for _, v := range current.Tags {
		currentTags[*v.Key] = *v.Value
	}

	if !reflect.DeepEqual(new.Tags, currentTags) {
		return true, nil
	}

	return false, nil
}

// DetachInstance represents
func (r *AsgService) DetachInstance(instanceID *string, asgName *string) bool {
	asgSvc := autoscaling.New(&r.AwsSession)

	input := autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           asgName,
		InstanceIds:                    []*string{instanceID},
		ShouldDecrementDesiredCapacity: aws.Bool(false),
	}

	output, err := asgSvc.DetachInstances(&input)
	if err != nil {
		log.Printf("Failed to detach instance: %v, error: %v", *instanceID, err)
		return false
	}
	activity := output.Activities[0]

	for {
		if *activity.StatusCode != "Successful" {
			log.Printf("Detaching instance: '%v', Message: '%v'", *instanceID, *activity.Description)
			time.Sleep(2 * time.Second)
		} else if *activity.StatusCode == "Successful" {
			log.Printf("Detached instance: '%v', Progress: '%v', Message: %v", *instanceID, *output.Activities[0].Progress, *activity.Description)
			return true
		} else if *activity.StatusCode == "Failed" || *activity.StatusCode == "Cancelled" {
			log.Printf("Failed to detach instance: '%v', Progress: '%v', Cause: %v", *instanceID, *output.Activities[0].Progress, activity.StatusMessage)
			return false
		}

		activity = r.GetAutoScalingActivityStatus(output.Activities[0].ActivityId)
		if activity == nil {
			time.Sleep(2 * time.Second)
			continue
		}
	}
}

//GetAutoScalingActivityStatus represents
func (r *AsgService) GetAutoScalingActivityStatus(activityID *string) *autoscaling.Activity {
	asgSvc := autoscaling.New(&r.AwsSession)

	activityIds := []*string{activityID}
	input := autoscaling.DescribeScalingActivitiesInput{
		ActivityIds: activityIds,
	}

	output, err := asgSvc.DescribeScalingActivities(&input)
	if err != nil {
		log.Printf("Failed to get activity: %v, error: %v", activityID, err)
		return nil
	}

	return output.Activities[0]
}
