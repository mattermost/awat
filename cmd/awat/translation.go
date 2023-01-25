// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/awat/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	translationID       = "translation-id"
	installationID      = "installation-id"
	archiveFilename     = "filename"
	teamFlag            = "team"
	translationTypeFlag = "type"
	uploadFile          = "upload"
)

func init() {
	translationCmd.PersistentFlags().String(installationID, "", "ID of the installation associated with a translation")
	translationCmd.PersistentFlags().String(serverFlag, "http://localhost:8077", "The AWAT to communicate with")

	getTranslationCmd.PersistentFlags().String(translationID, "", "ID of the translation to operate on")

	startTranslationCmd.PersistentFlags().String(archiveFilename, "", "The name of the file holding the input for the translation, assumed to be stored in the root of the S3 bucket")
	startTranslationCmd.PersistentFlags().String(teamFlag, "", "The Team in Mattermost which is the intended destination of the import")
	startTranslationCmd.PersistentFlags().String(translationTypeFlag, string(model.SlackWorkspaceBackupType), "The type of backup being translated & imported (default: slack; valid options: mattermost, slack)")

	startTranslationCmd.PersistentFlags().Bool(uploadFile, false, "Whether or not to upload the file provided before proceeding")

	translationCmd.AddCommand(getTranslationCmd)
	translationCmd.AddCommand(listTranslationCmd)
	translationCmd.AddCommand(startTranslationCmd)
}

var translationCmd = &cobra.Command{
	Use:   "translation",
	Short: "Control translations on the AWAT",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var getTranslationCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch a translation from the AWAT to get its status",
	RunE: func(cmd *cobra.Command, args []string) error {

		installation, _ := cmd.Flags().GetString(installationID)
		translation, _ := cmd.Flags().GetString(translationID)

		server, _ := cmd.Flags().GetString(serverFlag)
		awat := model.NewClient(server)

		if (installation == "" && translation == "") ||
			(installation != "" && translation != "") {
			return errors.New("one and only one of translation-id or installation-id must be specified")
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
		server, _ := cmd.Flags().GetString(serverFlag)
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

		server, _ := cmd.Flags().GetString(serverFlag)
		awat := model.NewClient(server)

		translationTypeString, _ := cmd.Flags().GetString(translationTypeFlag)
		translationType := model.BackupType(translationTypeString)
		if translationType != model.MattermostWorkspaceBackupType &&
			translationType != model.SlackWorkspaceBackupType {
			return errors.Errorf("unknown Translation type %q provided", translationType)
		}

		installation, _ := cmd.Flags().GetString(installationID)
		if installation == "" {
			return errors.New("the installation ID to which this translation pertains must be specified")
		}
		team, _ := cmd.Flags().GetString(teamFlag)
		if team == "" && translationType != model.MattermostWorkspaceBackupType {
			// Mattermost backups include their team names, but other types don't
			return errors.New("the team name to which this translation pertains must be specified")
		}
		archive, _ := cmd.Flags().GetString(archiveFilename)
		if archive == "" {
			return errors.New("the archive filename to which this translation pertains must be specified")
		}

		var err error
		var uploadID string
		upload, _ := cmd.Flags().GetBool(uploadFile)
		if upload {
			archive, err = awat.UploadArchiveForTranslation(archive, translationType)
			if err != nil {
				return errors.Wrapf(err, "failed to upload %s", archive)
			}

			uploadID = strings.TrimSuffix(archive, ".zip")

			if err = awat.WaitForUploadToComplete(uploadID); err != nil {
				return errors.Wrapf(err, "failed to upload %s", archive)
			}
		}

		var status *model.TranslationStatus
		status, err = awat.CreateTranslation(
			&model.TranslationRequest{
				Type:           translationType,
				InstallationID: installation,
				Archive:        archive,
				UploadID:       &uploadID,
				Team:           team,
			})

		if status != nil {
			_ = printJSON(status)
		}

		return err
	},
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)
}
