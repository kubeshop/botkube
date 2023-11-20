package graphql

// Usage describes organization usage statistics.
type Usage struct {
	DeploymentCount    *int `json:"deploymentCount"`
	MemberCount        *int `json:"memberCount"`
	NodeCount          *int `json:"nodeCount"`
	CloudSlackUseCount *int `json:"cloudSlackUseCount"`

	// All internal, ignored fields are removed
}
