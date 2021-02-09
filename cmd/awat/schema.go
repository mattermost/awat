package main

import (
	"github.com/mattermost/workspace-translator/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	schemaCmd.AddCommand(schemaMigrateCmd)
	schemaCmd.PersistentFlags().String("database", "postgres://cloud.db", "The database backing the AWAT server.")
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manipulate the schema used by the provisioning server.",
}

func sqlStore(command *cobra.Command) (*store.SQLStore, error) {
	database, _ := command.Flags().GetString("database")
	sqlStore, err := store.New(database, logger)
	if err != nil {
		return nil, err
	}

	return sqlStore, nil
}

var schemaMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate the schema to the latest supported version.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		return sqlStore.Migrate()
	},
}
