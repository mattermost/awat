// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/awat/model"
	"github.com/pkg/errors"
)

var uploadSelect sq.SelectBuilder

// UploadTableName is the name of the database table used for storing upload records.
var UploadTableName = "Upload"

func init() {
	uploadSelect = sq.
		Select(
			"ID",
			"CompleteAt",
			"CreateAt",
			"Error",
		).
		From(UploadTableName)
}

// GetUpload will fetch the metadata about an upload from the database
// by ID
func (sqlStore *SQLStore) GetUpload(id string) (*model.Upload, error) {
	upload := new(model.Upload)

	err := sqlStore.getBuilder(sqlStore.db, upload,
		uploadSelect.Where("ID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to get upload by id")
	}

	return upload, nil
}

// CreateUpload creates an upload object in the database to represent a
// started upload
func (sqlStore *SQLStore) CreateUpload(id string, archiveType model.BackupType) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(UploadTableName).
		SetMap(map[string]interface{}{
			"CreateAt":   model.GetMillis(),
			"CompleteAt": 0,
			"ID":         id,
			"Error":      "",
			"Type":       archiveType,
		}),
	)

	return err
}

// CompleteUpload marks an upload as complete in the database, with or
// without an error message (provide an empty string if no error)
func (sqlStore *SQLStore) CompleteUpload(uploadID, errorMessage string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(UploadTableName).
		Where("ID = ?", uploadID).
		SetMap(map[string]interface{}{
			"CompleteAt": model.GetMillis(),
			"ID":         uploadID,
			"Error":      errorMessage,
		}),
	)
	return err
}
