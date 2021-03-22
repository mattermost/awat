package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/awat/model"
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
	outputJSON(c, w, importStatusFromImport(work))
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
	outputJSON(c, w, importStatusListFromImports(imports))
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
	outputJSON(c, w, importStatusFromImport(imprt))
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
	outputJSON(c, w, importStatusListFromImports(imports))
}

func importStatusFromImport(imp *model.Import) (status *model.ImportStatus) {
	return &model.ImportStatus{
		Import: *imp,
		State:  imp.State(),
	}
}

func importStatusListFromImports(imports []*model.Import) (statuses []*model.ImportStatus) {
	for _, t := range imports {
		statuses = append(statuses, importStatusFromImport(t))
	}
	return
}
