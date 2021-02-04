package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/mattermost/workspace-translator/internal/slack"
)

var rootCmd = &cobra.Command{
	Use:   "awat",
	Short: "The Automatic Workspace Archive Translator is a microservice that converts workspaces from various formats into the Mattermost Bulk Import Format such that they can be imported into Mattermost Cloud",
	RunE: func(cmd *cobra.Command, args []string) error {
		bucket, _ := cmd.Flags().GetString("bucket")
		return slack.FetchAttachments("/home/ian/Downloads/customer_slack_backup_deleteme/Intra.zip", bucket)
		// return serverCmd.RunE(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().String("bucket", "", "S3 URI where the input can be found and to which the output can be written")
	rootCmd.MarkFlagRequired("bucket")
	rootCmd.AddCommand(serverCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
