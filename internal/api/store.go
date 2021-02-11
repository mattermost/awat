package api

import "github.com/mattermost/awat/internal/model"

type Store interface {
	GetTranslation(id string) (*model.Translation, error)
	GetTranslationByInstallation(id string) (*model.Translation, error)
	StoreTranslation(t *model.Translation) error
	UpdateTranslation(t *model.Translation) (*model.Translation, error)
}
