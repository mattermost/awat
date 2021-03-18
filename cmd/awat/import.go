package main

import (
	"errors"
	"fmt"

	"github.com/mattermost/awat/model"
	"github.com/spf13/cobra"
)

const importID = "import-id"

func init() {
	importCmd.PersistentFlags().String(serverFlag, "http://localhost:8077", "The AWAT to communicate with")

	getImportCmd.PersistentFlags().String(importID, "", "ID of the Import to operate on")
	getImportCmd.PersistentFlags().String(installationId, "", "ID of the installation associated with an import")

	importCmd.AddCommand(getImportCmd)
	importCmd.AddCommand(listImportCmd)
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Control imports with the AWAT",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var getImportCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch an import by ID from the AWAT to get its status",
	RunE: func(cmd *cobra.Command, args []string) error {
		imprt, _ := cmd.Flags().GetString(importID)
		server, _ := cmd.Flags().GetString(serverFlag)
		_ = model.NewClient(server)

		installation, _ := cmd.Flags().GetString(installationId)
		awat := model.NewClient(server)
		if (installation == "" && imprt == "") || (installation != "" && imprt != "") {
			return errors.New("one and only one of translation-id or import-id must be specified")
		}
		if installation != "" {
			status, err := awat.GetImportStatusesByInstallation(installation)
			if len(status) > 0 {
				_ = printJSON(status)
			}
			return err
		} else {
			status, err := awat.GetImportStatus(imprt)
			if err != nil {
				return err
			}

			if status == nil {
				fmt.Println("No translations found")
			}

			if status != nil {
				_ = printJSON(status)
			}
		}

		return nil
	},
}

var listImportCmd = &cobra.Command{
	Use:   "list",
	Short: "List multiple imports from the AWAT",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString(serverFlag)
		awat := model.NewClient(server)

		var err error
		var statuses []*model.ImportStatus
		statuses, err = awat.ListImports()

		if err != nil {
			return err
		}

		if len(statuses) == 0 {
			fmt.Println("No imports found")
			return nil
		}

		return printJSON(statuses)
	},
}
