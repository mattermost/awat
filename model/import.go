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

const (
	ImportStateRequested  = "import-requested"
	ImportStateInProgress = "import-in-progress"
	ImportStateComplete   = "import-complete"
)

// Import represents a completed Translation that is being imported
// into an Installation in order to track that process
type Import struct {
	ID            string
	CreateAt      int64
	CompleteAt    int64
	StartAt       int64
	LockedBy      string
	TranslationID string
	Resource      string
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

// NewImport creates an Import with the appropriate creation-time
// metadata and associates it with the given translationID
func NewImport(translationID string, input string) *Import {
	return &Import{
		ID:            NewID(),
		TranslationID: translationID,
		CreateAt:      Timestamp(),
		Resource:      input,
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

// State determines and returns the current state of the Import given
// its metadata
func (i *Import) State() string {
	if i.CompleteAt != 0 {
		return ImportStateComplete
	}

	if i.StartAt == 0 {
		return ImportStateRequested
	}

	return ImportStateInProgress
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

func NewImportCompletedWorkRequestFromReader(reader io.Reader) (*ImportCompletedWorkRequest, error) {
	var request ImportCompletedWorkRequest
	err := json.NewDecoder(reader).Decode(&request)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode import completion request")
	}
	return &request, nil
}

// NewImportFromReader creates a Import from a Reader
func NewImportFromReader(reader io.Reader) (*Import, error) {
	var imp Import
	err := json.NewDecoder(reader).Decode(&imp)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode Import")
	}
	return &imp, nil
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

// Matches determines whether or not two *ImportCompletedWorkRequests
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

// String outputs a stringular representation of
// ImportCompletedWorkRequest. It is needed to satisfy the Matcher
// interface
func (i *ImportCompletedWorkRequest) String() string {
	return fmt.Sprintf("ID: %s, Error: \"%s\", CompleteAt: %d", i.ID, i.Error, i.CompleteAt)
}
