// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/awat/internal/api"
	"github.com/mattermost/awat/internal/store"
	"github.com/mattermost/awat/internal/supervisor"
	"github.com/mattermost/awat/model"
	cmodel "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	databaseFlag         = "database"
	listenFlag           = "listen"
	bucketFlag           = "bucket"
	workingDirectoryFlag = "workdir"
	serverFlag           = "server"
	provisionerFlag      = "provisioner"
	debugFlag            = "debug"
	keepImportDataFlag   = "keep-import-data"
)

func init() {
	serverCmd.PersistentFlags().String(listenFlag, "localhost:8077", "Local interface and port to listen on")
	serverCmd.PersistentFlags().String(bucketFlag, "", "S3 URI where the input can be found")
	serverCmd.PersistentFlags().String(workingDirectoryFlag, "/tmp/awat/workdir", "The directory to which attachments can be fetched and where the input can be extracted. In production, this will contain the location where the EBS volume is mounted.")
	serverCmd.PersistentFlags().String(databaseFlag, "postgres://localhost:5435", "Location of a Postgres database for the server to use")
	serverCmd.PersistentFlags().String(provisionerFlag, "http://localhost:8075", "Address of the Provisioner")
	serverCmd.PersistentFlags().Bool(keepImportDataFlag, true, "Whether to preserve import bundles after import completion or not")
	serverCmd.PersistentFlags().Bool(debugFlag, true, "Whether to output debug logs")
	_ = serverCmd.MarkPersistentFlagRequired(bucketFlag)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the AWAT server.",
	RunE: func(command *cobra.Command, args []string) error {

		debug, _ := command.Flags().GetBool(debugFlag)
		if debug {
			logger.SetLevel(logrus.DebugLevel)
		}

		listen, _ := command.Flags().GetString(listenFlag)
		if listen == "" {
			return errors.New("the server command requires the --listen flag not be empty")
		}

		workdir, _ := command.Flags().GetString(workingDirectoryFlag)
		if workdir == "" {
			return errors.New("the server command requires the --workdir flag not be empty")
		}
		_, err := os.Stat(workdir)
		if err != nil {
			if os.IsNotExist(err) {
				// this message might wind up being somewhat redundant but that's alright
				return errors.Wrapf(err, "the provided path for the working directory \"%s\" does not exist. Create it and try again?", workdir)
			}
			return errors.Wrapf(err, "failed to check status of working directory \"%s\"", workdir)

		}

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		currentVersion, err := sqlStore.GetCurrentVersion()
		if err != nil {
			return err
		}
		serverVersion := store.LatestVersion()

		// Require the schema to be at least the server version, and also the same major
		// version.
		if currentVersion.LT(serverVersion) || currentVersion.Major != serverVersion.Major {
			return errors.Errorf("server requires at least schema %s, current is %s", serverVersion, currentVersion)
		}

		bucket, _ := command.Flags().GetString(bucketFlag)
		provisionerURL, _ := command.Flags().GetString(provisionerFlag)
		keepImportData, _ := command.Flags().GetBool(keepImportDataFlag)

		logger.WithFields(logrus.Fields{
			"build-hash":         model.BuildHash,
			provisionerFlag:      provisionerURL,
			bucketFlag:           bucket,
			workingDirectoryFlag: workdir,
			keepImportDataFlag:   keepImportData,
			debugFlag:            debug,
		}).Info("Starting AWAT Server")

		cloud, err := buildCloudClientAndCheckConnectivity(provisionerURL)
		if err != nil {
			return err
		}

		awsContext, err := api.NewAWSContext(bucket)
		if err != nil {
			return err
		}

		translationSupervisor := supervisor.NewTranslationSupervisor(sqlStore, logger, bucket, workdir)
		translationSupervisor.Start()

		importSupervisor := supervisor.NewImportSupervisor(sqlStore, logger, cloud, bucket, keepImportData)
		go importSupervisor.Start()

		router := mux.NewRouter()
		api.Register(router,
			&api.Context{
				Store:   sqlStore,
				Logger:  logger,
				AWS:     awsContext,
				Workdir: workdir,
			})

		srv := &http.Server{
			Addr:           listen,
			Handler:        router,
			ReadTimeout:    180 * time.Second,
			WriteTimeout:   180 * time.Second,
			IdleTimeout:    time.Second * 180,
			MaxHeaderBytes: 1 << 20,
		}

		go func() {
			logger.WithField("addr", srv.Addr).Info("Listening")
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Failed to listen and serve")
			}
		}()

		c := make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via:
		//  - SIGINT (Ctrl+C)
		//  - SIGTERM (Ctrl+/) (Kubernetes pod rolling termination)
		// SIGKILL and SIGQUIT will not be caught.
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		sig := <-c
		logger.WithField("shutdown-signal", sig.String()).Info("Shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	},
}

func buildCloudClientAndCheckConnectivity(provisionerURL string) (*cmodel.Client, error) {
	cloudClient := cmodel.NewClient(provisionerURL)
	_, err := cloudClient.GetInstallationsCount(false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check provisioner connectivity")
	}

	return cloudClient, nil
}
