package main

import (
	"github.com/mattermost/awat/internal/model"
	"github.com/spf13/cobra"
)

var getTranslationCmd = &cobra.Command{
	Use:   "awat translation get",
	Short: "Fetch ongoing translations on the AWAT to get their status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var translationCmd = &cobra.Command{
	Use:   "awat translation",
	Short: "Control translations on the AWAT",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString("server")
		_ = model.NewClient(server)
		return nil
	},
}

func init() {
	translationCmd.AddCommand(getTranslationCmd)
	translationCmd.PersistentFlags().String("server", "http://localhost:8077", "The AWAT to communicate with")
}
