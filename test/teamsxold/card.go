package teamsxold

import (
	"encoding/json"

	cards "github.com/DanielTitkov/go-adaptive-cards"
)

// Card struct represents a card with CardMSTeamsData and embedded cards.Card.
type Card struct {
	MsTeams CardMSTeamsData `json:"msTeams"`
	Refresh *CardRefresh    `json:"refresh,omitempty"`
	*cards.Card
}

// CardRefresh represents a refresh object for a card.
type CardRefresh struct {
	Action CardActionRefresh `json:"action"`
	// It's not yet properly handled by Teams platform, but we're already specifying that, so we will start getting
	// fewer events as soon as they will start supporting that.
	//
	// Since Adaptive Card 1.6.
	Expires string `json:"expires,omitempty"`
}

// CardActionRefresh represents a refresh action in a card.
type CardActionRefresh struct {
	Type string      `json:"type"`
	Verb string      `json:"verb,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// MarshalJSON marshals the CardActionRefresh object to its JSON representation.
// Only Action.Execute is supported.
// For more information, refer to: https://adaptivecards.io/explorer/Refresh.html
func (a CardActionRefresh) MarshalJSON() ([]byte, error) {
	a.Type = "Action.Execute"
	type alias CardActionRefresh
	return json.Marshal(alias(a))
}

// CardMSTeamsData struct holds data specific to Microsoft Teams.
type CardMSTeamsData struct {
	Width    string              `json:"width,omitempty"`
	Entities []CardMSTeamsEntity `json:"entities,omitempty"`
}

// CardMSTeamsEntity represents user or bot under Microsoft Teams.
type CardMSTeamsEntity struct {
	Type      string        `json:"type"`
	Text      string        `json:"text"`
	Mentioned CardMentioned `json:"mentioned"`
}

// CardMentioned struct represents a mentioned entity with ID and Name.
type CardMentioned struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	//Type string `json:"type"`
}
