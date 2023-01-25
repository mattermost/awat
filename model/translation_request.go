// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type BackupType string

const (
	SlackWorkspaceBackupType      BackupType = "slack"
	MattermostWorkspaceBackupType BackupType = "mattermost"
)

type TranslationRequest struct {
	Type           BackupType
	InstallationID string
	Archive        string
	Team           string
	UploadID       *string
}

// Validate validates the values of a translation create request.
func (request *TranslationRequest) Validate() error {
	if len(request.InstallationID) == 0 {
		return errors.New("must specify installation ID")
	}
	if len(request.Type) == 0 {
		return errors.New("must specify backup type")
	}
	if request.Type == SlackWorkspaceBackupType && len(request.Team) == 0 {
		return errors.New("must specify team with slack buckup type")
	}
	if !strings.HasSuffix(request.Archive, ".zip") {
		return errors.New("archive must be a valid zip file")
	}
	if request.Archive == ".zip" {
		return errors.New("zip archive has no filename")
	}

	return nil
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

	err = request.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "translation request failed validation")
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

type ArchiveUploadRequest struct {
	Type BackupType
}

func (r ArchiveUploadRequest) Validate() error {
	if r.Type != SlackWorkspaceBackupType && r.Type != MattermostWorkspaceBackupType {
		return errors.New("invalid backup type")
	}

	return nil
}

func NewArchiveUploadFromURLQuery(values url.Values) (*ArchiveUploadRequest, error) {
	var request ArchiveUploadRequest

	request.Type = BackupType(values.Get("type"))

	return &request, nil
}
