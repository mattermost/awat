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
	"github.com/mattermost/awat/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ImportSupervisor is responsible for supervising the import process.
// It manages the import lifecycle and communicates with other services like AWS and Mattermost Cloud.
type ImportSupervisor struct {
	id             string
	logger         log.FieldLogger
	store          importStore
	cloud          *cloud.Client
	bucket         string
	keepImportData bool
}

// importStore defines the interface for interacting with the import storage.
type importStore interface {
	GetUnlockedImportPendingWork() ([]*model.Import, error)
	GetTranslation(id string) (*model.Translation, error)
	UpdateImport(imp *model.Import) error
	TryLockImport(imp *model.Import, owner string) error
	UnlockImport(imp *model.Import) error
}

// NewImportSupervisor creates a new ImportSupervisor instance.
// It initializes the supervisor with provided parameters including the import store, logger, cloud client, etc.
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

// Start begins the import supervision process.
// It regularly checks for new import tasks and processes them accordingly.
func (s *ImportSupervisor) Start() {
	s.logger.Info("Import supervisor started")

	tick := time.NewTicker(time.Second * 30)

	for range tick.C {
		s.do()
	}
}

// do performs a single supervision iteration.
// It fetches pending import tasks and processes each one.
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

// supervise handles the supervision of a single import task.
// It involves locking the import, checking its state, and processing it based on its current state.
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

// transitionImport manages the state transition of an import.
// Depending on the current state of the import and the associated installation, it moves the import to the next state.
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

// transitionImportRequested handles the transition for an import in the 'requested' state.
// It checks the installation's readiness and prepares it for the import process.
func (s *ImportSupervisor) transitionImportRequested(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	if installation.State != cloud.InstallationStateStable {
		logger.Debug("Waiting for installation to be stable")
		return imp.State
	}

	logger.Info("Running pre-import installation configuration check")
	patch := getPreImportPatch(installation.Installation, logger)
	if patch == nil {
		logger.Info("No installation adjustments required")
		return model.ImportStateInProgress
	}

	logger.Info("Adjusting installation configuration")

	err := s.handleInstallationUpdate(installation, patch, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to update installation")
		return imp.State
	}

	return model.ImportStateInstallationPreAdjustment
}

// transitionImportInstallationPreAdjustment handles the transition for an import in the 'pre-adjustment' state.
// It waits for the installation to become stable after initial adjustments.
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

	if installation.PriorityEnv[model.S3EnvKey].Value != fmt.Sprintf("%d", model.S3ExtendedTimeout) {
		logger.Debug("S3 timeout is not extended")
		return model.ImportStateRequested
	}

	if installation.PriorityEnv[model.ExtractContentKey].Value != model.ExtractContentDisabled {
		logger.Debug("File content extraction is not disabled")
		return model.ImportStateRequested
	}

	logger.Info("Installation has the correct import configuration")

	return model.ImportStateInProgress
}

// transitionImportInProgress handles the transition for an import in the 'in-progress' state.
// It monitors the import process and updates the state once the import is complete.
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

// transitionImportComplete handles the transition for an import in the 'complete'
// state. It performs final adjustments and cleanup after the import is done.
func (s *ImportSupervisor) transitionImportComplete(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	if installation.State != cloud.InstallationStateStable {
		logger.Debug("Waiting for installation to be stable")
		return imp.State
	}

	logger.Info("Running post-import installation configuration check")
	patch := getPostImportPatch(installation.Installation, logger)
	if patch == nil {
		logger.Info("No installation adjustments required")
		if imp.Error != "" {
			return model.ImportStateFailed
		}
		return model.ImportStateSucceeded
	}

	logger.Info("Adjusting installation configuration")

	err := s.handleInstallationUpdate(installation, patch, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to update installation")
		return imp.State
	}

	return model.ImportStateInstallationPostAdjustment
}

