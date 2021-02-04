package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func Register(rootRouter *mux.Router) {
	rootRouter.HandleFunc("/translate", handleStartTranslation).Methods("POST")
	rootRouter.HandleFunc("/translate", handleGetTranslationStatus).Methods("GET")
}

func handleStartTranslation(w http.ResponseWriter, r *http.Request) {
}

func handleGetTranslationStatus(w http.ResponseWriter, r *http.Request) {
}
