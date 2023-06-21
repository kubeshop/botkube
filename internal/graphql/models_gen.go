// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graphql

import (
	"fmt"
	"io"
	"strconv"
)

type AuditEvent interface {
	IsAuditEvent()
	GetID() string
	GetType() *AuditEventType
	GetDeploymentID() string
	GetCreatedAt() string
	GetPluginName() string
	GetDeployment() *Deployment
}

type Pageable interface {
	IsPageable()
	GetPageInfo() *PageInfo
	GetTotalCount() int
}

type Action struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	DisplayName string          `json:"displayName"`
	Enabled     bool            `json:"enabled"`
	Command     string          `json:"command"`
	Bindings    *ActionBindings `json:"bindings"`
}

type ActionBindings struct {
	Sources   []string `json:"sources"`
	Executors []string `json:"executors"`
}

type ActionCreateUpdateInput struct {
	ID          *string                          `json:"id"`
	Name        string                           `json:"name"`
	DisplayName string                           `json:"displayName"`
	Enabled     bool                             `json:"enabled"`
	Command     string                           `json:"command"`
	Bindings    *ActionCreateUpdateInputBindings `json:"bindings"`
}

type ActionCreateUpdateInputBindings struct {
	Sources   []string `json:"sources"`
	Executors []string `json:"executors"`
}

type ActionPatchDeploymentConfigInput struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}

type AddMemberForOrganizationInput struct {
	OrgID     string  `json:"orgId"`
	UserID    *string `json:"userId"`
	UserEmail *string `json:"userEmail"`
}

type Alias struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Command     string            `json:"command"`
	Deployments []*DeploymentInfo `json:"deployments"`
}

