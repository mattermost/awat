package model

import (
	cloudModel "github.com/mattermost/mattermost-cloud/model"
)

const TranslationStateRequested = "transaction-requested"
const TranslationStateInProgress = "transaction-in-progress"
const TranslationStateComplete = "transaction-complete"

type Translation struct {
	ID             string
	InstallationID string
	Type           string
	Metadata       *TranslationMetadata
	Resource       string
	Error          string
	StartAt        uint64
	CompleteAt     uint64
	LockedBy       string
}

func (t *Translation) State() string {
	if t.StartAt == 0 {
		return TranslationStateRequested
	}

	if t.CompleteAt == 0 {
		return TranslationStateInProgress
	}

	return TranslationStateComplete
}

func NewTranslationFromRequest(translationRequest *TranslationRequest) *Translation {
	return &Translation{
		ID:             cloudModel.NewID(),
		InstallationID: translationRequest.InstallationID,
		Type:           translationRequest.Type,
		Metadata:       translationRequest.Metadata,
		Resource:       translationRequest.Archive,
	}
}
