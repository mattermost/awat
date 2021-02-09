package model

const SlackWorkspaceBackupType string = "slack"

type StartTranslationRequest struct {
	Type string
	ID   string
}

type TranslationStatus struct{}
