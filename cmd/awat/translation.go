package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/mattermost/awat/model"
	"github.com/spf13/cobra"
)

const (
	translationId   = "translation-id"
	installationId  = "installation-id"
	serverFlag      = "server"
	archiveFilename = "filename"
	teamFlag        = "team"
)

func init() {
	translationCmd.PersistentFlags().String(installationId, "", "ID of the installation associated with a translation")
	translationCmd.PersistentFlags().String(serverFlag, "http://localhost:8077", "The AWAT to communicate with")

	getTranslationCmd.PersistentFlags().String(translationId, "", "ID of the translation to operate on")

	startTranslationCmd.PersistentFlags().String(archiveFilename, "", "The name of the file holding the input for the translation, assumed to be stored in the root of the S3 bucket")

	startTranslationCmd.PersistentFlags().String(teamFlag, "", "The Team in Mattermost which is the intended destination of the import")

	translationCmd.AddCommand(getTranslationCmd)
	translationCmd.AddCommand(listTranslationCmd)
	translationCmd.AddCommand(startTranslationCmd)
}

var getTranslationCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch a translation from the AWAT to get its status",
	RunE: func(cmd *cobra.Command, args []string) error {

		installation, _ := cmd.Flags().GetString(installationId)
		_ = model.NewClient(server)

		translation, _ := cmd.Flags().GetString(translationId)
		_ = model.NewClient(server)

		server, _ := cmd.Flags().GetString(server)
		awat := model.NewClient(server)

		if (installation == "" && translation == "") ||
			(installation != "" && translation != "") {
			return errors.New("one and only one of translation-id or installation-id must be specified unless --work is specified")
		}

		if installation != "" {
			status, err := awat.GetTranslationStatusesByInstallation(installation)
			if len(status) > 0 {
				_ = printJSON(status)
				return err
			}
		} else {
			status, err := awat.GetTranslationStatus(translation)
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

var listTranslationCmd = &cobra.Command{
	Use:   "list",
	Short: "List all translations from the AWAT",
	RunE: func(cmd *cobra.Command, args []string) error {

		server, _ := cmd.Flags().GetString(server)
		awat := model.NewClient(server)

		var err error
		var statuses []*model.TranslationStatus
		statuses, err = awat.GetAllTranslations()

		if err != nil {
			return err
		}

		if len(statuses) == 0 {
			fmt.Println("No translations found")
			return nil
		}

		return printJSON(statuses)
	},
}

var startTranslationCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a translation",
	RunE: func(cmd *cobra.Command, args []string) error {

		server, _ := cmd.Flags().GetString(server)
		awat := model.NewClient(server)

		installation, _ := cmd.Flags().GetString(installationId)
		if installation == "" {
			return errors.New("the installation ID to which this translation pertains must be specified")
		}
		team, _ := cmd.Flags().GetString(teamFlag)
		if team == "" {
			return errors.New("the team name to which this translation pertains must be specified")
		}
		archive, _ := cmd.Flags().GetString(archiveFilename)
		if archive == "" {
			return errors.New("the archive filename to which this translation pertains must be specified")
		}

		var err error
		var status *model.TranslationStatus
		status, err = awat.CreateTranslation(
			&model.TranslationRequest{
				Type:           model.SlackWorkspaceBackupType,
				InstallationID: installation,
				Archive:        archive,
				Team:           team,
			})

		if status != nil {
			_ = printJSON(status)
		}

		return err
	},
}

var translationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Control translations on the AWAT",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)
}
