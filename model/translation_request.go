// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE for license information.
//

package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

const (
	SlackWorkspaceBackupType      string = "slack"
	MattermostWorkspaceBackupType string = "mattermost"
)

type TranslationRequest struct {
	Type           string
	InstallationID string
	Archive        string
	Team           string
}

type TranslationMetadata struct {
	Options interface{}
}

type TranslationStatus struct {
	Translation

	State string
}

// TODO replace all these functions with a generic one after generics ship :D

func NewTranslationRequestFromReader(reader io.Reader) (*TranslationRequest, error) {
	var request TranslationRequest
	err := json.NewDecoder(reader).Decode(&request)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &request, nil
}

func NewTranslationStatusFromReader(reader io.Reader) (*TranslationStatus, error) {
	var status TranslationStatus
	err := json.NewDecoder(reader).Decode(&status)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &status, nil
}

func NewTranslationStatusListFromReader(reader io.Reader) ([]*TranslationStatus, error) {
	var status []*TranslationStatus
	err := json.NewDecoder(reader).Decode(&status)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return status, nil
}
