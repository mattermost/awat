// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE for license information.
//

package api

import (
	"github.com/gorilla/mux"
)

func Register(rootRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	rootRouter.Handle("/translate", addContext(handleStartTranslation)).Methods("POST")
	rootRouter.Handle("/translation/{id}", addContext(handleGetTranslationStatus)).Methods("GET")
	rootRouter.Handle("/translation/{id}/import", addContext(handleGetImportStatusesForTranslation)).Methods("GET")
	rootRouter.Handle("/translations", addContext(handleListTranslations)).Methods("GET")

	rootRouter.Handle("/import", addContext(handleStartImport)).Methods("POST")
	rootRouter.Handle("/import", addContext(handleCompleteImport)).Methods("PUT")
	rootRouter.Handle("/import/{id}", addContext(handleGetImport)).Methods("GET")
	rootRouter.Handle("/import/{id}/release", addContext(handleGetImport)).Methods("GET")
	rootRouter.Handle("/imports", addContext(handleListImports)).Methods("GET")

	rootRouter.Handle("/installation/translation/{id}", addContext(handleGetTranslationStatusesByInstallation)).Methods("GET")
	rootRouter.Handle("/installation/import/{id}", addContext(handleGetImportStatusesByInstallation)).Methods("GET")
}
