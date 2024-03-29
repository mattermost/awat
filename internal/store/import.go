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

// ImportTableName is the name of the database table used for storing import records.
const ImportTableName = "Import"

var importSelect sq.SelectBuilder

func init() {
	importSelect = sq.
		Select(
			"CompleteAt",
			"CreateAt",
			"ID",
			"LockedBy",
			"ImportBy",
			"StartAt",
			"TranslationID",
			"State",
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
	var imports []*model.Import
	err := sqlStore.selectBuilder(sqlStore.db, &imports,
		importSelect.
			Where("StartAt = 0").
			Where("CompleteAt = 0").
			Where("ImportBy = ''").
			Where("State = ?", model.ImportStateInProgress).
			OrderBy("CreateAt ASC").
			Limit(1),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run query to get Imports ready to run")
	}
	if len(imports) < 1 {
		return nil, nil
	}
	imp := imports[0]

	err = sqlStore.tryLockImportByProvisioner(imp, provisionerID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lock Import %s", imp.ID)
	}

	imp.StartAt = model.GetMillis()
	err = sqlStore.UpdateImport(imp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to mark Import %s as started", imp.ID)
	}

	return imp, nil
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
			"ImportBy":      imp.ImportBy,
			"TranslationID": imp.TranslationID,
			"State":         imp.State,
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
			"ImportBy":      imp.ImportBy,
			"StartAt":       imp.StartAt,
			"TranslationID": imp.TranslationID,
			"State":         imp.State,
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

// GetImportsInProgress retrieves all imports that are currently in progress.
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

// GetUnlockedImportPendingWork retrieves all imports that are pending work and not locked.
func (sqlStore *SQLStore) GetUnlockedImportPendingWork() ([]*model.Import, error) {
	var imports []*model.Import
	err := sqlStore.selectBuilder(sqlStore.db, &imports,
		importSelect.
			Where(sq.Eq{
				"State": model.AllImportStatesPendingWork,
			}).
			Where("LockedBy = ?", "").
			OrderBy("CreateAt ASC"),
	)
	if err != nil {
		return nil, err
	}
	return imports, nil
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
		}
		return errors.Errorf("wrong number of rows while trying to unlock %s", imp.ID)
	}
	return nil
}

// UnlockImport clears the lock for the given Import
func (sqlStore *SQLStore) UnlockImport(imp *model.Import) error {
	_, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(ImportTableName).
			SetMap(map[string]interface{}{"LockedBy": ""}).
			Where("ID = ?", imp.ID),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to unlock Import %s", imp.ID)
	}
	return nil
}

// tryLockImportByProvisioner attempts to lock the input Import with the given
// owner, but will not do so if the column already contains something,
// and will return an error instead in that case
func (sqlStore *SQLStore) tryLockImportByProvisioner(imp *model.Import, owner string) error {
	sqlStore.logger.Infof("Locking Import %s by %s", imp.ID, owner)
	imp.ImportBy = owner

	result, err := sqlStore.execBuilder(
		sqlStore.db, sq.
			Update(ImportTableName).
			SetMap(map[string]interface{}{"ImportBy": owner}).
			Where("ID = ?", imp.ID).
			Where("ImportBy = ?", ""),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to lock Import %s by provisioner", imp.ID)
	}
	if rows, err := result.RowsAffected(); rows != 1 || err != nil {
		if err != nil {
			return errors.Wrapf(err, "wrong number of rows while trying to unlock %s", imp.ID)
		}
		return errors.Errorf("wrong number of rows while trying to unlock %s", imp.ID)
	}
	return nil
}
