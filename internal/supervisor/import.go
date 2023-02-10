// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mattermost/awat/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type ImportSupervisor struct {
	id             string
	logger         log.FieldLogger
	store          importStore
	cloud          *cloud.Client
	awsConfig      *aws.Config
	bucket         string
	keepImportData bool
}

type importStore interface {
	GetUnlockedImportPendingWork() ([]*model.Import, error)
	GetTranslation(id string) (*model.Translation, error)
	UpdateImport(imp *model.Import) error
	TryLockImport(imp *model.Import, owner string) error
	UnlockImport(imp *model.Import) error
}

func NewImportSupervisor(store importStore, logger log.FieldLogger, cloudClient *cloud.Client, bucket string, keepImportData bool) *ImportSupervisor {
	id := model.NewID()
	return &ImportSupervisor{
		id:             id,
		logger:         logger.WithField("import-supervisor", id),
		store:          store,
		cloud:          cloudClient,
		bucket:         bucket,
		keepImportData: keepImportData,
	}
}

func (s *ImportSupervisor) Start() {
	s.logger.Info("Import supervisor started")

	tick := time.NewTicker(time.Second * 30)

	for range tick.C {
		s.do()
	}
}

func (s *ImportSupervisor) do() {
	imports, err := s.store.GetUnlockedImportPendingWork()
	if err != nil {
		s.logger.WithError(err).Error("Failed to query for import pending work")
		return
	}
	for _, imp := range imports {
		s.supervise(imp)
	}
}

func (s *ImportSupervisor) supervise(imp *model.Import) {
	logger := s.logger.WithFields(log.Fields{
		"import": imp.ID,
	})

	lockErr := s.store.TryLockImport(imp, s.id)
	if lockErr != nil {
		logger.WithError(lockErr).Warn("Failed to lock import")
		return
	}
	defer func(imp *model.Import) {
		unlockErr := s.store.UnlockImport(imp)
		if unlockErr != nil {
			logger.WithError(unlockErr).Warn("Failed to unlock import")
		}
	}(imp)

	translation, err := s.store.GetTranslation(imp.TranslationID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to look up Translation %s", imp.TranslationID)
		return
	}

	installation, err := s.cloud.GetInstallation(
		translation.InstallationID,
		&cloud.GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		})
	if err != nil {
		logger.WithError(err).Error("Failed to fetch installation")
		return
	}

	if installation == nil || installation.State == cloud.InstallationStateDeleted {
		logger.Error("No Installation found")
		imp.State = model.ImportStateFailed
		err := s.store.UpdateImport(imp)
		if err != nil {
			logger.WithError(err).Error("Failed to update import")
			return
		}
		return
	}

	logger = s.logger.WithFields(log.Fields{
		"installation": installation.ID,
	})

	newState := s.transitionImport(imp, installation, logger)

	if newState != imp.State {
		imp.State = newState
		err := s.store.UpdateImport(imp)
		if err != nil {
			logger.WithError(err).Error("Failed to update import")
			return
		}
	}
}

func (s *ImportSupervisor) transitionImport(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	switch imp.State {
	case model.ImportStateRequested:
		return s.transitionImportRequested(imp, installation, logger)
	case model.ImportStateInstallationPreAdjustment:
		return s.transitionImportInstallationPreAdjustment(imp, installation, logger)
	case model.ImportStateInProgress:
		return s.transitionImportInProgress(imp, installation, logger)
	case model.ImportStateComplete:
		return s.transitionImportComplete(imp, installation, logger)
	case model.ImportStateInstallationPostAdjustment:
		return s.transitionImportInstallationPostAdjustment(imp, installation, logger)
	}

	return imp.State
}

