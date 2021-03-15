package supervisor

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mattermost/awat/internal/store"
	"github.com/mattermost/awat/internal/translator"
	"github.com/mattermost/awat/model"
)

type Supervisor struct {
	logger  log.FieldLogger
	store   *store.SQLStore
	bucket  string
	workdir string
}

func NewSupervisor(store *store.SQLStore, logger log.FieldLogger, bucket, workdir string) *Supervisor {
	return &Supervisor{
		store:   store,
		logger:  logger.WithField("supervisor", model.NewID()),
		bucket:  bucket,
		workdir: workdir,
	}
}

func (s *Supervisor) Start() {
	s.logger.Info("Supervisor started")
	go func() {
		for {
			s.supervise()
			time.Sleep(15 * time.Second)
		}
	}()
}

func (s *Supervisor) supervise() {
	work, err := s.store.GetTranslationsReadyToStart()
	if err != nil {
		s.logger.WithError(err).Error("Failed to query database for pending translations")
		return
	}

	if len(work) > 0 {
		s.logger.Debugf("Found %d requests pending to be translated", len(work))
	}

	for _, translation := range work {
		s.logger.Debugf("Translating %s for Installation %s...", translation.ID, translation.InstallationID)
		// TODO XXX expose the Pod name as an env var and use it as the second argument here
		err = s.store.TryLockTranslation(translation, model.NewID())
		if err != nil {
			s.logger.WithError(err).Warnf("failed to lock Translation %s", translation.ID)
			continue
		}
		defer s.store.UnlockTranslation(translation)

		translator, err := translator.NewTranslator(
			&translator.TranslatorOptions{
				ArchiveType: translation.Type,
				Bucket:      s.bucket,
				WorkingDir:  s.workdir,
			})
		if err != nil {
			s.logger.WithError(err).Error("failed to create translator for Translation %s", translation.ID)
			continue
		}

		translation.StartAt = model.Timestamp()
		err = s.store.UpdateTranslation(translation)
		if err != nil {
			s.logger.WithError(err).Errorf("failed to mark Translation %s as started; will not claim or begin translation process at this time", translation.ID)
			continue
		}
		output, err := translator.Translate(translation)
		if err != nil {
			s.logger.WithError(err).Errorf("failed to translate Translation %s", translation.ID)
			err = s.store.UpdateTranslation(translation)
			if err != nil {
				s.logger.WithError(err).Errorf("failed to store error from failed translation")
			}
			continue
		}

		translation.CompleteAt = model.Timestamp()
		translation.Output = output
		err = s.store.UpdateTranslation(translation)
		if err != nil {
			s.logger.WithError(err).Warnf("failed to store completed Translation %s; the Translation may be erroneously repeated!", translation.ID)
			continue
		}

		imp := model.NewImport(translation.ID)
		err = s.store.StoreImport(imp)
		if err != nil {
			s.logger.WithError(err).Errorf("failed to create an Import for Translation %s; the Translation may be complete but it will not be imported automatically", translation.ID)
			continue
		}

	}
}
