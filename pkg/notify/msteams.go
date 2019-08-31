package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"
)

const (
	// normal colour
	normal string = "2DC72D"
	// warning colour
	warning string = "DEFF22"
	// danger colour
	danger string = "8C1A1A"

	// Constants for sending  messageCards
	messageType   = "MessageCard"
	schemaContext = "http://schema.org/extensions"
)

var themeColor = map[events.Level]string{
	events.Info:     normal,
	events.Warn:     warning,
	events.Debug:    normal,
	events.Error:    danger,
	events.Critical: danger,
}

// MsTeams contains msteams webhook url to send notification to
type MsTeams struct {
	URL         string
	NotifType   config.NotifType
	ClusterName string
}

// MessageCard is for the Card Fields to send in Teams
type MessageCard struct {
	Type       string    `json:"@type"`
	Context    string    `json:"@context"`
	ThemeColor string    `json:"themeColor"`
	Summary    string    `json:"summary"`
	Title      string    `json:"title,omitempty"`
	Text       string    `json:"text,omitempty"`
	Sections   []Section `json:"sections"`
}

// Section is placed under MessageCard.Sections
type Section struct {
	ActivityTitle string `json:"activitytitle"`
	Facts         []Fact `json:"facts"`
	Markdown      bool   `json:"markdown"`
}

// Fact is placed under Section.Fact
type Fact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NewMsTeams returns new MsTeams object
func NewMsTeams(c *config.Config) Notifier {
	return &MsTeams{
		URL:         c.Communications.MsTeams.URL,
		NotifType:   c.Communications.MsTeams.NotifType,
		ClusterName: c.Settings.ClusterName,
	}
}

// SendEvent sends event notification to MsTeams
func (m *MsTeams) SendEvent(event events.Event) error {

	// set missing cluster name to event object
	event.Cluster = m.ClusterName

	card := &MessageCard{
		Type:    messageType,
		Context: schemaContext,
		Title:   fmt.Sprintf("%s", event.Kind+" "+string(event.Type)),
	}

	card.Summary = card.GenerateTitle(event)

	card.ThemeColor = themeColor[event.Level]

	switch m.NotifType {
	case config.LongNotify:

		sectionFacts := []Fact{}

		if event.Cluster != "" {
			sectionFacts = append(sectionFacts, Fact{
				Name:  "Cluster",
				Value: event.Cluster,
			})
		}

		sectionFacts = append(sectionFacts, Fact{
			Name:  "Name",
			Value: event.Name,
		})

		if event.Namespace != "" {
			sectionFacts = append(sectionFacts, Fact{
				Name:  "Namespace",
				Value: event.Namespace,
			})
		}

		if event.Reason != "" {
			sectionFacts = append(sectionFacts, Fact{
				Name:  "Reason",
				Value: event.Reason,
			})
		}

		if len(event.Messages) > 0 {
			message := ""
			for _, m := range event.Messages {
				message = message + m + "\n"
			}
			sectionFacts = append(sectionFacts, Fact{
				Name:  "Message",
				Value: message,
			})
		}

		if event.Action != "" {
			sectionFacts = append(sectionFacts, Fact{
				Name:  "Action",
				Value: event.Action,
			})
		}

		if len(event.Recommendations) > 0 {
			rec := ""
			for _, r := range event.Recommendations {
				rec = rec + r + "\n"
			}
			sectionFacts = append(sectionFacts, Fact{
				Name:  "Recommendations",
				Value: rec,
			})
		}

		if len(event.Warnings) > 0 {
			warn := ""
			for _, w := range event.Warnings {
				warn = warn + w + "\n"
			}
			sectionFacts = append(sectionFacts, Fact{
				Name:  "Warnings",
				Value: warn,
			})
		}

		card.Sections = append(card.Sections, Section{
			Facts:    sectionFacts,
			Markdown: true,
		})

	case config.ShortNotify:
		fallthrough

	default:
		card.Sections = append(card.Sections, Section{
			ActivityTitle: card.GenerateTitle(event),
			Markdown:      true,
		})

		card.Sections = append(card.Sections, Section{
			ActivityTitle: card.GenerateMessage(event),
			Markdown:      true,
		})
	}

	// post card to Msteam channel
	if _, err := m.PostCard(card); err != nil {
		log.Logger.Error(err.Error())
		return err
	}

	log.Logger.Debugf("Event successfully sent to MS Teams >> %+v", event)
	return nil
}

