// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mattermost/awat/internal/store"
	"github.com/mattermost/awat/internal/translator"
	"github.com/mattermost/awat/model"
)

// TranslationSupervisor is responsible for scheduling and launching Translations
// in series
type TranslationSupervisor struct {
	logger  log.FieldLogger
	store   *store.SQLStore
	bucket  string
	workdir string
}

// NewTranslationSupervisor returns a Supervisor prepared with the needed
// metadata to operate
func NewTranslationSupervisor(store *store.SQLStore, logger log.FieldLogger, bucket, workdir string) *TranslationSupervisor {
	return &TranslationSupervisor{
		store:   store,
		logger:  logger.WithField("translation-supervisor", model.NewID()),
		bucket:  bucket,
		workdir: workdir,
	}
}

// Start runs the Supervisor's main routine on a new goroutine both
// periodically and forever
func (s *TranslationSupervisor) Start() {
	s.logger.Info("Translation supervisor started")
	go func() {
		for {
			s.supervise()
			time.Sleep(60 * time.Second) // TODO: make this configurable
		}
	}()
}

// supervise queries the database for available Translations and
// works through the batch returned serially
func (s *TranslationSupervisor) supervise() {
	translation, err := s.store.GetTranslationReadyToStart()
	if err != nil {
		s.logger.WithError(err).Error("Failed to query database for pending translations")
		return
	}
	if translation == nil {
		return
	}

	logger := s.logger.WithFields(log.Fields{"translation": translation.ID, "installation": translation.InstallationID})
	logger.Info("Beginning translation")

	// TODO XXX expose the Pod name as an env var and use it as the second argument here
	err = s.store.TryLockTranslation(translation, model.NewID())
	if err != nil {
		logger.WithError(err).Error("failed to lock translation")
		return
	}
	defer s.store.UnlockTranslation(translation)

	trans, err := translator.NewTranslator(
		&translator.TranslatorOptions{
			ArchiveType: translation.Type,
			Bucket:      s.bucket,
			WorkingDir:  s.workdir,
		})
	if err != nil {
		logger.WithError(err).Error("Failed to create translator")
		return
	}

	translation.StartAt = model.GetMillis()
	err = s.store.UpdateTranslation(translation)
	if err != nil {
		logger.WithError(err).Error("Failed to mark translation as started")
		return
	}

	output, err := trans.Translate(translation)
	if err != nil {
		logger.WithError(err).Error("Failed translation")
		return
	}

	translation.CompleteAt = model.GetMillis()
	err = s.store.UpdateTranslation(translation)
	if err != nil {
		logger.WithError(err).Error("Failed to mark translation as completed")
		return
	}

	importResource := fmt.Sprintf("%s/%s", s.bucket, output)
	imp := model.NewImport(translation.ID, importResource)
	err = s.store.CreateImport(imp)
	if err != nil {
		logger.WithError(err).Error("Failed to create an import for translation")
		return
	}

	logger.Info("Translation completed")
}
