package store

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/awat/internal/model"
	"github.com/pkg/errors"
)

const ImportTableName = "Import"

var importSelect sq.SelectBuilder

func init() {
	importSelect = sq.
		Select(
			"CompleteAt",
			"CreateAt",
			"ID",
			"LockedBy",
			"StartAt",
			"TranslationID",
		).
		From(ImportTableName)
}

func (sqlStore *SQLStore) GetNextReadyImport(provisionerID string) (*model.Import, error) {
	imprt := &model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, imprt,
		importSelect.
			Where("StartAt = 0").
			Where("LockedBy = ''").
			OrderBy("CreateAt ASC").
			Limit(1),
	)

	if err != nil {
		return nil, err
	}

	return imprt, nil

}

func (sqlStore *SQLStore) StoreImport(imp *model.Import) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(ImportTableName).
		SetMap(map[string]interface{}{
			"CreateAt":      model.Timestamp(),
			"CompleteAt":    imp.CompleteAt,
			"ID":            imp.ID,
			"LockedBy":      imp.LockedBy,
			"StartAt":       imp.StartAt,
			"TranslationID": imp.TranslationID,
		}),
	)
	return err
}
func (sqlStore *SQLStore) UpdateImport(imp *model.Import) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(ImportTableName).
		SetMap(map[string]interface{}{
			"CreateAt":      imp.CreateAt,
			"CompleteAt":    imp.CompleteAt,
			"ID":            imp.ID,
			"LockedBy":      imp.LockedBy,
			"StartAt":       imp.StartAt,
			"TranslationID": imp.TranslationID,
		}).
		Where("ID = ?", imp.ID),
	)
	return err
}

func (sqlStore *SQLStore) TryLockImport(importID string, owner string) error {
	_, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(ImportTableName).
			SetMap(map[string]interface{}{
				"LockedBy": owner,
			}).
			Where("ID = ?", importID).
			Where("LockedBy = ?", ""),
	)

	if err != nil {
		return errors.Wrapf(err, "failed to lock Import %s", importID)
	}
	return nil
}

func (sqlStore *SQLStore) UnlockImport(importID string) error {
	_, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(ImportTableName).
			SetMap(map[string]interface{}{
				"LockedBy": "",
			}).
			Where("ID = ?", importID),
	)

	if err != nil {
		return errors.Wrapf(err, "failed to unlock Import %s", importID)
	}
	return nil
}
