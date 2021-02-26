package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/awat/internal/model"
	"github.com/pkg/errors"
)

const TranslationTableName = "Translation"

var translationSelect sq.SelectBuilder

func init() {
	translationSelect = sq.
		Select(
			"ID",
			"InstallationID",
			"Type",
			"Resource",
			"Error",
			"StartAt",
			"CompleteAt",
			"LockedBy",
		).
		From(TranslationTableName)
}

func (sqlStore *SQLStore) GetTranslation(id string) (*model.Translation, error) {
	return sqlStore.getTranslationByField("ID", id)
}

func (sqlStore *SQLStore) GetAllTranslations() ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations, translationSelect)
	if err != nil {
		return nil, err
	}

	return *translations, nil
}

func (sqlStore *SQLStore) GetTranslationByInstallation(id string) (*model.Translation, error) {
	return sqlStore.getTranslationByField("InstallationID", id)
}

func (sqlStore *SQLStore) GetTranslationsReadyToStart() ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations, translationSelect.Where("StartAt = 0"))
	if err != nil {
		return nil, err
	}

	return *translations, nil
}

func (sqlStore *SQLStore) getTranslationByField(field, value string) (*model.Translation, error) {
	translation := new(model.Translation)
	var err error
	builder := translationSelect

	// this could be a string replace to be more flexible, but at the
	// risk of allowing fields to be specified here that are not
	// previously ordained by this software.
	if field == "ID" {
		builder = builder.Where("ID = ?", value)
	} else if field == "InstallationID" {
		builder = builder.Where("InstallationID = ?", value)
	} else {
		return nil, errors.Errorf("tried to query for a translation with unsupported input field %s", field)
	}

	err = sqlStore.getBuilder(sqlStore.db, translation, builder)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get translation by id")
	}

	return translation, nil
}

func (sqlStore *SQLStore) StoreTranslation(translation *model.Translation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(TranslationTableName).
		SetMap(map[string]interface{}{
			"ID":             translation.ID,
			"InstallationID": translation.InstallationID,
			"Type":           translation.Type,
			"Resource":       translation.Resource,
			"Error":          translation.Error,
			"StartAt":        translation.StartAt,
			"CompleteAt":     translation.CompleteAt,
			"LockedBy":       translation.LockedBy,
		}),
	)
	return err
}

func (sqlStore *SQLStore) UpdateTranslation(translation *model.Translation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(TranslationTableName).
		SetMap(map[string]interface{}{
			"ID":             translation.ID,
			"InstallationID": translation.InstallationID,
			"Type":           translation.Type,
			"Resource":       translation.Resource,
			"Error":          translation.Error,
			"StartAt":        translation.StartAt,
			"CompleteAt":     translation.CompleteAt,
			"LockedBy":       translation.LockedBy,
		}).Where("ID = ?", translation.ID),
	)
	return err
}
