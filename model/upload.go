// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "strings"

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

// TrimExtensionFromArchiveFilename returns the archive filename without the extension, mostly to
// retrieve the ID from an upload/archive to use on database entries.
func TrimExtensionFromArchiveFilename(filename string) string {
	return strings.TrimSuffix(filename, archiveExtension)
}

// IsValidArchiveName checks if the provided filename is a valid for an awat supported archive
func IsValidArchiveName(filename string) bool {
	return strings.HasSuffix(filename, archiveExtension)
}
