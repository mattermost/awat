package translator

import (
	"fmt"

	"github.com/mattermost/awat/internal/model"
	"github.com/mattermost/awat/internal/slack"
)

type Translator interface {
	Translate(translation *model.Translation, bucket string) error
}

type Metadata struct {
	Options interface{}
}

func NewTranslator(archiveType string) (Translator, error) {
	if archiveType != model.SlackWorkspaceBackupType {
		return nil, fmt.Errorf("%s is not a supported workspace archive input type", archiveType)
	}

	return new(slack.SlackTranslator), nil
}
