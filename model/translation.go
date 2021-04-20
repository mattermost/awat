package model

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

const (
	TranslationStateRequested  = "translation-requested"
	TranslationStateInProgress = "translation-in-progress"
	TranslationStateComplete   = "translation-complete"
)

// Translation represents a single process of converting a foreign
// workspace archive into a native Mattermost workspace import archive
type Translation struct {
	ID             string
	InstallationID string
	Team           string
	Users          int
	Type           string
	Resource       string
	CreateAt       int64
	StartAt        int64
	CompleteAt     int64
	LockedBy       string
}

// State provides a container for returning the state with the
// Translation to the client without explicitly needing to store a state
// attribute in the database
func (t *Translation) State() string {
	if t.StartAt == 0 {
		return TranslationStateRequested
	}

	if t.CompleteAt == 0 {
		return TranslationStateInProgress
	}

	return TranslationStateComplete
}

// NewTranslationFromRequest creates a new Translation from a
// TranslationRequest
func NewTranslationFromRequest(translationRequest *TranslationRequest) *Translation {
	teamName := cleanTeamName(translationRequest.Team)

	return &Translation{
		ID:             NewID(),
		InstallationID: translationRequest.InstallationID,
		Type:           translationRequest.Type,
		Resource:       translationRequest.Archive,
		Team:           teamName,
	}
}

const (
	TEAM_NAME_MAX_LENGTH = 64
	TEAM_NAME_MIN_LENGTH = 2
)

var validTeamNameCharacter = regexp.MustCompile(`^[a-z0-9-]$`)
var validAlphaNum = regexp.MustCompile(`^[a-z0-9]+([a-z\-0-9]+|(__)?)[a-z0-9]+$`)

// Team Display Name may be <= 64 chars but Team Name must be <= 16 so
// this helper function will truncate names that are too long
func shortenTeamName(teamName string) string {
	return teamName[:int(math.Min(float64(len(teamName)), float64(16)))]
}

var reservedTeamNames = []string{
	"admin",
	"api",
	"channel",
	"claim",
	"error",
	"files",
	"help",
	"landing",
	"login",
	"mfa",
	"oauth",
	"plug",
	"plugins",
	"post",
	"signup",
}

// lifted from mattermost-server
func cleanTeamName(s string) string {
	s = strings.ToLower(strings.Replace(s, " ", "-", -1))

	for _, value := range reservedTeamNames {
		if strings.Index(s, value) == 0 {
			s = strings.Replace(s, value, "", -1)
		}
	}

	s = strings.TrimSpace(s)

	for _, c := range s {
		char := fmt.Sprintf("%c", c)
		if !validTeamNameCharacter.MatchString(char) {
			s = strings.Replace(s, char, "", -1)
		}
	}

	s = strings.Trim(s, "-")

	if !isValidTeamName(s) {
		s = NewID()
	}

	return s
}

// Also lifted from mattermost-server but with the addition of a check
// for max length
func isValidTeamName(s string) bool {
	if !isValidAlphaNum(s) {
		return false
	}

	if len(s) < TEAM_NAME_MIN_LENGTH ||
		len(s) > TEAM_NAME_MAX_LENGTH {
		return false
	}

	return true
}

func isValidAlphaNum(s string) bool {
	return validAlphaNum.MatchString(s)
}
