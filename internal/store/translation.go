package store

import (
	"database/sql"
	"time"

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
			"CreateAt",
			"StartAt",

			"ID",
			"InstallationID",
			"LockedBy",
			"Output",
			"Resource",
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

func (sqlStore *SQLStore) GetTranslationsReadyToStart() ([]*model.Translation, error) {
	translations := &[]*model.Translation{}
	err := sqlStore.selectBuilder(sqlStore.db, translations,
		translationSelect.
			Where("StartAt = 0").
			Where("LockedBy = ''").
			OrderBy("CreateAt ASC").
			Limit(10),
	)

	if err != nil {
		return nil, err
	}

	return *translations, nil
}

func (sqlStore *SQLStore) StoreTranslation(translation *model.Translation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(TranslationTableName).
		SetMap(map[string]interface{}{
			"CompleteAt": translation.CompleteAt,
			"CreateAt":   time.Now().Unix() / 1000,
			"StartAt":    translation.StartAt,

			"ID":             translation.ID,
			"InstallationID": translation.InstallationID,
			"LockedBy":       translation.LockedBy,
			"Output":         translation.Output,
			"Resource":       translation.Resource,
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
			"CompleteAt": translation.CompleteAt,
			"StartAt":    translation.StartAt,

			"ID":             translation.ID,
			"InstallationID": translation.InstallationID,
			"LockedBy":       translation.LockedBy,
			"Output":         translation.Output,
			"Resource":       translation.Resource,
			"Team":           translation.Team,
			"Type":           translation.Type,
		}).Where("ID = ?", translation.ID),
	)
	return err
}

func (sqlStore *SQLStore) TryLockTranslation(translationID string, owner string) error {
	_, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(TranslationTableName).
			SetMap(map[string]interface{}{
				"LockedBy": owner,
			}).
			Where("ID = ?", translationID).
			Where("LockedBy = ?", ""),
	)

	return errors.Wrapf(err, "failed to lock Translation %s", translationID)
}

func (sqlStore *SQLStore) UnlockTranslation(translationID string) error {
	_, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(TranslationTableName).
			SetMap(map[string]interface{}{
				"LockedBy": "",
			}).
			Where("ID = ?", translationID),
	)

	return errors.Wrapf(err, "failed to unlock Translation %s", translationID)
}
