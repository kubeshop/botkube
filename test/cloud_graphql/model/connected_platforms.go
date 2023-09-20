package model

// OrganizationConnectedPlatforms represents connected platforms.
type OrganizationConnectedPlatforms struct {
	OrganizationID string            `graphql:"-"`
	Slacks         []*SlackWorkspace `json:"slacks"`
	Slack          *SlackWorkspace   `json:"slack"`
}
