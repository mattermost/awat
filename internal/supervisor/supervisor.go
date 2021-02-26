package supervisor

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mattermost/awat/internal/store"
	"github.com/mattermost/awat/internal/translator"
	"github.com/mattermost/mattermost-cloud/model"
)

type Supervisor struct {
	logger log.FieldLogger
	store  *store.SQLStore
	bucket string
}

func NewSupervisor(store *store.SQLStore, logger log.FieldLogger, bucket string) *Supervisor {
	return &Supervisor{
		store:  store,
		logger: logger.WithField("supervisor", model.NewID()),
		bucket: bucket,
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
		s.logger.WithError(err).Error("failed to query database for pending translations")
		return
	}

	s.logger.Debugf("Found %d requests pending to be translated", len(work))
	for _, translation := range work {
		s.logger.Debugf("Translating %s for Installation %s...", translation.ID, translation.InstallationID)
		translator, err := translator.NewTranslator(translation.Type)
		if err != nil {
			s.logger.WithError(err).Error("failed to create translator for Translation %s", translation.ID)
			continue
		}
		translation.StartAt = time.Now().UnixNano() / 1000
		err = s.store.UpdateTranslation(translation)
		if err != nil {
			s.logger.WithError(err).Errorf("failed to mark Translation %s as started; will not claim or begin translation process at this time", translation.ID)
			continue
		}
		err = translator.Translate(translation, s.bucket)
		if err != nil {
			s.logger.WithError(err).Errorf("failed to translate Translation %s", translation.ID)
			continue
		}

		translation.CompleteAt = time.Now().UnixNano() / 1000
		err = s.store.UpdateTranslation(translation)
		if err != nil {
			s.logger.WithError(err).Warnf("failed to store completed Translation %s; the Translation may be erroneously repeated!", translation.ID)
			continue
		}
	}
}
