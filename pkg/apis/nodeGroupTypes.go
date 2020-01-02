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
	Name               string            `yaml:"name"`
	AmiID              string            `yaml:"-"`
	PublicIps          bool              `yaml:"publicIps"`
	InstanceType       string            `yaml:"instanceType"`
	KeyName            string            `yaml:"keyName"`
	SecurityGroups     []*string         `yaml:"securityGroups"`
	UserData           string            `yaml:"userData"`
	IamInstanceProfile string            `yaml:"iamInstanceProfile"`
	Tags               map[string]string `yaml:"tags"`
	EbsVolume          `yaml:"ebs"`
}

// AutoScalingGroupOptions represents all the fields to create a AutoScalingGroup config
type AutoScalingGroupOptions struct {
	Name               string            `yaml:"name"`
	Subnets            string            `yaml:"subnets"`
	DesiredInstances   int64             `yaml:"desired"`
	MaxInstances       int64             `yaml:"max"`
	MinInstances       int64             `yaml:"min"`
	LaunchConfName     string            `yaml:"-"`
	LaunchTemplateName string            `yaml:"-"`
	Tags               map[string]string `yaml:"tags"`
}

// EbsVolume represents
type EbsVolume struct {
	VolumeType string `yaml:"volumeType"`
	VolumeSize int64  `yaml:"volumeSize"`
}

// Ec2Options represents
type Ec2Options struct {
	LaunchTemplateOptions `yaml:"launchTemplate"`
	NamePrefix            string `yaml:"string"`
}

// SSMOptions represents
type SSMOptions struct {
	AutoUpgradeAmiChange bool `yaml:"autoAmiUpgrade"`
}

// OperatorModel represents
type OperatorModel struct {
	Ec2Options              `yaml:"ec2"`
	AutoScalingGroupOptions `yaml:"asg"`
	SSMOptions              `yaml:"ssm"`
}
