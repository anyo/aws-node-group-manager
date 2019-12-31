package apis

// SsmRecommendedEksAmiValue represets the json response from ssm response
type SsmRecommendedEksAmiValue struct {
	ImageName     string `json:"image_name"`
	ImageID       string `json:"image_id"`
	Schemaversion string `json:"schema_version"`
}

// SsmRecommendedEksAmi represets the json response from ssm response
type SsmRecommendedEksAmi struct {
	SsmRecommendedEksAmiValue
	Name string
	ARN  string
}
