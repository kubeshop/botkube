package graphql

// OrganizationConnectedPlatforms represents connected platforms.
type OrganizationConnectedPlatforms struct {
	Slacks []*SlackWorkspace `json:"slacks"`
	Slack  *SlackWorkspace   `json:"slack"`

	TeamsOrganizations []*TeamsOrganization `json:"teamsOrganizations"`
	TeamsOrganization  *TeamsOrganization   `json:"teamsOrganization"`
}

type TeamsOrganization struct {
	ID                     string `json:"id"`
	TenantID               string `json:"tenantId"`
	ConsentGiven           bool   `json:"consentGiven"`
	IsReConsentingRequired bool   `json:"isReConsentingRequired"`
}

type TeamsOrganizationTeam struct {
	ID                    string `json:"id"`
	AADGroupID            string `json:"aadGroupId"`
	DefaultConversationID string `json:"defaultConversationId"`
}
