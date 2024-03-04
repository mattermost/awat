// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/awat/model"
	"github.com/spf13/cobra"
)

const (
	uploadID = "upload-id"
)

func init() {
	uploadCmd.PersistentFlags().String(serverFlag, "http://localhost:8077", "The AWAT to communicate with")

	getUploadCmd.PersistentFlags().String(uploadID, "", "ID of the upload to get")
	getUploadCmd.MarkPersistentFlagRequired(uploadID)

	uploadCmd.AddCommand(getUploadCmd)
	uploadCmd.AddCommand(getUploadsCmd)
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Commands for reviewing upload objects",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

var getUploadCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch an upload from the AWAT",
	RunE: func(cmd *cobra.Command, args []string) error {
		uploadID, _ := cmd.Flags().GetString(uploadID)

		server, _ := cmd.Flags().GetString(serverFlag)
		client := model.NewClient(server)

		upload, err := client.GetUpload(uploadID)
		if err != nil {
			return err
		}

		return printJSON(upload)
	},
}

var getUploadsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all uploads from the AWAT",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, _ := cmd.Flags().GetString(serverFlag)
		client := model.NewClient(server)

		statuses, err := client.GetUploads()
		if err != nil {
			return err
		}

		return printJSON(statuses)
	},
}
