// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Constants defining various states of translation.
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
	Resource       string
	UploadID       *string
	Type           BackupType
	Team           string
	Users          int
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

// NewTranslationFromRequest returns a new Translation from a TranslationRequest.
func NewTranslationFromRequest(translationRequest *TranslationRequest) *Translation {
	teamName := translationRequest.Team
	if translationRequest.Type != MattermostWorkspaceBackupType {
		teamName = cleanTeamName(teamName)
	}

	return &Translation{
		InstallationID: translationRequest.InstallationID,
		Type:           translationRequest.Type,
		Resource:       translationRequest.Archive,
		UploadID:       translationRequest.UploadID,
		Team:           teamName,
	}
}

const (
	// TeamNameMaxLength is the maximum length allowed for team names.
	TeamNameMaxLength = 64
	// TeamNameMinLength is the minimum length required for team names.
	TeamNameMinLength = 2
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

	if len(s) < TeamNameMinLength ||
		len(s) > TeamNameMaxLength {
		return false
	}

	return true
}

// isValidAlphaNum checks if a string is alphanumeric and matches the valid team name pattern.
func isValidAlphaNum(s string) bool {
	return validAlphaNum.MatchString(s)
}
