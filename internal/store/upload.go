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

func (sqlStore *SQLStore) CreateUpload(id string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(UploadTableName).
		SetMap(map[string]interface{}{
			"CreateAt":   model.Timestamp(),
			"CompleteAt": 0,
			"ID":         id,
			"Error":      "",
		}),
	)

	return err
}

// StoreUpload saves the specified Upload to the database,
// assuming it is new
func (sqlStore *SQLStore) CompleteUpload(uploadID, errorMessage string) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(UploadTableName).
		Where("ID = ?", uploadID).
		SetMap(map[string]interface{}{
			"CompleteAt": model.Timestamp(),
			"ID":         uploadID,
			"Error":      errorMessage,
		}),
	)
	return err
}
