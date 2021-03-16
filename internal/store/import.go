package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/awat/model"
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

func (sqlStore *SQLStore) GetImport(id string) (*model.Import, error) {
	imprt := new(model.Import)
	builder := importSelect
	builder = builder.Where("ID = ?", id)

	err := sqlStore.getBuilder(sqlStore.db, imprt, builder)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to get import by id")
	}

	return imprt, nil
}

func (sqlStore *SQLStore) GetAllImports() ([]*model.Import, error) {
	imprts := &[]*model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, imprts, importSelect)
	if err != nil {
		return nil, err
	}

	return *imprts, nil
}

func (sqlStore *SQLStore) GetAndClaimNextReadyImport(provisionerID string) (*model.Import, error) {
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

	err = sqlStore.TryLockImport(imprt, provisionerID)
	if err != nil {
		return nil, err
	}

	imprt.StartAt = model.Timestamp()
	err = sqlStore.UpdateImport(imprt)
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

func (sqlStore *SQLStore) GetImportsByInstallation(id string) ([]*model.Import, error) {
	imports := &[]*model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, imports,
		importSelect.
			Where("InstallationID = ?", id),
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return *imports, nil
}

func (sqlStore *SQLStore) TryLockImport(imp *model.Import, owner string) error {
	sqlStore.logger.Infof("locking %s as %s", imp.ID, owner)
	imp.LockedBy = owner

	result, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(ImportTableName).
			SetMap(map[string]interface{}{"LockedBy": owner}).
			Where("ID = ?", imp.ID).
			Where("LockedBy = ?", ""),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to lock Import %s", imp.ID)
	}
	if rows, err := result.RowsAffected(); rows != 1 || err != nil {
		if err != nil {
			return errors.Wrapf(err, "wrong number of rows while trying to unlock %s", imp.ID)
		} else {
			return errors.Errorf("wrong number of rows while trying to unlock %s", imp.ID)
		}
	}
	return nil
}

func (sqlStore *SQLStore) UnlockImport(imp *model.Import) error {
	imp.LockedBy = ""
	err := sqlStore.UpdateImport(imp)
	if err != nil {
		return errors.Wrapf(err, "failed to unlock Import %s", imp.ID)
	}
	return nil
}
