package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mattermost/awat/internal/model"
)

func Register(rootRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	rootRouter.Handle("/translate", addContext(handleStartTranslation)).Methods("POST")
	rootRouter.Handle("/transaction/{id}", addContext(handleGetTranslationStatus)).Methods("GET")
	rootRouter.Handle("/installation/{id}", addContext(handleGetTranslationStatusByInstallation)).Methods("GET")
}

func handleStartTranslation(c *Context, w http.ResponseWriter, r *http.Request) {
	translationRequest, err := model.NewTranslationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to unmarshal JSON from request")
		w.WriteHeader(http.StatusInternalServerError)
	}
	err = c.Store.StoreTranslation(model.NewTranslationFromRequest(translationRequest))
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to store the translation request in the database")
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer r.Body.Close()
}

func handleGetTranslationStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transactionID := vars["id"]
	transaction, err := c.Store.GetTranslation(transactionID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to fetch transaction with ID %s", transactionID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if transaction == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w,
		model.TranslationStatus{
			ID:             transactionID,
			InstallationID: transaction.InstallationID,
			State:          transaction.State(),
		})
}

func handleGetTranslationStatusByInstallation(c *Context, w http.ResponseWriter, r *http.Request) {

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
