package translator

import (
	"errors"
	"fmt"

	"github.com/mattermost/awat/internal/model"
	"github.com/mattermost/awat/internal/slack"
)

type Translator interface {
	Translate(translation *model.Translation) (string, error)
}

type Metadata struct {
	Options interface{}
}

type TranslatorOptions struct {
	ArchiveType string
	Bucket      string
	WorkingDir  string
}

func NewTranslator(t *TranslatorOptions) (Translator, error) {
	if t == nil {
		return nil, errors.New("options struct must not be nil")
	}

	if t.ArchiveType != model.SlackWorkspaceBackupType {
		return nil, fmt.Errorf("%s is not a supported workspace archive input type", t.ArchiveType)
	}

	return slack.NewSlackTranslator(t.Bucket, t.WorkingDir), nil
}
