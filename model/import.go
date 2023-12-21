// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// Constants defining various states of import.
const (
	ImportStateRequested                  = "import-requested"
	ImportStateInstallationPreAdjustment  = "installation-pre-adjustment"
	ImportStateInProgress                 = "import-in-progress"
	ImportStateComplete                   = "import-complete"
	ImportStateInstallationPostAdjustment = "installation-post-adjustment"
	ImportStateSucceeded                  = "import-succeeded"
	ImportStateFailed                     = "import-failed"

	SizeCloud10Users  = "cloud10users"
	Size1000String    = "1000users"
	S3ExtendedTimeout = 48 * 60 * 60 * 1000
	S3DefaultTimeout  = 10 * 60 * 1000
	S3EnvKey          = "MM_FILESETTINGS_AMAZONS3REQUESTTIMEOUTMILLISECONDS"
)

// AllImportStatesPendingWork contains all import states that indicate pending work.
var AllImportStatesPendingWork = []string{
	ImportStateRequested,
	ImportStateInstallationPreAdjustment,
	ImportStateInProgress,
	ImportStateComplete,
	ImportStateInstallationPostAdjustment,
}

// Import represents a completed Translation that is being imported
// into an Installation in order to track that process
type Import struct {
	ID            string
	TranslationID string
	Resource      string
	CreateAt      int64
	StartAt       int64
	CompleteAt    int64
	State         string
	LockedBy      string
	ImportBy      string
	Error         string
}

// ImportWorkRequest contains an identifier from the caller in order
// to claim an import for the caller at request time
type ImportWorkRequest struct {
	ProvisionerID string
}

// ImportCompletedWorkRequest contains the metadata needed from the
// Provisioner for the AWAT to mark that an import has finished with
// or without an error
type ImportCompletedWorkRequest struct {
	ID         string
	CompleteAt int64
	Error      string
}

// NewImport returns a new import resource.
func NewImport(translationID, importResource string) *Import {
	return &Import{
		TranslationID: translationID,
		Resource:      importResource,
		State:         ImportStateRequested,
	}
}

// ImportStatus provides a container for returning the State with the
// Import to the client without explicitly needing to store a state
// attribute in the database
type ImportStatus struct {
	Import

	InstallationID string
	Users          int
	Team           string
	State          string
	Type           BackupType
}

// NewImportWorkRequestFromReader creates a ImportWorkRequest from a
// Reader
func NewImportWorkRequestFromReader(reader io.Reader) (*ImportWorkRequest, error) {
	var request ImportWorkRequest
	err := json.NewDecoder(reader).Decode(&request)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode import start request")
	}
	return &request, nil
}

// NewImportCompletedWorkRequestFromReader creates an ImportCompletedWorkRequest from an io.Reader.
func NewImportCompletedWorkRequestFromReader(reader io.Reader) (*ImportCompletedWorkRequest, error) {
	var request ImportCompletedWorkRequest
	err := json.NewDecoder(reader).Decode(&request)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode import completion request")
	}
	return &request, nil
}

// NewImportStatusFromReader creates a ImportStatus from a Reader
func NewImportStatusFromReader(reader io.Reader) (*ImportStatus, error) {
	var status ImportStatus
	err := json.NewDecoder(reader).Decode(&status)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode import status")
	}
	return &status, nil
}

// NewImportStatusListFromReader creates a list of ImportStatuses from
// a Reader
func NewImportStatusListFromReader(reader io.Reader) ([]*ImportStatus, error) {
	var status []*ImportStatus
	err := json.NewDecoder(reader).Decode(&status)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode import status list")
	}
	return status, nil
}

// Matches determines whether two *ImportCompletedWorkRequests
// point to the same logical request, in testing. Since this is for
// testing, timestamps are ignored
func (a *ImportCompletedWorkRequest) Matches(input interface{}) bool {
	switch b := input.(type) {
	case *ImportCompletedWorkRequest:
		return a.ID == b.ID && a.Error == b.Error
	default:
		return false
	}
}

// String outputs a string representation of
// ImportCompletedWorkRequest. It is needed to satisfy the Matcher
// interface
func (a *ImportCompletedWorkRequest) String() string {
	return fmt.Sprintf("ID: %s, Error: \"%s\", CompleteAt: %d", a.ID, a.Error, a.CompleteAt)
}
