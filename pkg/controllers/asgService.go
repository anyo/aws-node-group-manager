package controllers

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

//AsgService represents ssm operations
type AsgService struct {
	AwsSession session.Session
	Region     string
}

//GetAutoScaingGroups reprensents
func (r *AsgService) GetAutoScaingGroups() {
	asgSvc := autoscaling.New(&r.AwsSession)

	input := autoscaling.DescribeAutoScalingGroupsInput{}
	grps, err := asgSvc.DescribeAutoScalingGroups(&input)
	if err != nil {
		log.Fatal("Error while getting asgs", err)
	}

	log.Println(grps)
}

//CreateAsgLaunchConfig represents
func (r *AsgService) CreateAsgLaunchConfig() {

}

//CreateAsg represents
func (r *AsgService) CreateAsg() {

}
