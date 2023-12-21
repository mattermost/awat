package validators

import (
	"fmt"

	"github.com/mattermost/awat/model"
)

// Validator defines an interface for validating data archives.
type Validator interface {
	Validate(archiveName string) error
}

// NewValidator creates a new validator based on the specified archive type.
// It supports different archive types, such as Mattermost and Slack.
// Returns the appropriate validator or an error if the archive type is unsupported.
func NewValidator(archiveType model.BackupType) (Validator, error) {
	switch archiveType {
	case model.MattermostWorkspaceBackupType:
		return NewMattermostValidator(), nil
	case model.SlackWorkspaceBackupType:
		return NewSlackValidator(), nil
	}

	return nil, fmt.Errorf("can't find validator for type: %s", archiveType)
}
