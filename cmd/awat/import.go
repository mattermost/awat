package main

import (
	"errors"
	"fmt"

	"github.com/mattermost/awat/model"
	"github.com/spf13/cobra"
)

const id = "id"

func init() {
	importCmd.PersistentFlags().String(serverFlag, "http://localhost:8077", "The AWAT to communicate with")
	importCmd.AddCommand(getImportCmd)
	importCmd.AddCommand(listImportCmd)
	getImportCmd.PersistentFlags().String(id, "", "ID of the item by which to select Imports")

	getImportCmd.AddCommand(getImportByIDCmd)
	getImportCmd.AddCommand(getImportByTranslationCmd)
	getImportCmd.AddCommand(getImportByInstallationCmd)
}

var getImportCmd = &cobra.Command{
	Use:   "get",
	Short: "Get Imports",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Control imports with the AWAT",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var getImportByTranslationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Get Imports by the Translation ID to which they correlate",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString(serverFlag)
		translation, _ := cmd.Flags().GetString(id)
		awat := model.NewClient(server)
		if translation == "" {
			return errors.New("Must provide a Translation ID")
		}
		statuses, err := awat.GetImportStatusesByTranslation(translation)
		if err != nil {
			return err
		}
		if len(statuses) > 0 {
			_ = printJSON(statuses)
			return nil
		}
		fmt.Println("No Imports found.")
		return nil
	},
}

var getImportByIDCmd = &cobra.Command{
	Use:   "import",
	Short: "Get an Import by its ID",
	RunE: func(cmd *cobra.Command, args []string) error {
		imprt, _ := cmd.Flags().GetString(id)
		server, _ := cmd.Flags().GetString(serverFlag)
		awat := model.NewClient(server)
		status, err := awat.GetImportStatus(imprt)
		if err != nil {
			return err
		}

		if status == nil {
			fmt.Printf("No Import found with ID %s", imprt)
		}

		if status != nil {
			_ = printJSON(status)
		}

		return nil
	},
}

var getImportByInstallationCmd = &cobra.Command{
	Use:   "installation",
	Short: "Get the translations which correlate to the given Installation",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString(serverFlag)
		installation, _ := cmd.Flags().GetString(id)
		awat := model.NewClient(server)
		if installation == "" {
			return errors.New("Must provide an Installation ID")
		}
		status, err := awat.GetImportStatusesByInstallation(installation)
		if len(status) > 0 {
			_ = printJSON(status)
		}
		return err
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
