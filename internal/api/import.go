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
		c.Logger.WithError(err).Error("failed to fetch translations")
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
	imprts, err := c.Store.GetAllImports()
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch imports")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	importStatuses := []*ImportStatus{}
	for _, t := range imprts {
		importStatuses = append(importStatuses, importStatusFromImport(t))
	}
	outputJSON(c, w, importStatuses)
}

func handleGetImport(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	importID := vars["id"]
	imprt, err := c.Store.GetImport(importID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to fetch transaction with ID %s", importID)
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

type ImportStatus struct {
	model.Import

	State string
}

func importStatusFromImport(imp *model.Import) (status *ImportStatus) {
	return &ImportStatus{
		Import: *imp,
		State:  imp.State(),
	}
}
