package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/awat/model"
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

// GetTranslationsReadyToStart returns a batch of Translations that
// are ready to go, with a maximum of ten, sorted from oldest to
// newest
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
			"CreateAt":   model.Timestamp(),
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
			"CreateAt":   translation.CreateAt,
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

func (sqlStore *SQLStore) TryLockTranslation(translation *model.Translation, owner string) error {
	sqlStore.logger.Infof("locking %s as %s", translation.ID, owner)
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
		} else {
			return errors.Errorf("wrong number of rows while trying to unlock %s", translation.ID)
		}
	}
	return nil
}

func (sqlStore *SQLStore) UnlockTranslation(translation *model.Translation) error {
	translation.LockedBy = ""
	err := sqlStore.UpdateTranslation(translation)
	if err != nil {
		return errors.Wrapf(err, "failed to unlock Translation %s", translation.ID)
	}
	return nil
}
