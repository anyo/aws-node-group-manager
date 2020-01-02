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
func (r *ReconcilerService) ReconcileLaunchTemplate(newLaunchTemplate *apiTypes.LaunchTemplateOptions) (*ec2.LaunchTemplate, bool) {
	newLaunchTemplate.Name = "OperatorGenerated-" + newLaunchTemplate.Name
	newLaunchTemplate.UserData = base64.StdEncoding.EncodeToString([]byte(newLaunchTemplate.UserData))

	launchTemplate := r.Ec2Service.GetLaunchTemplate(newLaunchTemplate.Name)

	if launchTemplate != nil {
		versionStr := strconv.Itoa(int(*launchTemplate.LatestVersionNumber))
		v := r.Ec2Service.GetLaunchTemplateVersion(launchTemplate.LaunchTemplateName, &versionStr)

		newLaunchTemplate, changed := r.Ec2Service.CompareLaunchTemplateData(newLaunchTemplate, v.LaunchTemplateData)

		// update the launch template since its changed compared to the current latest version
		if changed {
			updated, success := r.updateLaunchTemplate(v, newLaunchTemplate)
			return updated, success
		}

		log.Println("Launch template already exists and has not changed: ", *launchTemplate.LaunchTemplateName)
		return launchTemplate, true
	}

	template, err := r.Ec2Service.CreateLaunchTemplate(newLaunchTemplate)
	if err != nil {
		log.Println("Failed to create launch template", err)
		return nil, false
	}

	return template, true
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
func (r *ReconcilerService) ReconcileAutoScalingGroup(asgInstance *apiTypes.AutoScalingGroupOptions, templateName string, templateVersion int64) (*autoscaling.Group, bool) {
	asgInstance.Name = "OperatorGenerated-" + asgInstance.Name
	asgInstance.LaunchTemplateName = templateName

	asg := r.AsgService.GetAutoScalingGroup(asgInstance.Name)

	if asg != nil {
		log.Println("Asg already exists: ", *asg.AutoScalingGroupName)

		changed, err := r.AsgService.CompareAsg(asgInstance, asg)
		if err != nil {
			log.Println("Failed to check if ASG has changed.", err, asg.AutoScalingGroupName)
			return nil, true
		}

		if changed {
			log.Println("ASG has changed: ", asg.AutoScalingGroupName)
			_, err := r.AsgService.UpdateAsg(asgInstance)

			if err != nil {
				log.Println("Failed to update ASG.", err, asg.AutoScalingGroupName)
				return asg, true
			}

			asg = r.AsgService.GetAutoScalingGroup(asgInstance.Name)
			log.Println("ASG updated: ", asg.AutoScalingGroupName)

			// check if the changes has been applied
			r.AsgStatusMonitor()

			return asg, false
		}

		// check launch template version number for all instances is insync, if not, detach
		staleInstances := make([]*autoscaling.Instance, 0)
		log.Println("Total instances in the asg: ", len(asg.Instances))
		for _, v := range asg.Instances {
			if *v.LaunchTemplate.Version != string(templateVersion) {
				staleInstances = append(staleInstances, v)
			}
		}

		log.Println("Stale Instances: ", len(staleInstances))

		return asg, false
	}

	_, asgErr := r.AsgService.CreateAsg(asgInstance)
	if asgErr != nil {
		log.Panicln("Failed to create asg", asgErr)
		return nil, false
	}

	return nil, true
}

//AsgStatusMonitor represents
func (r *ReconcilerService) AsgStatusMonitor() {
	asg := r.AsgService.GetAutoScalingGroup("OperatorGenerated-GeneralPurpose")
	for {
		if *asg.DesiredCapacity == int64(len(asg.Instances)) {
			log.Printf("Desired Capacity matched Current instances: %v == %v \n", *asg.DesiredCapacity, len(asg.Instances))
			break
		} else {
			log.Printf("Waiting for Desired Capacity matched Current instances: %v == %v ... \n", *asg.DesiredCapacity, len(asg.Instances))
			time.Sleep(2 * time.Second)
			asg = r.AsgService.GetAutoScalingGroup("OperatorGenerated-GeneralPurpose")
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
			asg = r.AsgService.GetAutoScalingGroup("OperatorGenerated-GeneralPurpose")
			log.Println("Awaiting all instances to be healthy...")
			time.Sleep(5 * time.Second)
		}
	}
}

//GetLatestEksAmi represents
func (r *ReconcilerService) GetLatestEksAmi(k8sVersion *string) string {
	ami, err := r.SsmService.GetEksOptimizedAmi(*k8sVersion)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("AWS AMI: ", ami.ImageName, ami.ImageID)
	return ami.ImageID
}
