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
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ImportSupervisor struct {
	logger    log.FieldLogger
	store     importStore
	cloud     *cloud.Client
	bucket    string
	awsConfig *aws.Config
}

type importStore interface {
	GetImportsInProgress() ([]*model.Import, error)
	GetTranslation(id string) (*model.Translation, error)
	UpdateImport(imp *model.Import) error
}

func NewImportSupervisor(store importStore, logger log.FieldLogger, bucket, provisionerURL string) *ImportSupervisor {
	return &ImportSupervisor{
		logger: logger,
		store:  store,
		cloud:  cloud.NewClient(provisionerURL),
		bucket: bucket,
	}
}

func (s *ImportSupervisor) Start() {
	s.logger.Info("Import supervisor started")
	go func() {
		for {
			err := s.supervise()
			if err != nil {
				s.logger.WithError(err).Error("failed an operation while supervising imports")
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

func (s *ImportSupervisor) supervise() error {
	imports, err := s.store.GetImportsInProgress()
	if err != nil {
		return errors.Wrap(err, "failed to look up ongoing Imports to supervise")
	}

	for _, i := range imports {
		translation, err := s.store.GetTranslation(i.TranslationID)
		if err != nil {
			s.logger.WithError(err).Warnf("failed to look up Translation %s", i.TranslationID)
			continue
		}

		installation, err := s.cloud.GetInstallation(
			translation.InstallationID,
			&cloud.GetInstallationRequest{
				IncludeGroupConfig:          false,
				IncludeGroupConfigOverrides: false,
			})
		if err != nil {
			s.logger.WithError(err).Warnf("failed to fetch information on Installation %s", translation.InstallationID)
			continue
		}
		if installation == nil {
			s.logger.WithError(errors.New("Installation not found")).Warnf("Installation with ID %s not found but no error was returned from the Provisioner", translation.InstallationID)
			continue
		}

		if startedImportIsComplete(installation, i) {
			i.CompleteAt = model.Timestamp()
			err := s.store.UpdateImport(i)
			if err != nil {
				s.logger.WithError(err).Warnf("Import %s was complete but couldn't be marked as such", i.ID)
			}

			key := fmt.Sprintf("%s.zip", i.TranslationID)
			cfg, err := config.LoadDefaultConfig(context.TODO())
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to clean up s3://%s/%s", s.bucket, key)
				return err
			}

			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
			defer cancelFunc()

			client := s3.NewFromConfig(cfg)
			_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: &s.bucket,
				Key:    &key,
			})

			return err
		}
	}

	return nil
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
