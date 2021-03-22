package supervisor

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mattermost/awat/internal/store"
	"github.com/mattermost/awat/internal/translator"
	"github.com/mattermost/awat/model"
)

// Supervisor is responsible for scheduling and launching Translations
// in series
type Supervisor struct {
	logger  log.FieldLogger
	store   *store.SQLStore
	bucket  string
	workdir string
}

// NewSupervisor returns a Supervisor prepared with the needed
// metadata to operate
func NewSupervisor(store *store.SQLStore, logger log.FieldLogger, bucket, workdir string) *Supervisor {
	return &Supervisor{
		store:   store,
		logger:  logger.WithField("supervisor", model.NewID()),
		bucket:  bucket,
		workdir: workdir,
	}
}

// Start runs the Supervisor's main routine on a new goroutine both
// periodically and forever
func (s *Supervisor) Start() {
	s.logger.Info("Supervisor started")
	go func() {
		for {
			s.supervise()
			time.Sleep(60 * time.Second)
		}
	}()
}

// supervise queries the database for available Translations and
// works through the batch returned serially
func (s *Supervisor) supervise() {
	translation, err := s.store.GetTranslationReadyToStart()
	if err != nil {
		s.logger.WithError(err).Error("Failed to query database for pending translations")
		return
	}

	if translation != nil {
		s.logger.Debugf("Found request %s pending to be translated", translation.ID)
	}

	s.logger.Debugf("Translating %s for Installation %s...", translation.ID, translation.InstallationID)
	// TODO XXX expose the Pod name as an env var and use it as the second argument here
	err = s.store.TryLockTranslation(translation, model.NewID())
	if err != nil {
		s.logger.WithError(err).Warnf("failed to lock Translation %s", translation.ID)
		return
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
		return
	}

	translation.StartAt = model.Timestamp()
	err = s.store.UpdateTranslation(translation)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to mark Translation %s as started; will not claim or begin translation process at this time", translation.ID)
		return
	}
	output, err := translator.Translate(translation)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to translate Translation %s", translation.ID)
		err = s.store.UpdateTranslation(translation)
		if err != nil {
			s.logger.WithError(err).Errorf("failed to store error from failed translation")
		}
		return
	}

	translation.CompleteAt = model.Timestamp()
	translation.Output = output
	err = s.store.UpdateTranslation(translation)
	if err != nil {
		s.logger.WithError(err).Warnf("failed to store completed Translation %s; the Translation may be erroneously repeated!", translation.ID)
		return
	}

	imp := model.NewImport(translation.ID)
	err = s.store.StoreImport(imp)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to create an Import for Translation %s; the Translation may be complete but it will not be imported automatically", translation.ID)
		return
	}
}
