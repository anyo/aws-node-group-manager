package main

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	controllers "github.com/anyo/aws-node-group-manager/pkg/controllers"
)

var region string
var k8sVersion string
var alwaysLatestAmi bool
var asgDesiredCount int
var asgMinCount int
var asgMaxCount int

func main() {
	region = "us-east-1"
	k8sVersion = "1.14"
	alwaysLatestAmi = false

	session := getSession()

	ssmSvc := controllers.SsmService{
		AwsSession: session,
		Region:     region,
	}

	ami, err := ssmSvc.GetEksOptimizedAmi(k8sVersion)
	if err != nil {
		log.Panicln(err)
	}

	log.Println(ami)

	asgSvc := controllers.AsgService{
		AwsSession: session,
		Region:     region,
	}

	asgSvc.GetAutoScaingGroups()
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
