package api

import "github.com/mattermost/awat/model"

type Store interface {
	GetTranslation(id string) (*model.Translation, error)
	GetTranslationsByInstallation(id string) ([]*model.Translation, error)
	GetAllTranslations() ([]*model.Translation, error)
	StoreTranslation(t *model.Translation) error
	UpdateTranslation(t *model.Translation) error

	GetAndClaimNextReadyImport(provisionerID string) (*model.Import, error)
	GetAllImports() ([]*model.Import, error)
	GetImport(id string) (*model.Import, error)
	GetImportsByInstallation(id string) ([]*model.Import, error)
}
