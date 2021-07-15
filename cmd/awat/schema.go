// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/mattermost/awat/internal/store"
	"github.com/spf13/cobra"
)

func init() {
	schemaCmd.AddCommand(schemaMigrateCmd)
	schemaCmd.PersistentFlags().String(databaseFlag, "postgres://localhost:5435", "The database backing the AWAT server.")
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manipulate the schema used by the provisioning server.",
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

func sqlStore(command *cobra.Command) (*store.SQLStore, error) {
	var database string
	if database = os.Getenv("AWAT_DATABASE"); database == "" {
		database, _ = command.Flags().GetString(databaseFlag)
	}
	return store.New(database, logger)
}
