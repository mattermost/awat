// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const archiveExtension = ".zip"

// Upload represents the details of an upload process in the system.
// It includes metadata like the ID, creation and completion timestamps,
// any errors encountered, and the type of backup being uploaded.
type Upload struct {
	ID         string
	CompleteAt int64
	CreateAt   int64
	Error      string
	Type       BackupType
}

// NewUploadFromReader creates an Upload from a Reader.
func NewUploadFromReader(reader io.Reader) (*Upload, error) {
	var upload Upload
	err := json.NewDecoder(reader).Decode(&upload)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode upload")
	}
	return &upload, nil
}

// NewUploadListFromReader creates a list of uploads from a Reader.
func NewUploadListFromReader(reader io.Reader) ([]*Upload, error) {
	var uploads []*Upload
	err := json.NewDecoder(reader).Decode(&uploads)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode upload list")
	}
	return uploads, nil
}

// TrimExtensionFromArchiveFilename returns the archive filename without the extension, mostly to
// retrieve the ID from an upload/archive to use on database entries.
func TrimExtensionFromArchiveFilename(filename string) string {
	return strings.TrimSuffix(filename, archiveExtension)
}

// IsValidArchiveName checks if the provided filename is a valid for an awat supported archive
func IsValidArchiveName(filename string) bool {
	return strings.HasSuffix(filename, archiveExtension)
}
