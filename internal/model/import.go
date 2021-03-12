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

type Import struct {
	ID            string
	CreateAt      int64
	CompleteAt    int64
	StartAt       int64
	LockedBy      string
	TranslationID string
}

type ImportWorkRequest struct {
	ProvisionerID string
}

func NewImportWorkRequestFromReader(reader io.Reader) (*ImportWorkRequest, error) {
	var request ImportWorkRequest
	err := json.NewDecoder(reader).Decode(&request)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &request, nil
}

func NewImportFromReader(reader io.Reader) (*Import, error) {
	var imp Import
	err := json.NewDecoder(reader).Decode(&imp)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode translation start request")
	}
	return &imp, nil
}
