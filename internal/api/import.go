package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/awat/model"
)

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

func handleListImports(c *Context, w http.ResponseWriter, r *http.Request) {
	imports, err := c.Store.GetAllImports()
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch imports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(imports) < 1 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, importStatusListFromImports(imports))
}

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

func handleGetImportStatusesByInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	imports, err := c.Store.GetImportsByInstallation(id)
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch imports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(imports) < 1 {
		w.WriteHeader(http.StatusNotFound)
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