// transitionImportInstallationPostAdjustment handles the transition for an import in the 'post-adjustment' state.
// It ensures the installation returns to its normal state after the import.
func (s *ImportSupervisor) transitionImportInstallationPostAdjustment(imp *model.Import, installation *cloud.InstallationDTO, logger log.FieldLogger) string {
	if installation.State != cloud.InstallationStateStable {
		logger.Debug("Waiting for installation to be stable")
		return imp.State
	}

	logger.Debug("Installation is Stable")

	if installation.Size == model.Size1000String {
		logger.Warn("Installation is not in the correct size")
		return model.ImportStateComplete
	}

	if installation.PriorityEnv[model.S3EnvKey].Value == fmt.Sprintf("%d", model.S3ExtendedTimeout) {
		logger.Warn("S3 timeout is still extended")
		return model.ImportStateComplete
	}

	logger.Info("Installation has been reverted to default configuration")

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

func getPreImportPatch(installation *cloud.Installation, logger log.FieldLogger) *cloud.PatchInstallationRequest {
	var adjustmentRequired bool
	patch := &cloud.PatchInstallationRequest{}

	importSize := model.Size1000String
	if installation.Size != importSize {
		logger.Debugf("Resizing installation to %s", importSize)
		patch.Size = &importSize
		adjustmentRequired = true
	}

	// For the env overrides we need to look at both the priority and normal env
	// vars for the installation to see if either is set.
	envPatches := cloud.EnvVarMap{}

	installationS3TimeoutEnvValue := getInstallationEnvValue(installation, model.S3EnvKey)
	importS3TimeoutString := fmt.Sprintf("%d", model.S3ExtendedTimeout)
	if installationS3TimeoutEnvValue != importS3TimeoutString {
		logger.Debug("Extending S3 timeout to 48 hours")
		envPatches[model.S3EnvKey] = cloud.EnvVar{Value: importS3TimeoutString}
		adjustmentRequired = true
	}
	installationExtractContent := getInstallationEnvValue(installation, model.ExtractContentKey)
	importExtractContent := model.ExtractContentDisabled
	if installationExtractContent != importExtractContent {
		logger.Debug("Disabling file content extraction")
		envPatches[model.ExtractContentKey] = cloud.EnvVar{Value: importExtractContent}
		adjustmentRequired = true
	}
	if len(envPatches) != 0 {
		patch.PriorityEnv = envPatches
	}

	if !adjustmentRequired {
		return nil
	}

	return patch
}

func getInstallationEnvValue(installation *cloud.Installation, key string) string {
	priorityValue := installation.PriorityEnv[key].Value
	if priorityValue != "" {
		return priorityValue
	}

	return installation.MattermostEnv[key].Value
}

func getPostImportPatch(installation *cloud.Installation, logger log.FieldLogger) *cloud.PatchInstallationRequest {
	var adjustmentRequired bool
	patch := &cloud.PatchInstallationRequest{}

	defaultSize := model.SizeCloud10Users
	if installation.Size == model.Size1000String {
		logger.Debugf("Resizing installation to %s", defaultSize)
		patch.Size = &defaultSize
		adjustmentRequired = true
	}

	if installation.PriorityEnv[model.S3EnvKey].Value == fmt.Sprintf("%d", model.S3ExtendedTimeout) ||
		installation.PriorityEnv[model.ExtractContentKey].Value == model.ExtractContentDisabled {
		// NOTE: We want to clear the priority env var instead of setting it to
		// a default value so that standard group environment variables are not
		// ignored on the installation. Clearing the priority env vars will
		// remove other custom env vars that were set. In order to not add extra
		// complexity that would be needed to see if other custom env vars need
		// to be re-applied as a follow-up step, we will assume that clearing
		// everything is okay. Installations receiving imports should always be
		// newly-created so it's unlikely they should have overrides.
		logger.Debug("Clearing all priority env to remove import overrides")
		patch.PriorityEnv = cloud.EnvVarMap{}
		adjustmentRequired = true
	}

	if !adjustmentRequired {
		return nil
	}

	return patch
}

func (s *ImportSupervisor) handleInstallationUpdate(installation *cloud.InstallationDTO, patch *cloud.PatchInstallationRequest, logger log.FieldLogger) error {
	var err error
	if installation.APISecurityLock {
		err = s.cloud.UnlockAPIForInstallation(installation.ID)
		if err != nil {
			return errors.Wrap(err, "Failed to unlock installation")
		}

		defer func() {
			err = s.cloud.LockAPIForInstallation(installation.ID)
			if err != nil {
				logger.WithError(err).Error("Failed to relock installation")
			}
		}()
	}

	_, err = s.cloud.UpdateInstallation(installation.ID, patch)
	if err != nil {
		return errors.Wrap(err, "Failed to update installation")
	}

	return nil
}
