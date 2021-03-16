package model

import (
	"encoding/json"
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
}

// ImportWorkRequest contains an identifier from the caller in order
// to claim an import for the caller at request time
type ImportWorkRequest struct {
	ProvisionerID string
}

// NewImport creates an Import with the appropriate creation-time
// metadata and associates it with the given translationID
func NewImport(translationID string) *Import {
	return &Import{
		ID:            NewID(),
		TranslationID: translationID,
		CreateAt:      Timestamp(),
	}
}

// ImportStatus provides a container for returning the State with the
// Import to the client without explicitly needing to store a state
// attribute in the database
type ImportStatus struct {
	Import

	State string
}

// State determines and returns the current state of the Import given
// its metadata
func (i *Import) State() string {
	if i.StartAt == 0 {
		return ImportStateRequested
	}

	if i.CompleteAt == 0 {
		return ImportStateInProgress
	}

	return ImportStateComplete
}

// NewImportWorkRequestFromReader creates a ImportWorkRequest from a
// Reader
func NewImportWorkRequestFromReader(reader io.Reader) (*ImportWorkRequest, error) {
	var request ImportWorkRequest
	err := json.NewDecoder(reader).Decode(&request)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &request, nil
}

// NewImportFromReader creates a Import from a Reader
func NewImportFromReader(reader io.Reader) (*Import, error) {
	var imp Import
	err := json.NewDecoder(reader).Decode(&imp)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
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
