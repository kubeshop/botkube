package graphql

// Platforms is used by a specific platform field resolvers to
// return only those that are connected with a given deployment ID.
type Platforms struct {
	SocketSlacks    []*SocketSlack   `json:"socketSlacks"`
	CloudSlacks     []*CloudSlack    `json:"cloudSlacks"`
	Discords        []*Discord       `json:"discords"`
	Mattermosts     []*Mattermost    `json:"mattermosts"`
	Webhooks        []*Webhook       `json:"webhooks"`
	MsTeams         []*MsTeams       `json:"msTeams"`
	Elasticsearches []*Elasticsearch `json:"elasticsearches"`
	CloudMsTeams    []*CloudMsTeams  `json:"cloudTeams"`

	// All internal, ignored fields are removed
}
