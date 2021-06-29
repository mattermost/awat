// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "awat",
	Long:  "The Automatic Workspace Archive Translator is a microservice that converts workspaces from various formats into the Mattermost Bulk Import Format such that they can be imported into Mattermost Cloud",
	Short: "The Automatic Workspace Archive Translator",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return serverCmd.RunE(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(translationCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(schemaCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
