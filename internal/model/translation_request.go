package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

const SlackWorkspaceBackupType string = "slack"

type TranslationRequest struct {
	Type           string
	InstallationID string
	Metadata       *TranslationMetadata
	Archive        string
}

type TranslationMetadata struct {
}

type TranslationStatus struct {
	ID             string
	InstallationID string
	State          string
}

func NewTranslationRequestFromReader(reader io.Reader) (*TranslationRequest, error) {
	var request TranslationRequest
	err := json.NewDecoder(reader).Decode(&request)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &request, nil
}
