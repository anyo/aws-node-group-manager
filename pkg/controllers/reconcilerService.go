package controllers

import (
	"encoding/base64"
	"log"
	"strconv"
	"time"

	apiTypes "github.com/anyo/aws-node-group-manager/pkg/apis"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

//ReconcilerService represents ssm operations
type ReconcilerService struct {
	AsgService
	Ec2Service
	SsmService
}

//ReconcileLaunchTemplate represents
func (r *ReconcilerService) ReconcileLaunchTemplate(newLaunchTemplate *apiTypes.LaunchTemplateOptions) (*string, *string, bool) {
	newLaunchTemplate.Name = "OperatorGenerated-" + newLaunchTemplate.Name
	newLaunchTemplate.UserData = base64.StdEncoding.EncodeToString([]byte(newLaunchTemplate.UserData))

	launchTemplate := r.Ec2Service.GetLaunchTemplate(newLaunchTemplate.Name)
	var versionStr string

	if launchTemplate != nil {
		versionStr = strconv.Itoa(int(*launchTemplate.LatestVersionNumber))

		v := r.Ec2Service.GetLaunchTemplateVersion(launchTemplate.LaunchTemplateName, &versionStr)

		newLaunchTemplate, changed := r.Ec2Service.CompareLaunchTemplateData(newLaunchTemplate, v.LaunchTemplateData)

		// update the launch template since its changed compared to the current latest version
		if changed {
			updated, success := r.updateLaunchTemplate(v, newLaunchTemplate)
			return updated.LaunchTemplateName, &versionStr, success
		}

		log.Println("Launch template already exists and has not changed: ", *launchTemplate.LaunchTemplateName)
		return launchTemplate.LaunchTemplateName, &versionStr, true
	}

	template, err := r.Ec2Service.CreateLaunchTemplate(newLaunchTemplate)
	if err != nil {
		log.Println("Failed to create launch template", err)
		return nil, &versionStr, false
	}

	log.Println("Launch template successfully created", template.LaunchTemplateName, &versionStr)
	return template.LaunchTemplateName, &versionStr, true
}

func (r *ReconcilerService) updateLaunchTemplate(launchTemplateVersion *ec2.LaunchTemplateVersion, newLaunchTemplate *apiTypes.LaunchTemplateOptions) (*ec2.LaunchTemplate, bool) {
	// ensure name and ebs volume does not change
	newLaunchTemplate.Name = *launchTemplateVersion.LaunchTemplateName
	newLaunchTemplate.EbsVolume.VolumeType = *launchTemplateVersion.LaunchTemplateData.BlockDeviceMappings[0].Ebs.VolumeType
	newLaunchTemplate.EbsVolume.VolumeSize = *launchTemplateVersion.LaunchTemplateData.BlockDeviceMappings[0].Ebs.VolumeSize
	newLaunchTemplate.SecurityGroups = launchTemplateVersion.LaunchTemplateData.SecurityGroupIds

	//encode userdata
	newLaunchTemplate.UserData = base64.StdEncoding.EncodeToString([]byte(newLaunchTemplate.UserData))

	log.Println("Launch template has changed: ", newLaunchTemplate.Name)
	updated, err := r.Ec2Service.UpdateLaunchTemplate(newLaunchTemplate)

	if err != nil {
		log.Println("Failed to update launch template.", err, newLaunchTemplate.Name)
		return updated, false
	}

	log.Printf("Launch template: %v has been update to version: %v. \n", *updated.LaunchTemplateName, *updated.LatestVersionNumber)
	return updated, true
}

//ReconcileAutoScalingGroup represents
func (r *ReconcilerService) ReconcileAutoScalingGroup(asgInstance *apiTypes.AutoScalingGroupOptions, templateName *string, templateVersion *string) (*autoscaling.Group, bool) {
	asgInstance.Name = "OperatorGenerated-" + asgInstance.Name
	asgInstance.LaunchTemplateName = *templateName

	asg := r.AsgService.GetAutoScalingGroup(asgInstance.Name)

	if asg != nil {
		log.Println("Asg already exists: ", *asg.AutoScalingGroupName)

		changed, err := r.AsgService.CompareAsg(asgInstance, asg)
		if err != nil {
			log.Println("Failed to check if ASG has changed.", err, asg.AutoScalingGroupName)
			return nil, false
		}

		if changed {
			log.Println("ASG has changed: ", *asg.AutoScalingGroupName)
			_, err := r.AsgService.UpdateAsg(asgInstance)

			if err != nil {
				log.Println("Failed to update ASG.", err, asg.AutoScalingGroupName)
				return asg, false
			}

			asg = r.AsgService.GetAutoScalingGroup(asgInstance.Name)
			log.Println("ASG updated: ", *asg.AutoScalingGroupName)

			// check if the changes has been applied
			r.AsgStatusMonitor(asg.AutoScalingGroupName)

			return asg, true
		}

		// check launch template version number for all instances is insync, if not, detach
		staleInstances := make([]*autoscaling.Instance, 0)
		log.Println("Total instances in the asg: ", len(asg.Instances))
		for _, v := range asg.Instances {
			if *v.LaunchTemplate.Version != *templateVersion {
				log.Printf("Stale instance: '%v', required-'%v' vs current-'%v'", *v.InstanceId, templateVersion, *v.LaunchTemplate.Version)
				staleInstances = append(staleInstances, v)
			}
		}

		if len(staleInstances) > 0 {
			log.Println("Stale Instances found in the ASG: ", *asg.AutoScalingGroupName, len(staleInstances))
			for _, v := range staleInstances {
				detached := r.AsgService.DetachInstance(v.InstanceId, asg.AutoScalingGroupName)
				if detached {
					r.AsgStatusMonitor(asg.AutoScalingGroupName)

					_ = r.Ec2Service.ShutDownInstance(v.InstanceId)
					_ = r.Ec2Service.TerminateInstance(v.InstanceId)
				}
			}
		} else {
			log.Printf("Stale Instances found in the ASG: '%v' - %v ", *asg.AutoScalingGroupName, len(staleInstances))
		}

		return asg, true
	}

	log.Println("Asg does not exist: ", asgInstance.Name)
	_, asgErr := r.AsgService.CreateAsg(asgInstance)
	if asgErr != nil {
		log.Println("Failed to create asg", asgErr)
		return nil, false
	}

	r.AsgStatusMonitor(&asgInstance.Name)

	return nil, true
}

//AsgStatusMonitor represents
func (r *ReconcilerService) AsgStatusMonitor(asgName *string) {
	asg := r.AsgService.GetAutoScalingGroup(*asgName)
	for {
		if asg == nil {
			log.Println("Awaiting ASG to come up")
			time.Sleep(time.Second * 10)
			asg = r.AsgService.GetAutoScalingGroup(*asgName)
		} else {
			break
		}
	}

	if asg == nil {
		return
	}

	for {
		if *asg.DesiredCapacity == int64(len(asg.Instances)) {
			log.Printf("Desired Capacity matched Current instances: Desired - %v == Current - %v \n", *asg.DesiredCapacity, len(asg.Instances))
			break
		} else {
			log.Printf("Waiting for Desired Capacity matched Current instances: Desired - %v == Current - %v ... \n", *asg.DesiredCapacity, len(asg.Instances))
			time.Sleep(2 * time.Second)
			asg = r.AsgService.GetAutoScalingGroup(*asgName)
		}
	}

	for {
		completed := true
		for _, v := range asg.Instances {
			if *v.HealthStatus != "Healthy" || *v.LifecycleState != "InService" {
				completed = false
			}
		}

		if completed {
			log.Println("All done.")
			break
		} else {
			asg = r.AsgService.GetAutoScalingGroup(*asgName)
			log.Println("Awaiting all instances to be healthy...")
			time.Sleep(5 * time.Second)
		}
	}
}

//GetLatestEksAmi represents
func (r *ReconcilerService) GetLatestEksAmi(k8sVersion *string) *string {
	ami, err := r.SsmService.GetEksOptimizedAmi(*k8sVersion)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Println("AWS AMI: ", ami.ImageName, ami.ImageID)
	return &ami.ImageID
}
