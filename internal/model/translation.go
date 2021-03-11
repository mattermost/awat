package model

import (
	cloudModel "github.com/mattermost/mattermost-cloud/model"
)

const (
	TranslationStateRequested        = "translation-requested"
	TranslationStateInProgress       = "translation-in-progress"
	TranslationStateComplete         = "translation-complete"
	TranslationStateImportInProgress = "translation-import-in-progress"
	TranslationStateImportComplete   = "translation-import-complete"
	TranslationStateUnknown          = "translation-state-invalid"
)

type Translation struct {
	ID               string
	InstallationID   string
	Team             string
	Type             string
	Output           string
	Resource         string
	Error            string
	StartAt          int64
	CompleteAt       int64
	ImportStartAt    int64
	ImportCompleteAt int64
	LockedBy         string
}

func (t *Translation) State() string {
	if t.StartAt == 0 {
		return TranslationStateRequested
	}

	if t.CompleteAt == 0 && t.LockedBy == "" {
		return TranslationStateInProgress
	}

	if t.CompleteAt > 0 && t.LockedBy == "" {
		return TranslationStateComplete
	}

	if t.LockedBy != "" {
		return TranslationStateImportInProgress
	}

	if t.ImportCompleteAt > 0 {
		return TranslationStateImportComplete
	}

	// this state shouldn't ever happen. If it does, something weird has
	// happened that will probably require SRE intervention,
	// unfortunately
	return TranslationStateUnknown
}

func NewTranslationFromRequest(translationRequest *TranslationRequest) *Translation {
	return &Translation{
		ID:             cloudModel.NewID(),
		InstallationID: translationRequest.InstallationID,
		Type:           translationRequest.Type,
		Resource:       translationRequest.Archive,
		Team:           translationRequest.Team,
	}
}
