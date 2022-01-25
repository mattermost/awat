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
			"Resource",
			"Error",
		).
		From(ImportTableName)
}

// GetImport returns a single Import with identifier id
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

// GetAllImports returns all of the Imports in the database
// TODO pagination
func (sqlStore *SQLStore) GetAllImports() ([]*model.Import, error) {
	imprts := &[]*model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, imprts, importSelect)
	if err != nil {
		return nil, err
	}

	return *imprts, nil
}

// GetAndClaimNextReadyImport finds the oldest unclaimed and
// non-started Import and locks it with the given provisionerID before
// returning that Import
// Returns nil with no error if no Import is available to lock
// Returns nil, error if the Import cannot be claimed for some other reason
func (sqlStore *SQLStore) GetAndClaimNextReadyImport(provisionerID string) (*model.Import, error) {
	imports := []*model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, &imports,
		importSelect.
			Where("StartAt = 0").
			Where("CompleteAt = 0").
			Where("LockedBy = ''").
			OrderBy("CreateAt ASC").
			Limit(1),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run query to get Imports ready to run")
	}
	if len(imports) < 1 {
		return nil, nil
	}
	imprt := imports[0]

	err = sqlStore.TryLockImport(imprt, provisionerID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lock Import %s", imprt.ID)
	}

	imprt.StartAt = model.GetMillis()
	err = sqlStore.UpdateImport(imprt)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to mark Import %s as started", imprt.ID)
	}

	return imprt, nil
}

// CreateImport stores a new import.
func (sqlStore *SQLStore) CreateImport(imp *model.Import) error {
	imp.ID = model.NewID()
	imp.CreateAt = model.GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(ImportTableName).
		SetMap(map[string]interface{}{
			"ID":            imp.ID,
			"CreateAt":      imp.CreateAt,
			"StartAt":       imp.StartAt,
			"CompleteAt":    imp.CompleteAt,
			"LockedBy":      imp.LockedBy,
			"TranslationID": imp.TranslationID,
			"Resource":      imp.Resource,
			"Error":         imp.Error,
		}),
	)
	return err
}

// UpdateImport writes changes to the input Import to the database
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
			"Resource":      imp.Resource,
			"Error":         imp.Error,
		}).
		Where("ID = ?", imp.ID),
	)
	return err
}

// GetImportsByInstallation provides a convenience function for
// looking up all Imports that belong to a given Installation
func (sqlStore *SQLStore) GetImportsByInstallation(id string) ([]*model.Import, error) {
	imports := &[]*model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, imports,
		sq.Select("import.*").
			From("import").
			Join("translation ON import.translationid = translation.id").
			Where("translation.installationid = ?", id),
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return *imports, nil
}

// GetImportsByTranslation provides a convenience function for
// looking up all Imports that belong to a given Translation
func (sqlStore *SQLStore) GetImportsByTranslation(id string) ([]*model.Import, error) {
	imprts := &[]*model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, imprts,
		importSelect.Where("TranslationID = ?", id),
	)
	if err != nil {
		return nil, err
	}

	return *imprts, nil
}

func (sqlStore *SQLStore) GetImportsInProgress() ([]*model.Import, error) {
	imprts := &[]*model.Import{}
	err := sqlStore.selectBuilder(sqlStore.db, imprts,
		importSelect.
			Where("CompleteAt = 0").
			Where("StartAt != 0"),
	)
	if err != nil {
		return nil, err
	}

	return *imprts, nil

}

// TryLockImport attempts to lock the input Import with the given
// owner, but will not do so if the column already contains something,
// and will return an error instead in that case
func (sqlStore *SQLStore) TryLockImport(imp *model.Import, owner string) error {
	sqlStore.logger.Infof("Locking Import %s as %s", imp.ID, owner)
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

// UnlockImport clears the lock for the given Import
func (sqlStore *SQLStore) UnlockImport(imp *model.Import) error {
	imp.LockedBy = ""
	err := sqlStore.UpdateImport(imp)
	if err != nil {
		return errors.Wrapf(err, "failed to unlock Import %s", imp.ID)
	}
	return nil
}
