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
			"CompleteAt",
			"Error",
			"ID",
			"InstallationID",
			"LockedBy",
			"Resource",
			"StartAt",
			"Team",
			"Type",
		).
		From(TranslationTableName)
}

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

func (sqlStore *SQLStore) GetAllTranslations() ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations, translationSelect)
	if err != nil {
		return nil, err
	}

	return *translations, nil
}

func (sqlStore *SQLStore) GetTranslationsByInstallation(id string) ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations, translationSelect.Where("InstallationID = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return *translations, nil
}

func (sqlStore *SQLStore) GetTranslationsReadyToStart() ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations, translationSelect.Where("StartAt = 0"))
	if err != nil {
		return nil, err
	}

	return *translations, nil
}

func (sqlStore *SQLStore) StoreTranslation(translation *model.Translation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(TranslationTableName).
		SetMap(map[string]interface{}{
			"CompleteAt":     translation.CompleteAt,
			"Error":          translation.Error,
			"ID":             translation.ID,
			"InstallationID": translation.InstallationID,
			"LockedBy":       translation.LockedBy,
			"Resource":       translation.Resource,
			"StartAt":        translation.StartAt,
			"Team":           translation.Team,
			"Type":           translation.Type,
		}),
	)
	return err
}

func (sqlStore *SQLStore) UpdateTranslation(translation *model.Translation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(TranslationTableName).
		SetMap(map[string]interface{}{
			"CompleteAt":     translation.CompleteAt,
			"Error":          translation.Error,
			"ID":             translation.ID,
			"InstallationID": translation.InstallationID,
			"LockedBy":       translation.LockedBy,
			"Resource":       translation.Resource,
			"StartAt":        translation.StartAt,
			"Team":           translation.Team,
			"Type":           translation.Type,
		}).Where("ID = ?", translation.ID),
	)
	return err
}
