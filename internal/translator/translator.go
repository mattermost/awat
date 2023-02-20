// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package translator

import (
	"errors"
	"fmt"

	"github.com/mattermost/awat/internal/mattermost"
	"github.com/mattermost/awat/internal/slack"
	"github.com/mattermost/awat/model"
)

// Translator defines the interface that must be satisfied to allow
// for converting foreign workspace archives to the Mattermost format
type Translator interface {

	// Translate performs the converstion from the input type to a mattermost supported import
	Translate(translation *model.Translation) (outputFilename string, err error)

	// GetOutputArchiveLocalPath returns the local accesible path to the archive file
	GetOutputArchiveLocalPath() (string, error)

	// Cleanup cleans up resources, like local files
	Cleanup() error
}

// TranslatorOptions holds the extra data needed to instantiate a
// concrete Translator
type TranslatorOptions struct {
	ArchiveType model.BackupType
	Bucket      string
	WorkingDir  string
}

// NewTranslator returns a Translator capable of translating some
// foreign workspace archive into a Mattermost backup
// archive. Currently only Slack is supported.
func NewTranslator(t *TranslatorOptions) (Translator, error) {
	if t == nil {
		return nil, errors.New("options struct must not be nil")
	}

	if t.ArchiveType == model.SlackWorkspaceBackupType {
		return slack.NewSlackTranslator(t.Bucket, t.WorkingDir)
	}

	if t.ArchiveType == model.MattermostWorkspaceBackupType {
		return mattermost.NewMattermostTranslator(), nil
	}

	return nil, fmt.Errorf("%s is not a supported workspace archive input type", t.ArchiveType)
}
