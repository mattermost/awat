package store

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	// only run these tests if they're explicitly enabled, or in CI
	database := os.Getenv("AWAT_DATABASE")
	if database == "" {
		t.Skip()
	}

	sqlStore, err := New(database, log.New())
	require.NoError(t, err)

	err = sqlStore.Migrate()
	require.NoError(t, err)
}
