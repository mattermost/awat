// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"

	"github.com/pkg/errors"
)

// BackupType defines the type of backup (e.g., Slack, Mattermost).
type BackupType string

// Backup type constants.
const (
	SlackWorkspaceBackupType      BackupType = "slack"
	MattermostWorkspaceBackupType BackupType = "mattermost"
)

// TranslationRequest represents a request for translating a workspace archive.
type TranslationRequest struct {
	Type            BackupType
	InstallationID  string
	Archive         string
	Team            string
	UploadID        *string
	ValidateArchive bool
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
		return errors.New("must specify team with slack backup type")
	}
	if !IsValidArchiveName(request.Archive) {
		return errors.New("archive must be a valid zip file")
	}
	if request.Archive == ".zip" {
		return errors.New("zip archive has no filename")
	}

	return nil
}

// TranslationMetadata holds metadata related to a translation.
type TranslationMetadata struct {
	Options interface{}
}

// TranslationStatus represents the status of a translation.
type TranslationStatus struct {
	Translation

	State string
}

// NewTranslationRequestFromReader creates a TranslationRequest from an io.Reader.
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

// NewTranslationStatusFromReader creates a TranslationStatus from an io.Reader.
func NewTranslationStatusFromReader(reader io.Reader) (*TranslationStatus, error) {
	var status TranslationStatus
	err := json.NewDecoder(reader).Decode(&status)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &status, nil
}

// NewTranslationStatusFromBytes creates a TranslationStatus from a byte slice.
func NewTranslationStatusFromBytes(data []byte) (*TranslationStatus, error) {
	var status TranslationStatus
	err := json.Unmarshal(data, &status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &status, nil
}

// NewTranslationStatusListFromReader creates a list of TranslationStatus from an io.Reader.
func NewTranslationStatusListFromReader(reader io.Reader) ([]*TranslationStatus, error) {
	var status []*TranslationStatus
	err := json.NewDecoder(reader).Decode(&status)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return status, nil
}

// ArchiveUploadRequest represents a request to upload an archive.
type ArchiveUploadRequest struct {
	Type BackupType
}

// Validate checks if the ArchiveUploadRequest fields are valid.
func (r ArchiveUploadRequest) Validate() error {
	if r.Type != SlackWorkspaceBackupType && r.Type != MattermostWorkspaceBackupType {
		return errors.New("invalid backup type")
	}

	return nil
}

// NewArchiveUploadFromURLQuery creates an ArchiveUploadRequest from URL query values.
func NewArchiveUploadFromURLQuery(values url.Values) (*ArchiveUploadRequest, error) {
	var request ArchiveUploadRequest

	request.Type = BackupType(values.Get("type"))

	return &request, nil
}
