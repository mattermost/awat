// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/awat/internal/validators"
	"github.com/mattermost/awat/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// handleListTranslations returns all Translations in the database. Responds to GET /translations
// TODO add pagination
func handleListTranslations(c *Context, w http.ResponseWriter, r *http.Request) {
	translations, err := c.Store.GetAllTranslations()
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch translations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, translationStatusListFromTranslations(translations))
}

// handleStartTranslation uses the TranslationRequest provided via
// POST /translation to start a new translation by storing it in the
// database. The supervisor will periodically discover stored
// Translations such as this, and begin work on them.
func handleStartTranslation(c *Context, w http.ResponseWriter, r *http.Request) {
	translationRequest, err := model.NewTranslationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to unmarshal JSON from request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger := c.Logger.WithFields(logrus.Fields{
		"backupType":   translationRequest.Type,
		"archive":      translationRequest.Archive,
		"installation": translationRequest.InstallationID,
	})

	translation := model.NewTranslationFromRequest(translationRequest)
	exists, err := c.AWS.CheckBucketFileExists(translation.Resource)
	if err != nil {
		logger.WithError(err).Error("failed to check if bucket and file exist")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		logger.Warnf("resource %s does not exist in bucket %s", translation.Resource, c.AWS.GetBucketName())
		w.WriteHeader(http.StatusNotFound)
		return
	}

	responseHeader, err := handleTranslationUpload(c, translationRequest, logger)
	if err != nil {
		logger.WithError(err).Error("failed to ensure upload")
		w.WriteHeader(responseHeader)
		return
	}

	err = c.Store.CreateTranslation(translation)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to store the translation request in the database")
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, translationStatusFromTranslation(translation))

	c.Logger.WithFields(logrus.Fields{
		"installation": translation.InstallationID,
		"resource":     translation.Resource,
		"translation":  translation.ID,
	}).Debug("Started new translation")
}

func handleTranslationUpload(c *Context, translationRequest *model.TranslationRequest, logger logrus.FieldLogger) (int, error) {
	// If we're providing an archive from a bucket (and not uploading it directly)
	// we need to download and validate it locally before trying to import it to
	// avoid import errors later.
	if translationRequest.UploadID != nil {
		upload, err := c.Store.GetUpload(*translationRequest.UploadID)
		if err != nil {
			return http.StatusInternalServerError, errors.Wrap(err, "failed to get upload")
		}
		if upload == nil {
			return http.StatusBadRequest, errors.Errorf("no upload with ID %s found", *translationRequest.UploadID)
		} else {
			logger.Debugf("Upload with ID %s exists, skipping archive validation...", *translationRequest.UploadID)
			return http.StatusOK, nil
		}
	}

	// Check if the upload already exists based on archive name.
	trimmedArchiveName := model.TrimExtensionFromArchiveFilename(translationRequest.Archive)
	upload, err := c.Store.GetUpload(trimmedArchiveName)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "failed to get upload")
	}
	if upload != nil {
		logger.Debugf("Upload with archive name %s exists, skipping archive validation...", trimmedArchiveName)
		return http.StatusOK, nil
	}

	if !translationRequest.ValidateArchive || translationRequest.Type == model.SlackWorkspaceBackupType {
		logger.Debug("Skipping archive validation...")
	} else {
		logger.Info("Downloading archive for validation")

		validator, err := validators.NewValidator(translationRequest.Type)
		if err != nil {
			return http.StatusInternalServerError, errors.Wrap(err, "error getting validator")
		}

		archivePath, cleanup, err := c.AWS.DownloadArchiveFromS3(translationRequest.Archive)
		if err != nil {
			return http.StatusInternalServerError, errors.Wrap(err, "error downloading archive for validation")
		}
		defer cleanup()

		logger = logger.WithField("archivePath", archivePath)
		logger.Debug("Downloaded archive for validation")

		err = validator.Validate(archivePath)
		if err != nil {
			return http.StatusBadRequest, errors.Wrap(err, "archive validation failed")
		}

		logger.Info("Archive validation successful")
	}

	err = c.Store.CreateUpload(trimmedArchiveName, translationRequest.Type)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "failed to store upload in database")
	}

	return http.StatusOK, nil
}

// handleGetTranslationStatus responds to GET /translation/{id} with
// the detailed status of the Translation as JSON
func handleGetTranslationStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	translationID := vars["id"]
	translation, err := c.Store.GetTranslation(translationID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to fetch transaction with ID %s", translationID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if translation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, translationStatusFromTranslation(translation))
}

// handleGetTranslationStatusesByInstallation returns a list of
// Translations with the given Installation ID in order to ease
// discovery of which Translation or Translations may be in progress
// for a given Installation
func handleGetTranslationStatusesByInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	translations, err := c.Store.GetTranslationsByInstallation(id)
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch translations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	outputJSON(c, w, translationStatusListFromTranslations(translations))
}

func handleGetImportStatusesForTranslation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	imports, err := c.Store.GetImportsByTranslation(id)
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch Imports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	importStatusList, err := importStatusListFromImports(imports, c.Store)
	if err != nil {
		c.Logger.WithError(err).Error("failed to generate ImportStatus list from Import slice")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outputJSON(c, w, importStatusList)
}

// outputJSON is a helper method to write the given data as JSON to the given writer.
//
// It only logs an error if one occurs, rather than returning, since there is no point in trying
// to send a new status code back to the client once the body has started sending.
func outputJSON(c *Context, w io.Writer, data interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(data)
	if err != nil {
		c.Logger.WithError(err).Error("failed to encode result")
	}
}

func translationStatusFromTranslation(t *model.Translation) (status *model.TranslationStatus) {
	return &model.TranslationStatus{
		State:       t.State(),
		Translation: *t,
	}
}

func translationStatusListFromTranslations(translations []*model.Translation) (translationStatusList []*model.TranslationStatus) {
	for _, t := range translations {
		translationStatusList = append(translationStatusList, translationStatusFromTranslation(t))
	}
	return
}
