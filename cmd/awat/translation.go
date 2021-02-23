package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/mattermost/awat/internal/model"
	"github.com/spf13/cobra"
)

func init() {
	translationCmd.PersistentFlags().String(translationId, "", "ID of the translation to operate on")
	translationCmd.PersistentFlags().String(installationId, "", "ID of the Installation associated with a translation")
	translationCmd.PersistentFlags().String(serverFlag, "http://localhost:8077", "The AWAT to communicate with")

	translationCmd.AddCommand(getTranslationCmd)
	translationCmd.AddCommand(listTranslationCmd)
	translationCmd.AddCommand(startTranslationCmd)

	startTranslationCmd.MarkPersistentFlagRequired(installationId)
	startTranslationCmd.MarkPersistentFlagRequired(archiveFilename)
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)

}

const (
	translationId   = "translation-id"
	installationId  = "installation-id"
	serverFlag      = "server"
	archiveFilename = "filename"
)

var getTranslationCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch a translation from the AWAT to get its status",
	RunE: func(cmd *cobra.Command, args []string) error {

		installation, _ := cmd.Flags().GetString(installationId)
		_ = model.NewClient(server)

		translation, _ := cmd.Flags().GetString(translationId)
		_ = model.NewClient(server)

		if (installation == "" && translation == "") ||
			(installation != "" && translation != "") {
			return errors.New("one and only one of translation-id or installation-id must be specified")
		}

		server, _ := cmd.Flags().GetString(server)
		awat := model.NewClient(server)

		var err error
		var status *model.TranslationStatus
		if installation != "" {
			status, err = awat.GetTranslationStatusByInstallation(installation)
		} else {
			status, err = awat.GetTranslationStatus(translation)
		}

		if status == nil {
			fmt.Println("No translations found")
		}

		if status != nil {
			_ = printJSON(status)
		}

		return err
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
		archive, _ := cmd.Flags().GetString(archiveFilename)

		var err error
		var status *model.TranslationStatus
		status, err = awat.CreateTranslation(
			&model.TranslationRequest{
				Type:           model.SlackWorkspaceBackupType,
				InstallationID: installation,
				Archive:        archive,
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
