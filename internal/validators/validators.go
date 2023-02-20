package validators

import (
	"fmt"

	"github.com/mattermost/awat/model"
)

type Validator interface {
	Validate(archiveName string) error
}

func NewValidator(archiveType model.BackupType) (Validator, error) {
	switch archiveType {
	case model.MattermostWorkspaceBackupType:
		return NewMattermostValidator(), nil
	case model.SlackWorkspaceBackupType:
		return NewSlackValidator(), nil
	}

	return nil, fmt.Errorf("can't find validator for type: %s", archiveType)
}
