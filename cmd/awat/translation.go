package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

var translationCmd = &cobra.Command{
	Use:        "awat translate",
	Short:      "Control translations on the AWAT",
	ArgAliases: []string{},
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")

		client := model.NewClient(serverAddress)
		return nil
	},
}
