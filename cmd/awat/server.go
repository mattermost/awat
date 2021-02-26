package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mattermost/awat/internal/api"
	"github.com/mattermost/awat/internal/supervisor"
	cloudModel "github.com/mattermost/mattermost-cloud/model"
)

const (
	databaseFlag = "database"
	listenFlag   = "listen"
	bucket       = "bucket"
	server       = "server"
)

func init() {
	serverCmd.PersistentFlags().String(listenFlag, "localhost:8077", "Local interface and port to listen on")
	serverCmd.PersistentFlags().String(bucket, "", "S3 URI where the input can be found and to which the output can be written")
	serverCmd.PersistentFlags().String(databaseFlag, "postgres://localhost:5435", "Location of a Postgres database for the server to use")
	serverCmd.MarkPersistentFlagRequired(bucket)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the AWAT server.",
	RunE: func(command *cobra.Command, args []string) error {

		logger.SetLevel(logrus.DebugLevel) // TODO add a flag for this
		logger.Debug("debug level")

		listen, _ := command.Flags().GetString(listenFlag)
		if listen == "" {
			return fmt.Errorf("the server command requires the --listen flag not be empty")
		}

		sqlStore, err := sqlStore(command)
		if err != nil {
			return err
		}

		supervisor := supervisor.NewSupervisor(sqlStore, bucket)
		supervisor.Start()

		router := mux.NewRouter()
		api.Register(router, &api.Context{
			Store:     sqlStore,
			Logger:    logger,
			RequestID: cloudModel.NewID(),
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
