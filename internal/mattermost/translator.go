// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package mattermost

import (
	"github.com/mattermost/awat/model"
)

// MattermostTranslator is a type that facilitates the translation of
// Mattermost workspace archives.
// nolint
type MattermostTranslator struct{}

// NewMattermostTranslator creates a new instance of MattermostTranslator.
func NewMattermostTranslator() *MattermostTranslator {
	return &MattermostTranslator{}
}

// Translate performs the translation operation for a Mattermost workspace
// archive, as defined in the provided Translation object.
func (mt *MattermostTranslator) Translate(translation *model.Translation) (string, error) {
	return translation.Resource, nil
}

// GetOutputArchiveLocalPath returns the local file system path to the
// translated archive.
func (mt *MattermostTranslator) GetOutputArchiveLocalPath() (string, error) {
	return "", nil
}

// Cleanup performs any necessary cleanup operations after translation,
// such as deleting temporary files.
func (mt *MattermostTranslator) Cleanup() error {
	return nil
}
