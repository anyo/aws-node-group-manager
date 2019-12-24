package controllers

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"

	apiTypes "github.com/anyo/aws-node-group-manager/pkg/apis"
)

//SsmService represents ssm operations
type SsmService struct {
	AwsSession session.Session
	Region     string
}

//GetEksOptimizedAmi represents getting the latest recomended Ami for Eks in this region
func (r *SsmService) GetEksOptimizedAmi(k8sVersion string) (apiTypes.SsmRecommendedEksAmi, error) {
	response := apiTypes.SsmRecommendedEksAmi{}

	ssmSvc := ssm.New(&r.AwsSession)

	paramName := "/aws/service/eks/optimized-ami/" + k8sVersion + "/amazon-linux-2/recommended"
	input := ssm.GetParameterInput{
		Name: &paramName,
	}
	param, err := ssmSvc.GetParameter(&input)

	if err != nil {
		log.Fatal("Error while getting session", err)
		return response, err
	}

	var recommended apiTypes.SsmRecommendedEksAmiValue
	err = json.Unmarshal([]byte(*param.Parameter.Value), &recommended)
	if err != nil {
		log.Fatal("Failed to unmarshal the ami response", err)
		return response, err
	}

	response = apiTypes.SsmRecommendedEksAmi{
		SsmRecommendedEksAmiValue: recommended,
		Name:                      *param.Parameter.Name,
		ARN:                       *param.Parameter.ARN,
	}

	return response, nil
}
