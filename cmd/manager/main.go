package main

import (
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	apiTypes "github.com/anyo/aws-node-group-manager/pkg/apis"
	controllers "github.com/anyo/aws-node-group-manager/pkg/controllers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var region string
var k8sVersion string
var alwaysLatestAmi bool

func main() {
	region = "us-east-1"
	k8sVersion = "1.14"

	session := getAwsSession(region)
	ssmSvc := controllers.SsmService{AwsSession: session, Region: region}
	asgSvc := controllers.AsgService{AwsSession: session, Region: region}
	ec2Svc := controllers.Ec2Service{AwsSession: session, Region: region}

	reconcilerSvc := controllers.ReconcilerService{
		AsgService: asgSvc,
		SsmService: ssmSvc,
		Ec2Service: ec2Svc,
	}

	c := loadConfig()
	c.AmiID = reconcilerSvc.GetLatestEksAmi(&k8sVersion)

	templateName, latestVersion, success := reconcilerSvc.ReconcileLaunchTemplate(&c.LaunchTemplateOptions)

	if !success {
		os.Exit(1)
	}

	_, success = reconcilerSvc.ReconcileAutoScalingGroup(&c.AutoScalingGroupOptions, templateName, latestVersion)
	if !success {
		os.Exit(1)
	}

	os.Exit(0)
}

//GetAwsSession represents
func getAwsSession(region string) session.Session {
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

func loadConfig() apiTypes.OperatorModel {
	dir, err := os.Getwd()
	//filePath := dir + "/cmd/manager/config.yaml"
	filePath := "config.yaml"
	config, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err, dir)
		os.Exit(1)
	}
	c := apiTypes.OperatorModel{}
	err = yaml.Unmarshal(config, &c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
	return c
}
