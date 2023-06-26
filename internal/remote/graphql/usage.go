package graphql

// Usage describes organization usage statistics.
type Usage struct {
	OrganizationID  string `graphql:"-"`
	DeploymentCount *int   `json:"deploymentCount"`
	MemberCount     *int   `json:"memberCount"`
	NodeCount       *int   `json:"nodeCount"`
}
