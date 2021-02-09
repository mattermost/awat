package api

import "github.com/mattermost/workspace-translator/internal/model"

type Store interface {
	GetTransaction(id string) (*model.Transaction, error)
	StoreTransaction(t *model.Transaction) error
	UpdateTransaction(t *model.Transaction) (*model.Transaction, error)
}
