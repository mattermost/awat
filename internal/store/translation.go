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
			"Metadata",
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

func (sqlStore *SQLStore) GetTranslationByInstallation(id string) (*model.Translation, error) {
	return sqlStore.getTranslationByField("InstallationID", id)
}

func (sqlStore *SQLStore) getTranslationByField(field, value string) (*model.Translation, error) {
	translation := new(model.Translation)
	err := sqlStore.getBuilder(sqlStore.db, translation,
		translationSelect.Where("? = ?", field, value))

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
			"Metadata":       translation.Metadata,
			"Resource":       translation.Resource,
			"Error":          translation.Error,
			"StartAt":        translation.StartAt,
			"CompleteAt":     translation.CompleteAt,
			"LockedBy":       translation.LockedBy,
		}),
	)
	return err
}

func (sqlStore *SQLStore) UpdateTranslation(translation *model.Translation) (*model.Translation, error) {
	return nil, nil
}