// SendMessage sends message to MsTeams
func (m *MsTeams) SendMessage(msg string) error {

	card := &MessageCard{
		Type:    messageType,
		Context: schemaContext,
		Summary: msg,
	}

	card.ThemeColor = themeColor[events.Info]

	card.Sections = append(card.Sections, Section{
		ActivityTitle: msg,
		Markdown:      true,
	})

	if _, err := m.PostCard(card); err != nil {
		log.Logger.Error(err.Error())
		return err
	}

	log.Logger.Debug("Message successfully sent to MS Teams")
	return nil
}

// PostCard sends the JSON Encoded MessageCard to the msteams URL
func (m *MsTeams) PostCard(card *MessageCard) (*http.Response, error) {

	buffer := new(bytes.Buffer)

	if err := json.NewEncoder(buffer).Encode(card); err != nil {
		return nil, fmt.Errorf("Failed encoding message card: %v", err)
	}

	res, err := http.Post(m.URL, "application/json", buffer)
	if err != nil {
		return nil, fmt.Errorf("Failed sending to msteams url %s. Got the error: %v", m.URL, err)
	}

	if res.StatusCode != http.StatusOK {
		resMessage, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("Failed reading Teams http response: %v", err)
		}
		return nil, fmt.Errorf("Failed sending to the Teams Channel. Teams http response: %s, %s",
			string(res.StatusCode), string(resMessage))
	}

	if err := res.Body.Close(); err != nil {
		return nil, err
	}
	return res, nil
}

// GenerateMessage generates event messages to be sent with the card
func (card *MessageCard) GenerateMessage(event events.Event) (message string) {

	if len(event.Messages) > 0 {
		for _, m := range event.Messages {
			message = message + "> " + m + "\n"
		}
	}
	if len(event.Recommendations) > 0 {
		recommend := ""
		for _, m := range event.Recommendations {
			recommend = recommend + "\n- " + m
		}
		message = message + "\n" + "Recommendations:\n" + recommend
	}
	if len(event.Warnings) > 0 {
		warning := ""
		for _, m := range event.Warnings {
			warning = warning + "\n- " + m
		}
		message = message + "\n" + "Warnings:\n" + warning
	}
	return message
}

// GenerateTitle generates message card title
func (card *MessageCard) GenerateTitle(event events.Event) (title string) {

	switch event.Type {
	case config.CreateEvent, config.DeleteEvent, config.UpdateEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			title = fmt.Sprintf(
				"%s `%s` in of cluster `%s` has been %s:",
				event.Kind,
				event.Name,
				event.Cluster,
				event.Type+"d",
			)
		default:
			title = fmt.Sprintf(
				"%s `%s` in of cluster `%s`, namespace `%s` has been %s:",
				event.Kind,
				event.Name,
				event.Cluster,
				event.Namespace,
				event.Type+"d",
			)
		}
	case config.ErrorEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			title = fmt.Sprintf(
				"Error Occurred in %s: `%s` of cluster `%s`:",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			title = fmt.Sprintf(
				"Error Occurred in %s: `%s` of cluster `%s`, namespace `%s`:",
				event.Kind,
				event.Name,
				event.Cluster,
				event.Namespace,
			)
		}
	case config.WarningEvent:
		switch event.Kind {
		case "Namespace", "Node", "PersistentVolume", "ClusterRole", "ClusterRoleBinding":
			title = fmt.Sprintf(
				"Warning %s: `%s` of cluster `%s`:",
				event.Kind,
				event.Name,
				event.Cluster,
			)
		default:
			title = fmt.Sprintf(
				"Warning %s: `%s` of cluster `%s`, namespace `%s`:",
				event.Kind,
				event.Name,
				event.Cluster,
				event.Namespace,
			)
		}
	}
	return title
}