func (s *ImportSupervisor) transitionImportRequested(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {

	if installation.State != cloud.InstallationStateStable {
		logger.Debug("Waiting for installation to be stable")
		return imp.State
	}

	var adjustmentRequired bool
	size := installation.Size
	if size != model.Size1000String {
		logger.Infof("Resizing installation to %s\n", model.Size1000String)
		size = model.Size1000String
		adjustmentRequired = true
	}

	s3TimeoutEnv := installation.PriorityEnv[model.S3EnvKey]
	if s3TimeoutEnv.Value != fmt.Sprintf("%d", model.S3ExtendedTimeout) {
		logger.Info("Extending S3 timeout to 48 hours")
		s3TimeoutEnv.Value = fmt.Sprintf("%d", model.S3ExtendedTimeout)
		adjustmentRequired = true
	}

	if !adjustmentRequired {
		logger.Info("No adjustments required")
		return model.ImportStateInProgress
	}

	logger.Info("Adjustments required")
	patchInstallation := &cloud.PatchInstallationRequest{
		Size: &size,
		PriorityEnv: cloud.EnvVarMap{
			model.S3EnvKey: s3TimeoutEnv,
		},
	}

	_, err := s.cloud.UpdateInstallation(installation.ID, patchInstallation)
	if err != nil {
		logger.WithError(err).Error("Failed to update installation")
		return imp.State
	}
	return model.ImportStateInstallationPreAdjustment
}

func (s *ImportSupervisor) transitionImportInstallationPreAdjustment(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	if installation.State != cloud.InstallationStateStable {
		logger.Debug("Waiting for installation to be stable")
		return imp.State
	}

	logger.Debug("Installation is Stable")

	if installation.Size != model.Size1000String {
		logger.Debug("Installation is not in the correct size")
		return model.ImportStateRequested
	}

	s3TimeoutEnv := installation.PriorityEnv[model.S3EnvKey]
	if s3TimeoutEnv.Value != fmt.Sprintf("%d", model.S3ExtendedTimeout) {
		logger.Debug("S3 timeout is not extended")
		return model.ImportStateRequested
	}

	logger.Info("Installation is in the correct size and S3 timeout is extended")

	return model.ImportStateInProgress
}

func (s *ImportSupervisor) transitionImportInProgress(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	if !startedImportIsComplete(installation) {
		logger.Debug("Import is still running")
		return imp.State
	}

	imp.CompleteAt = model.GetMillis()
	err := s.store.UpdateImport(imp)
	if err != nil {
		logger.WithError(err).Error("Failed to mark import as completed")
		return imp.State
	}

	logger.Info("Import completed")

	if s.keepImportData {
		logger.Debug("Skipping import bundle cleanup")
		return model.ImportStateComplete
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		logger.WithError(err).Error("Failed to load AWS config")
		return model.ImportStateComplete
	}

	key := fmt.Sprintf("%s.zip", imp.TranslationID)
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc()

	client := s3.NewFromConfig(cfg)
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to delete translation from S3")
		return model.ImportStateComplete
	}

	logger.Debug("Import cleanup completed successfully")

	return model.ImportStateComplete
}

func (s *ImportSupervisor) transitionImportComplete(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	if installation.State != cloud.InstallationStateStable {
		logger.Debug("Waiting for installation to be stable")
		return imp.State
	}

	logger.Debug("Installation is Stable")

	var adjustmentRequired bool
	size := installation.Size
	if size == model.Size1000String {
		logger.Infof("Resizing installation to %s\n", model.SizeCloud10Users)
		size = model.SizeCloud10Users
		adjustmentRequired = true
	}

	s3TimeoutEnv := installation.PriorityEnv[model.S3EnvKey]
	if s3TimeoutEnv.Value == fmt.Sprintf("%d", model.S3ExtendedTimeout) {
		logger.Info("Setting S3 timeout back to the default value")
		s3TimeoutEnv.Value = fmt.Sprintf("%d", model.S3DefaultTimeout)
		adjustmentRequired = true
	}

	if !adjustmentRequired {
		logger.Info("No adjustments required")
		if imp.Error != "" {
			return model.ImportStateFailed
		}
		return model.ImportStateSucceeded
	}

	logger.Info("Adjustments required")
	patchInstallation := &cloud.PatchInstallationRequest{
		Size: &size,
		PriorityEnv: cloud.EnvVarMap{
			model.S3EnvKey: s3TimeoutEnv,
		},
	}

	_, err := s.cloud.UpdateInstallation(installation.ID, patchInstallation)
	if err != nil {
		logger.WithError(err).Error("Failed to update installation")
		return imp.State
	}
	return model.ImportStateInstallationPostAdjustment
}

func (s *ImportSupervisor) transitionImportInstallationPostAdjustment(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	if installation.State != cloud.InstallationStateStable {
		logger.Debug("Waiting for installation to be stable")
		return imp.State
	}

	logger.Debug("Installation is Stable")

	if installation.Size == model.Size1000String {
		logger.Debug("Installation is not in the correct size")
		return model.ImportStateComplete
	}

	s3TimeoutEnv := installation.PriorityEnv[model.S3EnvKey]
	if s3TimeoutEnv.Value == fmt.Sprintf("%d", model.S3ExtendedTimeout) {
		logger.Debug("S3 timeout is not extended")
		return model.ImportStateComplete
	}

	logger.Info("Installation is in the correct size and S3 timeout is extended")

	if imp.Error != "" {
		return model.ImportStateFailed
	}

	return model.ImportStateSucceeded
}

// startedImportIsComplete returns true if an Import with a nonzero
// StartAt value has been completed, and false otherwise.
func startedImportIsComplete(installation *cloud.InstallationDTO) bool {
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
