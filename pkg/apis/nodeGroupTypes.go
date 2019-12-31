package apis

// LaunchConfigurationOptions represents all the fields to create a Launch config
type LaunchConfigurationOptions struct {
	NamePrefix         string
	AmiID              string
	PublicIps          bool
	InstanceType       string
	KeyName            string
	SecurityGroups     []*string
	UserData           string
	IamInstanceProfile string
	EbsVolumeSize      int64
}

// LaunchTemplateOptions represents all the fields to create a Launch config
type LaunchTemplateOptions struct {
	Name               string
	AmiID              string
	PublicIps          bool
	InstanceType       string
	KeyName            string
	SecurityGroups     []*string
	UserData           string
	IamInstanceProfile string
	EbsVolumeSize      int64
	Tags               map[string]string
}

// AutoScalingGroupOptions represents all the fields to create a AutoScalingGroup config
type AutoScalingGroupOptions struct {
	Name               string
	Subnets            string //csv list
	DesiredInstances   int64
	MaxInstances       int64
	MinInstances       int64
	LaunchConfName     string
	LaunchTemplateName string
	Tags               map[string]string
}