type AliasCreateInput struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"displayName"`
	Command       string   `json:"command"`
	DeploymentIds []string `json:"deploymentIds"`
}

type AliasPage struct {
	Data       []*Alias  `json:"data"`
	PageInfo   *PageInfo `json:"pageInfo"`
	TotalCount int       `json:"totalCount"`
	TotalPages int       `json:"totalPages"`
}

func (AliasPage) IsPageable()                 {}
func (this AliasPage) GetPageInfo() *PageInfo { return this.PageInfo }
func (this AliasPage) GetTotalCount() int     { return this.TotalCount }

type AliasUpdateInput struct {
	Name          *string  `json:"name"`
	DisplayName   *string  `json:"displayName"`
	Command       *string  `json:"command"`
	DeploymentIds []string `json:"deploymentIds"`
}

type APIKey struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AuditEventCommandCreateInput struct {
	PlatformUser string       `json:"platformUser"`
	Channel      string       `json:"channel"`
	BotPlatform  *BotPlatform `json:"botPlatform"`
	Command      string       `json:"command"`
}

type AuditEventCreateInput struct {
	Type               AuditEventType                `json:"type"`
	CreatedAt          string                        `json:"createdAt"`
	DeploymentID       string                        `json:"deploymentId"`
	PluginName         string                        `json:"pluginName"`
	SourceEventEmitted *AuditEventSourceCreateInput  `json:"sourceEventEmitted"`
	CommandExecuted    *AuditEventCommandCreateInput `json:"commandExecuted"`
}

type AuditEventFilter struct {
	DeploymentID *string `json:"deploymentId"`
	StartDate    *string `json:"startDate"`
	EndDate      *string `json:"endDate"`
}

type AuditEventPage struct {
	Data       []AuditEvent `json:"data"`
	PageInfo   *PageInfo    `json:"pageInfo"`
	TotalCount int          `json:"totalCount"`
	TotalPages int          `json:"totalPages"`
}

func (AuditEventPage) IsPageable()                 {}
func (this AuditEventPage) GetPageInfo() *PageInfo { return this.PageInfo }
func (this AuditEventPage) GetTotalCount() int     { return this.TotalCount }

type AuditEventSourceCreateInput struct {
	Event  string                        `json:"event"`
	Source *AuditEventSourceDetailsInput `json:"source"`
}

type AuditEventSourceDetailsInput struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type BotBindings struct {
	Sources   []string `json:"sources"`
	Executors []string `json:"executors"`
}

type BotBindingsCreateInput struct {
	Sources   []*string `json:"sources"`
	Executors []*string `json:"executors"`
}

type BotBindingsUpdateInput struct {
	Sources   []*string `json:"sources"`
	Executors []*string `json:"executors"`
}

type ChannelBindingsByID struct {
	ID                    string       `json:"id"`
	Bindings              *BotBindings `json:"bindings"`
	NotificationsDisabled *bool        `json:"notificationsDisabled"`
}

type ChannelBindingsByIDCreateInput struct {
	ID                    string                  `json:"id"`
	Bindings              *BotBindingsCreateInput `json:"bindings"`
	NotificationsDisabled *bool                   `json:"notificationsDisabled"`
}

type ChannelBindingsByIDUpdateInput struct {
	ID       string                  `json:"id"`
	Bindings *BotBindingsUpdateInput `json:"bindings"`
}

type ChannelBindingsByName struct {
	Name                  string       `json:"name"`
	Bindings              *BotBindings `json:"bindings"`
	NotificationsDisabled *bool        `json:"notificationsDisabled"`
}

type ChannelBindingsByNameCreateInput struct {
	Name                  string                  `json:"name"`
	Bindings              *BotBindingsCreateInput `json:"bindings"`
	NotificationsDisabled *bool                   `json:"notificationsDisabled"`
}

type ChannelBindingsByNameUpdateInput struct {
	Name     string                  `json:"name"`
	Bindings *BotBindingsUpdateInput `json:"bindings"`
}

type CommandExecutedEvent struct {
	ID           string          `json:"id"`
	Type         *AuditEventType `json:"type"`
	PlatformUser *string         `json:"platformUser"`
	DeploymentID string          `json:"deploymentId"`
	Deployment   *Deployment     `json:"deployment"`
	CreatedAt    string          `json:"createdAt"`
	Command      string          `json:"command"`
	BotPlatform  *BotPlatform    `json:"botPlatform"`
	Channel      string          `json:"channel"`
	PluginName   string          `json:"pluginName"`
}

func (CommandExecutedEvent) IsAuditEvent()                   {}
func (this CommandExecutedEvent) GetID() string              { return this.ID }
func (this CommandExecutedEvent) GetType() *AuditEventType   { return this.Type }
func (this CommandExecutedEvent) GetDeploymentID() string    { return this.DeploymentID }
func (this CommandExecutedEvent) GetCreatedAt() string       { return this.CreatedAt }
func (this CommandExecutedEvent) GetPluginName() string      { return this.PluginName }
func (this CommandExecutedEvent) GetDeployment() *Deployment { return this.Deployment }

type Deployment struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Actions         []*Action         `json:"actions"`
	Plugins         []*Plugin         `json:"plugins"`
	Platforms       *Platforms        `json:"platforms"`
	Status          *DeploymentStatus `json:"status"`
	APIKey          *APIKey           `json:"apiKey"`
	YamlConfig      *string           `json:"yamlConfig"`
	Aliases         []*Alias          `json:"aliases"`
	HelmCommand     *string           `json:"helmCommand"`
	ResourceVersion int               `json:"resourceVersion"`
	Heartbeat       *Heartbeat        `json:"heartbeat"`
}

type DeploymentConfig struct {
	ResourceVersion int         `json:"resourceVersion"`
	Value           interface{} `json:"value"`
}

type DeploymentCreateInput struct {
	Name      string                     `json:"name"`
	Plugins   []*PluginsCreateInput      `json:"plugins"`
	Platforms *PlatformsCreateInput      `json:"platforms"`
	Actions   []*ActionCreateUpdateInput `json:"actions"`
}

type DeploymentFailureInput struct {
	ResourceVersion int    `json:"resourceVersion"`
	Message         string `json:"message"`
}

type DeploymentHeartbeatInput struct {
	NodeCount int `json:"nodeCount"`
}

type DeploymentInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeploymentPage struct {
	Data       []*Deployment `json:"data"`
	PageInfo   *PageInfo     `json:"pageInfo"`
	TotalCount int           `json:"totalCount"`
}

func (DeploymentPage) IsPageable()                 {}
func (this DeploymentPage) GetPageInfo() *PageInfo { return this.PageInfo }
func (this DeploymentPage) GetTotalCount() int     { return this.TotalCount }

type DeploymentStatus struct {
	Phase              DeploymentStatusPhase    `json:"phase"`
	Message            *string                  `json:"message"`
	BotkubeVersion     *string                  `json:"botkubeVersion"`
	Upgrade            *DeploymentUpgradeStatus `json:"upgrade"`
	LastTransitionTime *string                  `json:"lastTransitionTime"`
}

type DeploymentStatusInput struct {
	Message *string                `json:"message"`
	Phase   *DeploymentStatusPhase `json:"phase"`
}

type DeploymentUpdateInput struct {
	Name            string                     `json:"name"`
	Platforms       *PlatformsUpdateInput      `json:"platforms"`
	Plugins         []*PluginsUpdateInput      `json:"plugins"`
	Actions         []*ActionCreateUpdateInput `json:"actions"`
	ResourceVersion int                        `json:"resourceVersion"`
}

type DeploymentUpgradeStatus struct {
	NeedsUpgrade         bool   `json:"needsUpgrade"`
	TargetBotkubeVersion string `json:"targetBotkubeVersion"`
}

type Discord struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Token    string                 `json:"token"`
	BotID    string                 `json:"botId"`
	Channels []*ChannelBindingsByID `json:"channels"`
}

type DiscordCreateInput struct {
	Name     string                            `json:"name"`
	Token    string                            `json:"token"`
	BotID    string                            `json:"botId"`
	Channels []*ChannelBindingsByIDCreateInput `json:"channels"`
}

type DiscordUpdateInput struct {
	ID       *string                           `json:"id"`
	Name     string                            `json:"name"`
	Token    string                            `json:"token"`
	BotID    string                            `json:"botId"`
	Channels []*ChannelBindingsByIDUpdateInput `json:"channels"`
}

type Elasticsearch struct {
	ID                string                `json:"id"`
	Name              string                `json:"name"`
	Username          string                `json:"username"`
	Password          string                `json:"password"`
	Server            string                `json:"server"`
	SkipTLSVerify     bool                  `json:"skipTlsVerify"`
	AwsSigningRegion  *string               `json:"awsSigningRegion"`
	AwsSigningRoleArn *string               `json:"awsSigningRoleArn"`
	Indices           []*ElasticsearchIndex `json:"indices"`
}

type ElasticsearchCreateInput struct {
	Name              string                           `json:"name"`
	Username          string                           `json:"username"`
	Password          string                           `json:"password"`
	Server            string                           `json:"server"`
	SkipTLSVerify     bool                             `json:"skipTlsVerify"`
	AwsSigningRegion  *string                          `json:"awsSigningRegion"`
	AwsSigningRoleArn *string                          `json:"awsSigningRoleArn"`
	Indices           []*ElasticsearchIndexCreateInput `json:"indices"`
}

type ElasticsearchIndex struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Shards   int           `json:"shards"`
	Replicas int           `json:"replicas"`
	Bindings *SinkBindings `json:"bindings"`
}

type ElasticsearchIndexCreateInput struct {
	Name     string                   `json:"name"`
	Type     string                   `json:"type"`
	Shards   int                      `json:"shards"`
	Replicas int                      `json:"replicas"`
	Bindings *SinkBindingsCreateInput `json:"bindings"`
}

type ElasticsearchIndexUpdateInput struct {
	Name     string                   `json:"name"`
	Type     string                   `json:"type"`
	Shards   int                      `json:"shards"`
	Replicas int                      `json:"replicas"`
	Bindings *SinkBindingsUpdateInput `json:"bindings"`
}

type ElasticsearchUpdateInput struct {
	ID                *string                          `json:"id"`
	Name              string                           `json:"name"`
	Username          string                           `json:"username"`
	Password          string                           `json:"password"`
	Server            string                           `json:"server"`
	SkipTLSVerify     bool                             `json:"skipTlsVerify"`
	AwsSigningRegion  *string                          `json:"awsSigningRegion"`
	AwsSigningRoleArn *string                          `json:"awsSigningRoleArn"`
	Indices           []*ElasticsearchIndexUpdateInput `json:"indices"`
}

type Heartbeat struct {
	NodeCount *int `json:"nodeCount"`
}

type Invoice struct {
	IsOnTrial             bool           `json:"isOnTrial"`
	UpcomingAmount        int            `json:"upcomingAmount"`
	Currency              string         `json:"currency"`
	EndOfBillingCycleDate *string        `json:"endOfBillingCycleDate"`
	EndOfTrialDate        *string        `json:"endOfTrialDate"`
	Items                 []*InvoiceItem `json:"items"`
}

type InvoiceItem struct {
	Amount          int     `json:"amount"`
	PriceUnitAmount string  `json:"priceUnitAmount"`
	Currency        string  `json:"currency"`
	Description     *string `json:"description"`
}

type Mattermost struct {
	ID       string                   `json:"id"`
	Name     string                   `json:"name"`
	BotName  string                   `json:"botName"`
	URL      string                   `json:"url"`
	Token    string                   `json:"token"`
	Team     string                   `json:"team"`
	Channels []*ChannelBindingsByName `json:"channels"`
}

type MattermostCreateInput struct {
	Name     string                              `json:"name"`
	BotName  string                              `json:"botName"`
	URL      string                              `json:"url"`
	Token    string                              `json:"token"`
	Team     string                              `json:"team"`
	Channels []*ChannelBindingsByNameCreateInput `json:"channels"`
}

type MattermostUpdateInput struct {
	ID       *string                             `json:"id"`
	Name     string                              `json:"name"`
	BotName  string                              `json:"botName"`
	URL      string                              `json:"url"`
	Token    string                              `json:"token"`
	Team     string                              `json:"team"`
	Channels []*ChannelBindingsByNameUpdateInput `json:"channels"`
}

type MsTeams struct {
	ID                    string       `json:"id"`
	Name                  string       `json:"name"`
	BotName               string       `json:"botName"`
	AppID                 string       `json:"appId"`
	AppPassword           string       `json:"appPassword"`
	Port                  string       `json:"port"`
	MessagePath           string       `json:"messagePath"`
	NotificationsDisabled *bool        `json:"notificationsDisabled"`
	Bindings              *BotBindings `json:"bindings"`
}

type MsTeamsCreateInput struct {
	Name                  string                  `json:"name"`
	BotName               string                  `json:"botName"`
	AppID                 string                  `json:"appId"`
	AppPassword           string                  `json:"appPassword"`
	Port                  string                  `json:"port"`
	MessagePath           string                  `json:"messagePath"`
	NotificationsDisabled *bool                   `json:"notificationsDisabled"`
	Bindings              *BotBindingsCreateInput `json:"bindings"`
}

type MsTeamsUpdateInput struct {
	ID          *string                 `json:"id"`
	Name        string                  `json:"name"`
	BotName     string                  `json:"botName"`
	AppID       string                  `json:"appId"`
	AppPassword string                  `json:"appPassword"`
	Port        string                  `json:"port"`
	MessagePath string                  `json:"messagePath"`
	Bindings    *BotBindingsUpdateInput `json:"bindings"`
}

type NotificationPatchDeploymentConfigInput struct {
	CommunicationGroupName string      `json:"communicationGroupName"`
	Platform               BotPlatform `json:"platform"`
	ChannelAlias           string      `json:"channelAlias"`
	Disabled               bool        `json:"disabled"`
}

type Organization struct {
	ID                      string                        `json:"id"`
	DisplayName             string                        `json:"displayName"`
	Subscription            *OrganizationSubscription     `json:"subscription"`
	OwnerID                 string                        `json:"ownerId"`
	Owner                   *User                         `json:"owner"`
	Members                 []*User                       `json:"members"`
	Quota                   *Quota                        `json:"quota"`
	BillingHistoryAvailable bool                          `json:"billingHistoryAvailable"`
	UpdateOperations        *OrganizationUpdateOperations `json:"updateOperations"`
	Usage                   *Usage                        `json:"usage"`
}

type OrganizationCreateInput struct {
	DisplayName string `json:"displayName"`
}

type OrganizationSubscription struct {
	PlanName        string   `json:"planName"`
	CustomerID      *string  `json:"customerId"`
	SubscriptionID  *string  `json:"subscriptionId"`
	PlanDisplayName *string  `json:"planDisplayName"`
	IsDefaultPlan   *bool    `json:"isDefaultPlan"`
	TrialConsumed   bool     `json:"trialConsumed"`
	Invoice         *Invoice `json:"invoice"`
}

type OrganizationUpdateInput struct {
	DisplayName string `json:"displayName"`
}

type OrganizationUpdateOperations struct {
	Blocked bool     `json:"blocked"`
	Reasons []string `json:"reasons"`
}

type PageInfo struct {
	Limit       int  `json:"limit"`
	Offset      int  `json:"offset"`
	HasNextPage bool `json:"hasNextPage"`
}

type PatchDeploymentConfigInput struct {
	ResourceVersion int                                      `json:"resourceVersion"`
	Notification    *NotificationPatchDeploymentConfigInput  `json:"notification"`
	SourceBinding   *SourceBindingPatchDeploymentConfigInput `json:"sourceBinding"`
	Action          *ActionPatchDeploymentConfigInput        `json:"action"`
}

type PlatformsCreateInput struct {
	Discords        []*DiscordCreateInput       `json:"discords"`
	SocketSlacks    []*SocketSlackCreateInput   `json:"socketSlacks"`
	Mattermosts     []*MattermostCreateInput    `json:"mattermosts"`
	Webhooks        []*WebhookCreateInput       `json:"webhooks"`
	MsTeams         []*MsTeamsCreateInput       `json:"msTeams"`
	Elasticsearches []*ElasticsearchCreateInput `json:"elasticsearches"`
}

type PlatformsUpdateInput struct {
	SocketSlacks    []*SocketSlackUpdateInput   `json:"socketSlacks"`
	Discords        []*DiscordUpdateInput       `json:"discords"`
	Mattermosts     []*MattermostUpdateInput    `json:"mattermosts"`
	Webhooks        []*WebhookUpdateInput       `json:"webhooks"`
	MsTeams         []*MsTeamsUpdateInput       `json:"msTeams"`
	Elasticsearches []*ElasticsearchUpdateInput `json:"elasticsearches"`
}

type Plugin struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	DisplayName       string     `json:"displayName"`
	Type              PluginType `json:"type"`
	ConfigurationName string     `json:"configurationName"`
	Configuration     string     `json:"configuration"`
}

type PluginConfigurationGroupInput struct {
	Name           string                      `json:"name"`
	DisplayName    string                      `json:"displayName"`
	Type           PluginType                  `json:"type"`
	Configurations []*PluginConfigurationInput `json:"configurations"`
}

type PluginConfigurationGroupUpdateInput struct {
	ID             *string                     `json:"id"`
	Name           string                      `json:"name"`
	DisplayName    string                      `json:"displayName"`
	Type           PluginType                  `json:"type"`
	Configurations []*PluginConfigurationInput `json:"configurations"`
}

type PluginConfigurationInput struct {
	Name          string `json:"name"`
	Configuration string `json:"configuration"`
}

type PluginPage struct {
	Data       []*Plugin `json:"data"`
	PageInfo   *PageInfo `json:"pageInfo"`
	TotalCount int       `json:"totalCount"`
}

func (PluginPage) IsPageable()                 {}
func (this PluginPage) GetPageInfo() *PageInfo { return this.PageInfo }
func (this PluginPage) GetTotalCount() int     { return this.TotalCount }

type PluginTemplate struct {
	Name        string      `json:"name"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Type        PluginType  `json:"type"`
	Schema      interface{} `json:"schema"`
}

