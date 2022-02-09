// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/mattermost/awat/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type ImportSupervisor struct {
	logger         log.FieldLogger
	store          importStore
	cloud          *cloud.Client
	awsConfig      *aws.Config
	bucket         string
	keepImportData bool
}

type importStore interface {
	GetImportsInProgress() ([]*model.Import, error)
	GetTranslation(id string) (*model.Translation, error)
	UpdateImport(imp *model.Import) error
}

func NewImportSupervisor(store importStore, logger log.FieldLogger, cloudClient *cloud.Client, bucket string, keepImportData bool) *ImportSupervisor {
	return &ImportSupervisor{
		logger:         logger,
		store:          store,
		cloud:          cloudClient,
		bucket:         bucket,
		keepImportData: keepImportData,
	}
}

func (s *ImportSupervisor) Start() {
	s.logger.Info("Import supervisor started")
	go func() {
		for {
			s.supervise()
			time.Sleep(30 * time.Second) // TODO: make this configurable
		}
	}()
}

func (s *ImportSupervisor) supervise() {
	imports, err := s.store.GetImportsInProgress()
	if err != nil {
		s.logger.WithError(err).Error("Failed to look up ongoing Imports to supervise")
		return
	}

	for _, i := range imports {
		translation, err := s.store.GetTranslation(i.TranslationID)
		if err != nil {
			s.logger.WithError(err).Errorf("Failed to look up Translation %s", i.TranslationID)
			continue
		}

		logger := s.logger.WithFields(log.Fields{"translation": translation.ID, "installation": translation.InstallationID})

		installation, err := s.cloud.GetInstallation(
			translation.InstallationID,
			&cloud.GetInstallationRequest{
				IncludeGroupConfig:          false,
				IncludeGroupConfigOverrides: false,
			})
		if err != nil {
			logger.WithError(err).Error("Failed to fetch installation")
			continue
		}
		if installation == nil {
			logger.Error("Installation not found")
			continue
		}
		if !startedImportIsComplete(installation, i) {
			logger.Debug("Import is still running")
			continue
		}

		i.CompleteAt = model.GetMillis()
		err = s.store.UpdateImport(i)
		if err != nil {
			logger.WithError(err).Error("Failed to mark import as completed")
			return
		}

		logger.Info("Import completed")

		if s.keepImportData {
			logger.Debug("Skipping import bundle cleanup")
			return
		}

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			logger.WithError(err).Error("Failed to load AWS config")
			return
		}

		key := fmt.Sprintf("%s.zip", i.TranslationID)
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelFunc()

		client := s3.NewFromConfig(cfg)
		_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: &s.bucket,
			Key:    &key,
		})
		if err != nil {
			logger.WithError(err).Error("Failed to delete translation from S3")
			return
		}

		logger.Debug("Import cleanup completed successfully")
	}
}

// startedImportIsComplete returns true if an Import with a nonzero
// StartAt value has been completed, and false otherwise.
func startedImportIsComplete(installation *cloud.InstallationDTO, i *model.Import) bool {
	switch {
	case
		// go ahead and mark Imports against Deleted Installations as
		// complete
		installation.State == cloud.InstallationStateDeleted:
	case
		installation.State == cloud.InstallationStateImportComplete:
	default:
		return false
	}
	return true
}
