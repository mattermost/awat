package supervisor

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
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
			err := s.supervise()
			if err != nil {
				s.logger.WithError(err).Error("failed an operation while supervising translations")
			}
			time.Sleep(60 * time.Second)
		}
	}()
}

// supervise queries the database for available Translations and
// works through the batch returned serially
func (s *Supervisor) supervise() error {
	translation, err := s.store.GetTranslationReadyToStart()
	if err != nil {
		return errors.Wrap(err, "Failed to query database for pending translations")
	}

	if translation == nil {
		// no work found so just return
		return nil
	}

	s.logger.Debugf("Translating %s for Installation %s...", translation.ID, translation.InstallationID)

	// TODO XXX expose the Pod name as an env var and use it as the second argument here
	err = s.store.TryLockTranslation(translation, model.NewID())
	if err != nil {
		return errors.Wrapf(err, "failed to lock Translation %s", translation.ID)
	}
	defer s.store.UnlockTranslation(translation)

	translator, err := translator.NewTranslator(
		&translator.TranslatorOptions{
			ArchiveType: translation.Type,
			Bucket:      s.bucket,
			WorkingDir:  s.workdir,
		})
	if err != nil {
		return errors.Wrapf(err, "failed to create translator for Translation %s", translation.ID)
	}

	translation.StartAt = model.Timestamp()
	err = s.store.UpdateTranslation(translation)
	if err != nil {
		return errors.Wrapf(err, "failed to mark Translation %s as started; will not claim or begin translation process at this time", translation.ID)
	}
	output, err := translator.Translate(translation)
	if err != nil {
		return errors.Wrapf(err, "failed to translate Translation %s", translation.ID)
	}

	translation.CompleteAt = model.Timestamp()
	err = s.store.UpdateTranslation(translation)
	if err != nil {
		return errors.Wrapf(err, "failed to store completed Translation %s; the Translation may be erroneously repeated!", translation.ID)
	}

	importResource := fmt.Sprintf("%s/%s", s.bucket, output)
	imp := model.NewImport(translation.ID, importResource)
	err = s.store.StoreImport(imp)
	if err != nil {
		return errors.Wrapf(err, "failed to create an Import for Translation %s; the Translation may be complete but it will not be imported automatically", translation.ID)
	}

	return nil
}