type PluginTemplatePage struct {
	Data []*PluginTemplate `json:"data"`
}

type PluginsCreateInput struct {
	Groups []*PluginConfigurationGroupInput `json:"groups"`
}

type PluginsUpdateInput struct {
	Groups []*PluginConfigurationGroupUpdateInput `json:"groups"`
}

type Quota struct {
	DeploymentCount      *int `json:"deploymentCount"`
	AuditRetentionPeriod *int `json:"auditRetentionPeriod"`
	MemberCount          *int `json:"memberCount"`
	NodeCount            *int `json:"nodeCount"`
}

type RemoveMemberFromOrganizationInput struct {
	OrgID  string `json:"orgId"`
	UserID string `json:"userId"`
}

type SinkBindings struct {
	Sources []string `json:"sources"`
}

type SinkBindingsCreateInput struct {
	Sources []*string `json:"sources"`
}

type SinkBindingsUpdateInput struct {
	Sources []*string `json:"sources"`
}

type SocketSlack struct {
	ID       string                   `json:"id"`
	Name     string                   `json:"name"`
	AppToken string                   `json:"appToken"`
	BotToken string                   `json:"botToken"`
	Channels []*ChannelBindingsByName `json:"channels"`
}

type SocketSlackCreateInput struct {
	Name     string                              `json:"name"`
	AppToken string                              `json:"appToken"`
	BotToken string                              `json:"botToken"`
	Channels []*ChannelBindingsByNameCreateInput `json:"channels"`
}

