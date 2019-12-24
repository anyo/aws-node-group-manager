package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"

	ssmTypes "github.com/anyo/aws-node-group-manager/pkg/apis"
)

var region string
var k8sVersion string

func main() {
	region = "us-east-1"
	k8sVersion = "1.14"
	session := getSession()

	ami := getEksOptimizedAmi(&session, region, k8sVersion)

	log.Println(ami)
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

func getEksOptimizedAmi(session *session.Session, region string, k8sVersion string) ssmTypes.SsmRecommendedEksAmi {
	ssmSvc := ssm.New(session)

	paramName := "/aws/service/eks/optimized-ami/1.14/amazon-linux-2/recommended"
	input := ssm.GetParameterInput{
		Name: &paramName,
	}
	param, err := ssmSvc.GetParameter(&input)

	if err != nil {
		log.Fatal("Error while getting session", err)
		os.Exit(1)
	}

	var recommended ssmTypes.SsmRecommendedEksAmiValue
	err = json.Unmarshal([]byte(*param.Parameter.Value), &recommended)
	if err != nil {
		log.Fatal("Failed to unmarshal the ami response", err)
		os.Exit(1)
	}

	response := ssmTypes.SsmRecommendedEksAmi{
		SsmRecommendedEksAmiValue: recommended,
		Name:                      *param.Parameter.Name,
		ARN:                       *param.Parameter.ARN,
	}

	return response
}
