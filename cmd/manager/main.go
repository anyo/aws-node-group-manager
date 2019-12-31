package main

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"encoding/base64"

	apiTypes "github.com/anyo/aws-node-group-manager/pkg/apis"
	controllers "github.com/anyo/aws-node-group-manager/pkg/controllers"
)

var region string
var k8sVersion string
var alwaysLatestAmi bool

func main() {
	region = "us-east-1"
	k8sVersion = "1.14"
	alwaysLatestAmi = false

	session := getSession()
	ssmSvc := controllers.SsmService{AwsSession: session, Region: region}

	ami, err := ssmSvc.GetEksOptimizedAmi(k8sVersion)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("AWS AMI: ", ami.ImageName)

	asgSvc := controllers.AsgService{AwsSession: session, Region: region}
	ec2Svc := controllers.Ec2Service{AwsSession: session, Region: region}

	// success := createLaunchConfig(asgSvc, ami)
	// log.Println("Launch Configuration created: ", success)

	template, success := createLaunchTemplate(ec2Svc, ami)
	log.Println("Launch Template created: ", template.LaunchTemplateName)

	success = createAutoScalingGroup(asgSvc, *template.LaunchTemplateName)
	log.Println("ASG created: ", success)

	asg := asgSvc.GetAutoScalingGroup("OperatorGenerated-GeneralPurpose")
	for {
		if *asg.DesiredCapacity == int64(len(asg.Instances)) {
			log.Printf("Desired Capacity matched Current instances: %v == %v \n", *asg.DesiredCapacity, len(asg.Instances))
			break
		} else {
			log.Printf("Waiting for Desired Capacity matched Current instances: %v == %v ... \n", *asg.DesiredCapacity, len(asg.Instances))
			time.Sleep(2 * time.Second)
			asg = asgSvc.GetAutoScalingGroup("OperatorGenerated-GeneralPurpose")
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
			asg = asgSvc.GetAutoScalingGroup("OperatorGenerated-GeneralPurpose")
			log.Println("Awaiting all instances to be healthy...")
			time.Sleep(5 * time.Second)
		}
	}
}

func createAutoScalingGroup(asgSvc controllers.AsgService, templateName string) bool {
	/////////////////////////////////////////////////////////////////
	///////// This information will come from the CRD yaml //////////
	/////////////////////////////////////////////////////////////////
	name := "OperatorGenerated-GeneralPurpose"
	subnets := "subnet-0fb3f183f38ba186f,subnet-0811b038c2d9a27ef,subnet-07f40d97bcba2e399"
	desiredInstances := int64(3)
	minInstances := int64(1)
	maxInstances := int64(5)
	//lcName := lcr
	tags := map[string]string{
		"CreatedBy": "ASG-OPERATOR",
	}
	/////////////////////////////////////////////////////////////////
	/////////////////////////////////////////////////////////////////
	/////////////////////////////////////////////////////////////////

	asgInstance := apiTypes.AutoScalingGroupOptions{
		Name:               name,
		Subnets:            subnets,
		DesiredInstances:   desiredInstances,
		MinInstances:       minInstances,
		MaxInstances:       maxInstances,
		LaunchConfName:     name,
		LaunchTemplateName: templateName,
		Tags:               tags,
	}

	_, asgErr := asgSvc.CreateAsg(&asgInstance)
	if asgErr != nil {
		log.Panicln("Failed to create asg", asgErr)
		return false
	}

	return true
}

func createLaunchConfig(asgSvc controllers.AsgService, ami apiTypes.SsmRecommendedEksAmi) bool {
	/////////////////////////////////////////////////////////////////
	///////// This information will come from the CRD yaml //////////
	/////////////////////////////////////////////////////////////////
	namePrefix := "OperatorGenerated-GeneralPurpose"
	publicIps := true
	instanceType := "t2.medium"
	keyName := "talhaverse"

	sg := "sg-03d8fd9741d919892"
	securityGroups := []*string{&sg}

	userData := base64.StdEncoding.EncodeToString([]byte(`
	yum upgrade -y
	yum install bind-utils -y
	yum instamm git -y
	`))
	iamInstanceProfile := "talhan-ec2-admin"
	ebsSize := int64(50)
	/////////////////////////////////////////////////////////////////
	/////////////////////////////////////////////////////////////////
	/////////////////////////////////////////////////////////////////

	launchConfiguration := asgSvc.GetLaunchConfiguration(namePrefix)
	if launchConfiguration != nil {
		log.Println("Launch configuration already exists: ", *launchConfiguration.LaunchConfigurationName)
		return true
	}

	lcInstance := apiTypes.LaunchConfigurationOptions{
		NamePrefix:         namePrefix,
		AmiID:              ami.ImageID,
		PublicIps:          publicIps,
		InstanceType:       instanceType,
		KeyName:            keyName,
		SecurityGroups:     securityGroups,
		UserData:           userData,
		IamInstanceProfile: iamInstanceProfile,
		EbsVolumeSize:      ebsSize,
	}

	_, lcErr := asgSvc.CreateAsgLaunchConfig(&lcInstance)
	if lcErr != nil {
		log.Panicln("Failed to create launch config", lcErr)
		return false
	}

	return true
}

func createLaunchTemplate(ec2Svc controllers.Ec2Service, ami apiTypes.SsmRecommendedEksAmi) (*ec2.LaunchTemplate, bool) {
	/////////////////////////////////////////////////////////////////
	///////// This information will come from the CRD yaml //////////
	/////////////////////////////////////////////////////////////////
	name := "OperatorGenerated-GeneralPurpose"
	publicIps := true
	instanceType := "t2.medium"
	keyName := "talhaverse"

	sg := "sg-03d8fd9741d919892"
	securityGroups := []*string{&sg}

	userData := base64.StdEncoding.EncodeToString([]byte(`
	yum upgrade -y
	yum install bind-utils -y
	yum instamm git -y
	`))
	iamInstanceProfile := "talhan-ec2-admin"
	ebsSize := int64(50)
	/////////////////////////////////////////////////////////////////
	/////////////////////////////////////////////////////////////////
	/////////////////////////////////////////////////////////////////

	launchTemplate := ec2Svc.GetLaunchTemplate(name)
	if launchTemplate != nil {
		log.Println("Launch configuration already exists: ", *launchTemplate.LaunchTemplateName)
		return launchTemplate, false
	}

	tags := map[string]string{
		"CreatedBy": "ASG-OPERATOR",
		"Name":      "ASG-OPERATOR-WORKER",
	}

	launchTemplateInput := apiTypes.LaunchTemplateOptions{
		Name:               name,
		AmiID:              ami.ImageID,
		PublicIps:          publicIps,
		InstanceType:       instanceType,
		KeyName:            keyName,
		SecurityGroups:     securityGroups,
		UserData:           userData,
		IamInstanceProfile: iamInstanceProfile,
		EbsVolumeSize:      ebsSize,
		Tags:               tags,
	}

	template, err := ec2Svc.CreateLaunchTemplate(&launchTemplateInput)
	if err != nil {
		log.Panicln("Failed to create launch config", err)
		return nil, true
	}

	return template, false
}

func getSession() session.Session {
	session, err := session.NewSessionWithOptions(session.Options{
		Profile: "argentus",
		Config:  aws.Config{Region: aws.String(region)},
	})

	if err != nil {
		log.Fatal("Error while getting session", err)
		os.Exit(1)
	}

	return *session
}