type SocketSlackUpdateInput struct {
	ID       *string                             `json:"id"`
	Name     string                              `json:"name"`
	AppToken string                              `json:"appToken"`
	BotToken string                              `json:"botToken"`
	Channels []*ChannelBindingsByNameUpdateInput `json:"channels"`
}

type SourceBindingPatchDeploymentConfigInput struct {
	CommunicationGroupName string      `json:"communicationGroupName"`
	Platform               BotPlatform `json:"platform"`
	ChannelAlias           string      `json:"channelAlias"`
	SourceBindings         []string    `json:"sourceBindings"`
}

type SourceEventDetails struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type SourceEventEmittedEvent struct {
	ID           string              `json:"id"`
	Type         AuditEventType      `json:"type"`
	DeploymentID string              `json:"deploymentId"`
	Deployment   *Deployment         `json:"deployment"`
	CreatedAt    string              `json:"createdAt"`
	Event        interface{}         `json:"event"`
	Source       *SourceEventDetails `json:"source"`
	PluginName   string              `json:"pluginName"`
}

func (SourceEventEmittedEvent) IsAuditEvent()                   {}
func (this SourceEventEmittedEvent) GetID() string              { return this.ID }
func (this SourceEventEmittedEvent) GetType() *AuditEventType   { return &this.Type }
func (this SourceEventEmittedEvent) GetDeploymentID() string    { return this.DeploymentID }
func (this SourceEventEmittedEvent) GetCreatedAt() string       { return this.CreatedAt }
func (this SourceEventEmittedEvent) GetPluginName() string      { return this.PluginName }
func (this SourceEventEmittedEvent) GetDeployment() *Deployment { return this.Deployment }

