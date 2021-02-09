package store

import (
	"github.com/mattermost/workspace-translator/internal/model"
)

func (s *SQLStore) GetTransaction(id string) (*model.Transaction, error) {
	return nil, nil
}

func (s *SQLStore) StoreTransaction(transaction *model.Transaction) error {
	return nil
}

func (s *SQLStore) UpdateTransaction(transaction *model.Transaction) (*model.Transaction, error) {
	return nil, nil
}
