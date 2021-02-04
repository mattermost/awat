package main

import "github.com/spf13/cobra"

var translateCmd = &cobra.Command{
	Use:        "awat translate",
	Short:      "Control translations on the AWAT",
	ArgAliases: []string{},
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
