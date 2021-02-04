package main

import (
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the AWAT server.",
	RunE: func(command *cobra.Command, args []string) error {

		router := mux.NewRouter()

		api.Register(router)
		return nil
	},
}
