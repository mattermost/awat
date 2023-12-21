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

// TranslationTableName is the name of the database table used for storing translation data.
const TranslationTableName = "Translation"

var translationSelect sq.SelectBuilder

func init() {
	translationSelect = sq.
		Select(
			"CompleteAt",
			"CreateAt",
			"StartAt",
			"ID",
			"InstallationID",
			"LockedBy",
			"Resource",
			"Team",
			"Users",
			"Type",
		).
		From(TranslationTableName)
}

// GetTranslation returns the Translation that corresponds to the identifier ID
func (sqlStore *SQLStore) GetTranslation(id string) (*model.Translation, error) {
	translation := new(model.Translation)
	builder := translationSelect
	builder = builder.Where("ID = ?", id)

	err := sqlStore.getBuilder(sqlStore.db, translation, builder)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to get translation by id")
	}

	return translation, nil

}

// GetAllTranslations returns every Translation in the DB
// TODO pagination
func (sqlStore *SQLStore) GetAllTranslations() ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations, translationSelect)
	if err != nil {
		return nil, err
	}

	return *translations, nil
}

// GetTranslationsByInstallation provides a convenience method for
// fetching every Translation related to a given Installation by its
// ID
func (sqlStore *SQLStore) GetTranslationsByInstallation(id string) ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations,
		translationSelect.
			Where("InstallationID = ?", id),
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return *translations, nil
}

// GetTranslationReadyToStart returns a batch of Translations that
// are ready to go, with a maximum of ten, sorted from oldest to
// newest
func (sqlStore *SQLStore) GetTranslationReadyToStart() (*model.Translation, error) {
	translations := []*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, &translations,
		translationSelect.
			Where("StartAt = 0").
			Where("LockedBy = ''").
			OrderBy("CreateAt ASC").
			Limit(1),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to find a ready Translation")
	}

	if len(translations) == 0 {
		return nil, nil
	}

	return translations[0], nil
}

// CreateTranslation stores a new translation.
func (sqlStore *SQLStore) CreateTranslation(translation *model.Translation) error {
	translation.ID = model.NewID()
	translation.CreateAt = model.GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(TranslationTableName).
		SetMap(map[string]interface{}{
			"ID":             translation.ID,
			"CreateAt":       translation.CreateAt,
			"StartAt":        translation.StartAt,
			"CompleteAt":     translation.CompleteAt,
			"InstallationID": translation.InstallationID,
			"LockedBy":       translation.LockedBy,
			"Resource":       translation.Resource,
			"Team":           translation.Team,
			"Users":          translation.Users,
			"Type":           translation.Type,
			"UploadID":       translation.UploadID,
		}),
	)
	return err
}

// UpdateTranslation stores changes to the provided translation in the database.
func (sqlStore *SQLStore) UpdateTranslation(translation *model.Translation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(TranslationTableName).
		SetMap(map[string]interface{}{
			"CompleteAt":     translation.CompleteAt,
			"CreateAt":       translation.CreateAt,
			"StartAt":        translation.StartAt,
			"ID":             translation.ID,
			"InstallationID": translation.InstallationID,
			"LockedBy":       translation.LockedBy,
			"Resource":       translation.Resource,
			"Team":           translation.Team,
			"Users":          translation.Users,
			"Type":           translation.Type,
		}).Where("ID = ?", translation.ID),
	)
	return err
}

// TryLockTranslation attempts to claim the given translation for the
// owner ID provided and returns an error if it fails to do so
func (sqlStore *SQLStore) TryLockTranslation(translation *model.Translation, owner string) error {
	sqlStore.logger.Infof("Locking Translation %s as %s", translation.ID, owner)
	translation.LockedBy = owner

	result, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(TranslationTableName).
			SetMap(map[string]interface{}{"LockedBy": owner}).
			Where("ID = ?", translation.ID).
			Where("LockedBy = ?", ""),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to lock Translation %s", translation.ID)
	}
	if rows, err := result.RowsAffected(); rows != 1 || err != nil {
		if err != nil {
			return errors.Wrapf(err, "wrong number of rows while trying to unlock %s", translation.ID)
		}
		return errors.Errorf("wrong number of rows while trying to unlock %s", translation.ID)
	}
	return nil
}

// UnlockTranslation clears the lock on the given translation
func (sqlStore *SQLStore) UnlockTranslation(translation *model.Translation) error {
	translation.LockedBy = ""
	err := sqlStore.UpdateTranslation(translation)
	if err != nil {
		return errors.Wrapf(err, "failed to unlock Translation %s", translation.ID)
	}
	return nil
}
