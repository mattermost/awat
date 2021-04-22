package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/awat/model"
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
	}

	translation := model.NewTranslationFromRequest(translationRequest)
	err = c.Store.StoreTranslation(translation)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to store the translation request in the database")
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, translationStatusFromTranslation(translation))

	c.Logger.Debugf("Started new translation with ID %s for Installation %s", translation.ID, translation.InstallationID)
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
