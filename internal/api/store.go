// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import "github.com/mattermost/awat/model"

type Store interface {
	GetTranslation(id string) (*model.Translation, error)
	GetTranslationsByInstallation(id string) ([]*model.Translation, error)
	GetAllTranslations() ([]*model.Translation, error)
	CreateTranslation(t *model.Translation) error
	UpdateTranslation(t *model.Translation) error

	GetAndClaimNextReadyImport(provisionerID string) (*model.Import, error)
	GetAllImports() ([]*model.Import, error)
	GetImport(id string) (*model.Import, error)
	GetImportsInProgress() ([]*model.Import, error)
	GetImportsByInstallation(id string) ([]*model.Import, error)
	GetImportsByTranslation(id string) ([]*model.Import, error)
	UpdateImport(imp *model.Import) error

	GetUpload(id string) (*model.Upload, error)
	CreateUpload(id string) error
	CompleteUpload(uploadID, errorMessage string) error
}