type SubscriptionPlan struct {
	Name             string `json:"name"`
	DisplayName      string `json:"displayName"`
	IsDefault        bool   `json:"isDefault"`
	DisplayUnitPrice int    `json:"displayUnitPrice"`
	TrialPeriodDays  int    `json:"trialPeriodDays"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type Webhook struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	URL      string        `json:"url"`
	Bindings *SinkBindings `json:"bindings"`
}

type WebhookCreateInput struct {
	Name     string                   `json:"name"`
	URL      string                   `json:"url"`
	Bindings *SinkBindingsCreateInput `json:"bindings"`
}

type WebhookUpdateInput struct {
	ID       *string                  `json:"id"`
	Name     string                   `json:"name"`
	URL      string                   `json:"url"`
	Bindings *SinkBindingsUpdateInput `json:"bindings"`
}

type AuditEventType string

const (
	AuditEventTypeCommandExecuted    AuditEventType = "COMMAND_EXECUTED"
	AuditEventTypeSourceEventEmitted AuditEventType = "SOURCE_EVENT_EMITTED"
)

var AllAuditEventType = []AuditEventType{
	AuditEventTypeCommandExecuted,
	AuditEventTypeSourceEventEmitted,
}

func (e AuditEventType) IsValid() bool {
	switch e {
	case AuditEventTypeCommandExecuted, AuditEventTypeSourceEventEmitted:
		return true
	}
	return false
}

func (e AuditEventType) String() string {
	return string(e)
}

func (e *AuditEventType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = AuditEventType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid AuditEventType", str)
	}
	return nil
}

func (e AuditEventType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type BotPlatform string

const (
	BotPlatformSLACk      BotPlatform = "SLACK"
	BotPlatformDiscord    BotPlatform = "DISCORD"
	BotPlatformMattermost BotPlatform = "MATTERMOST"
	BotPlatformMsTeams    BotPlatform = "MS_TEAMS"
)

var AllBotPlatform = []BotPlatform{
	BotPlatformSLACk,
	BotPlatformDiscord,
	BotPlatformMattermost,
	BotPlatformMsTeams,
}

func (e BotPlatform) IsValid() bool {
	switch e {
	case BotPlatformSLACk, BotPlatformDiscord, BotPlatformMattermost, BotPlatformMsTeams:
		return true
	}
	return false
}

func (e BotPlatform) String() string {
	return string(e)
}

func (e *BotPlatform) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = BotPlatform(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid BotPlatform", str)
	}
	return nil
}

func (e BotPlatform) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type DeploymentStatusPhase string

const (
	DeploymentStatusPhaseConnected    DeploymentStatusPhase = "CONNECTED"
	DeploymentStatusPhaseDisconnected DeploymentStatusPhase = "DISCONNECTED"
	DeploymentStatusPhaseFailed       DeploymentStatusPhase = "FAILED"
	DeploymentStatusPhaseCreating     DeploymentStatusPhase = "CREATING"
	DeploymentStatusPhaseUpdating     DeploymentStatusPhase = "UPDATING"
	DeploymentStatusPhaseDeleted      DeploymentStatusPhase = "DELETED"
)

var AllDeploymentStatusPhase = []DeploymentStatusPhase{
	DeploymentStatusPhaseConnected,
	DeploymentStatusPhaseDisconnected,
	DeploymentStatusPhaseFailed,
	DeploymentStatusPhaseCreating,
	DeploymentStatusPhaseUpdating,
	DeploymentStatusPhaseDeleted,
}

func (e DeploymentStatusPhase) IsValid() bool {
	switch e {
	case DeploymentStatusPhaseConnected, DeploymentStatusPhaseDisconnected, DeploymentStatusPhaseFailed, DeploymentStatusPhaseCreating, DeploymentStatusPhaseUpdating, DeploymentStatusPhaseDeleted:
		return true
	}
	return false
}

func (e DeploymentStatusPhase) String() string {
	return string(e)
}

func (e *DeploymentStatusPhase) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = DeploymentStatusPhase(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid DeploymentStatusPhase", str)
	}
	return nil
}

func (e DeploymentStatusPhase) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type PluginType string

const (
	PluginTypeSource   PluginType = "SOURCE"
	PluginTypeExecutor PluginType = "EXECUTOR"
)

var AllPluginType = []PluginType{
	PluginTypeSource,
	PluginTypeExecutor,
}

func (e PluginType) IsValid() bool {
	switch e {
	case PluginTypeSource, PluginTypeExecutor:
		return true
	}
	return false
}

func (e PluginType) String() string {
	return string(e)
}

func (e *PluginType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = PluginType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid PluginType", str)
	}
	return nil
}

func (e PluginType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
