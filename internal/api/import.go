package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/awat/model"
	"github.com/pkg/errors"
)

// handleStartImport handles POST requests sent to /import This
// endpoint takes an ID, locks the oldest Import awaiting work with
// that ID, and returns the metadata associated with that Import in
// order for a Provisioner to being work on it
func handleStartImport(c *Context, w http.ResponseWriter, r *http.Request) {
	workRequest, err := model.NewImportWorkRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to unmarshal request for work")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	work, err := c.Store.GetAndClaimNextReadyImport(workRequest.ProvisionerID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch import")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if work == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	status, err := importStatusFromImport(work, c.Store)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to get ImportStatus for Import %s", work.ID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outputJSON(c, w, status)
}

func handleReleaseLockOnImport(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	importID := vars["id"]
	imprt, err := c.Store.GetImport(importID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to fetch import with ID %s", importID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	imprt.LockedBy = ""
	imprt.StartAt = 0

	err = c.Store.UpdateImport(imprt)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to release lock on Import %s", importID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleListImports responds to GET /imports and returns all Imports
// in the database
// TODO add pagination to this endpoint
func handleListImports(c *Context, w http.ResponseWriter, r *http.Request) {
	imports, err := c.Store.GetAllImports()
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch imports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	statuses, err := importStatusListFromImports(imports, c.Store)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get ImportStatuses")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outputJSON(c, w, statuses)
}

// handleGetImport responds to GET /import/{id} with one Import
func handleGetImport(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	importID := vars["id"]
	imprt, err := c.Store.GetImport(importID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to fetch import with ID %s", importID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if imprt == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	status, err := importStatusFromImport(imprt, c.Store)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to generate ImportStatus with ID %s", importID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outputJSON(c, w, status)
}

// handleGetImportStatusesByInstallation allows easily looking up all
// Imports related to an Installation by the Installation ID
func handleGetImportStatusesByInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	imports, err := c.Store.GetImportsByInstallation(id)
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch imports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	statuses, err := importStatusListFromImports(imports, c.Store)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get ImportStatuses")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	outputJSON(c, w, statuses)
}

func handleCompleteImport(c *Context, w http.ResponseWriter, r *http.Request) {
	completed, err := model.NewImportCompletedWorkRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode completed work request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	imp, err := c.Store.GetImport(completed.ID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to look up Import %s", completed.ID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if imp == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	imp.CompleteAt = completed.CompleteAt
	imp.LockedBy = ""
	imp.Error = completed.Error

	err = c.Store.UpdateImport(imp)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to update Import with info: %+v", completed)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func importStatusFromImport(imp *model.Import, store Store) (*model.ImportStatus, error) {
	translation, err := store.GetTranslation(imp.TranslationID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lookup Translation %s", imp.TranslationID)
	}
	return &model.ImportStatus{
		Import:         *imp,
		InstallationID: translation.InstallationID,
		Users:          translation.Users,
		Team:           translation.Team,
		State:          imp.State(),
	}, nil
}

func importStatusListFromImports(imports []*model.Import, store Store) (statuses []*model.ImportStatus, err error) {
	for _, t := range imports {
		status, err := importStatusFromImport(t, store)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get create ImportStatus from Import %s", t.ID)
		}
		statuses = append(statuses, status)
	}
	return
}
