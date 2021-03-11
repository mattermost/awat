package store

import (
	"database/sql"
	"fmt"

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
			"Output",
			"Resource",
			"StartAt",
			"ImportStartAt",
			"ImportCompleteAt",
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

func (sqlStore *SQLStore) GetTranslationReadyToImport(provisionerID string) (*model.Translation, error) {
	translation := new(model.Translation)
	const tries = 5

	for count := 0; count < tries; count++ {
		err := sqlStore.getBuilder(sqlStore.db,
			translation,
			translationSelect.
				Where("LockedBy = ''").
				OrderBy("CompleteAt DESC").
				Limit(1),
		)

		if err == sql.ErrNoRows {
			// success, but nothing needs to be worked on
			return nil, nil
		}

		if err != nil {
			// db communication error
			return nil, errors.Wrap(err, "failed to get translation needing work")
		}

		// try to update the translation
		translation.LockedBy = provisionerID
		_, err = sqlStore.execBuilder(sqlStore.db, sq.
			Update(TranslationTableName).
			SetMap(map[string]interface{}{
				"CompleteAt":       translation.CompleteAt,
				"Error":            translation.Error,
				"ID":               translation.ID,
				"InstallationID":   translation.InstallationID,
				"LockedBy":         translation.LockedBy,
				"Resource":         translation.Resource,
				"Output":           translation.Output,
				"StartAt":          translation.StartAt,
				"Team":             translation.Team,
				"Type":             translation.Type,
				"ImportStartAt":    translation.ImportStartAt,
				"ImportCompleteAt": translation.ImportCompleteAt,
			}).
			Where("ID = ?", translation.ID).
			Where("LockedBy = ?", ""), // be sure nobody else locked it between our queries
		)

		if err == nil {
			return translation, nil
		}

		// if there's an error here, it's almost definitely a race condition. Try again.
		sqlStore.logger.WithError(err).Warning("failed to claim translation %s")
	}

	return nil, fmt.Errorf("failed to claim any translation despite finding some that need work after %d tries", tries)
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
			"CompleteAt":       translation.CompleteAt,
			"Error":            translation.Error,
			"ID":               translation.ID,
			"InstallationID":   translation.InstallationID,
			"LockedBy":         translation.LockedBy,
			"Resource":         translation.Resource,
			"StartAt":          translation.StartAt,
			"Team":             translation.Team,
			"Output":           translation.Output,
			"Type":             translation.Type,
			"ImportStartAt":    translation.ImportStartAt,
			"ImportCompleteAt": translation.ImportCompleteAt,
		}),
	)
	return err
}

func (sqlStore *SQLStore) UpdateTranslation(translation *model.Translation) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(TranslationTableName).
		SetMap(map[string]interface{}{
			"CompleteAt":       translation.CompleteAt,
			"Error":            translation.Error,
			"ID":               translation.ID,
			"InstallationID":   translation.InstallationID,
			"LockedBy":         translation.LockedBy,
			"Resource":         translation.Resource,
			"Output":           translation.Output,
			"StartAt":          translation.StartAt,
			"Team":             translation.Team,
			"Type":             translation.Type,
			"ImportStartAt":    translation.ImportStartAt,
			"ImportCompleteAt": translation.ImportCompleteAt,
		}).Where("ID = ?", translation.ID),
	)
	return err
}
