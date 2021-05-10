package supervisor

import (
	"time"

	"github.com/mattermost/awat/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ImportSupervisor struct {
	logger log.FieldLogger
	store  importStore
	cloud  *cloud.Client
}

type importStore interface {
	GetImportsInProgress() ([]*model.Import, error)
	GetTranslation(id string) (*model.Translation, error)
	UpdateImport(imp *model.Import) error
}

func NewImportSupervisor(store importStore, logger log.FieldLogger, provisionerURL string) *ImportSupervisor {
	cloudClient := cloud.NewClient(provisionerURL)
	return &ImportSupervisor{
		logger: logger,
		store:  store,
		cloud:  cloudClient,
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
			time.Sleep(60 * time.Second)
		}
	}()
}

func (s *ImportSupervisor) supervise() error {
	imports, err := s.store.GetImportsInProgress()
	if err != nil {
		return errors.Wrap(err, "failed to look up ongoing Imports to supervise")
	}

	for _, i := range imports {
		if (time.Now().UnixNano()/1000)-i.StartAt < time.Second.Milliseconds()*10 {
			// if the above condition is true, the Import was claimed to be
			// started very recently (less than 10 seconds ago) and it's
			// possible that the Provisioner hasn't changed the state on the
			// corresponding Installation yet

			// the first thing the Provisioner does after getting some work
			// is to lock the Installation, so the window for a race
			// condition should be a sub-second window and this pause should
			// be reliable if slightly overkill
			continue
		}

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

		// if the State is stable and the Installation is slightly old, we
		// can safely assume that an import happened and was completed
		if installation.State == "stable" {
			i.CompleteAt = time.Now().Unix() / 1000
			err := s.store.UpdateImport(i)
			if err != nil {
				s.logger.WithError(err).Warnf("Import %s was done but couldn't be marked as such", i.ID)
			}
		}
	}

	return nil
}
