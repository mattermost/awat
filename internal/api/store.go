package api

import "github.com/mattermost/awat/internal/model"

type Store interface {
	GetTranslation(id string) (*model.Translation, error)
	GetTranslationsByInstallation(id string) ([]*model.Translation, error)
	GetAllTranslations() ([]*model.Translation, error)
	StoreTranslation(t *model.Translation) error
	UpdateTranslation(t *model.Translation) error

	GetNextReadyImport(provisionerID string) (*model.Import, error)
}
