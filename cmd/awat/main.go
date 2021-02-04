package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "awat",
	Long:  "The Automatic Workspace Archive Translator is a microservice that converts workspaces from various formats into the Mattermost Bulk Import Format such that they can be imported into Mattermost Cloud",
	Short: "The Automatic Workspace Archive Translator",
	RunE: func(cmd *cobra.Command, args []string) error {
		return serverCmd.RunE(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().String("bucket", "", "S3 URI where the input can be found and to which the output can be written")

	serverCmd.PersistentFlags().String("listen", "localhost:8077", "Local interface and port to listen on")

	rootCmd.MarkFlagRequired("bucket")

	serverCmd.MarkFlagRequired("listen")
	rootCmd.MarkFlagRequired("input")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(translateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
