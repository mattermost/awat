// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mock_api "github.com/mattermost/awat/internal/mocks/api"
	"github.com/mattermost/awat/internal/testlib"
	"github.com/mattermost/awat/model"
)

type MockAWS struct {
	resourceExists bool
}

func (a *MockAWS) GetBucketName() string {
	return "test"
}

func (a *MockAWS) CheckBucketFileExists(file string) (bool, error) {
	return a.resourceExists, nil
}

func (a *MockAWS) UploadArchiveToS3(uploadFileName, destKeyName string) error {
	return nil
}

func (a *MockAWS) DownloadArchiveFromS3(archiveName string) (string, error) {
	return "", nil
}

func TestTranslations(t *testing.T) {
	logger := testlib.MakeLogger(t)
	mockController := gomock.NewController(t)
	store := mock_api.NewMockStore(mockController)
	router := mux.NewRouter()
	mockAWS := &MockAWS{resourceExists: true}
	Register(router, &Context{
		Store:  store,
		Logger: logger,
		AWS:    mockAWS,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("unknown translation", func(t *testing.T) {
		store.EXPECT().
			GetTranslation("bogusID").
			Return(nil, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/translation/bogusID", ts.URL))
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("fetch a translation successfully", func(t *testing.T) {
		translationID := model.NewID()
		store.EXPECT().
			GetTranslation(translationID).
			Return(&model.Translation{ID: translationID}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/translation/%s", ts.URL, translationID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		translation, err := model.NewTranslationStatusFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, translationID, translation.ID)
	})

	t.Run("encounter an error from the db", func(t *testing.T) {
		translationID := model.NewID()
		store.EXPECT().
			GetTranslation(translationID).
			Return(nil, errors.New("problem talking to database")).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/translation/%s", ts.URL, translationID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("fetch all translations", func(t *testing.T) {
		translationID := model.NewID()
		store.EXPECT().
			GetAllTranslations().
			Return([]*model.Translation{
				{ID: translationID},
			}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/translations", ts.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		translations, err := model.NewTranslationStatusListFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, 1, len(translations))
		assert.Equal(t, translationID, translations[0].ID)
	})

	t.Run("start a new translation", func(t *testing.T) {
		store.EXPECT().
			CreateTranslation(
				// a more specific expectation could be applied here, but it
				// doesn't seem worth the time to define a Matcher and get it
				// all working just to ignore the nondeterministic ID that's
				// passed to this function because the ID is generated at
				// runtime
				gomock.Any(),
			).
			Return(nil).
			Times(1)

		resp, err := http.Post(fmt.Sprintf("%s/translate", ts.URL), "application/json",
			strings.NewReader(
				`{"Type": "slack", "InstallationID": "installationID", "Archive": "foo.zip", "Team": "teamname"}`,
			))
		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		translation, err := model.NewTranslationStatusFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "foo.zip", translation.Resource)
		assert.Equal(t, "teamname", translation.Team)
		assert.Equal(t, "installationID", translation.InstallationID)
	})

	t.Run("start a new translation, bad resource name", func(t *testing.T) {
		mockAWS.resourceExists = false

		resp, err := http.Post(fmt.Sprintf("%s/translate", ts.URL), "application/json",
			strings.NewReader(
				`{"Type": "slack", "InstallationID": "installationID", "Archive": "foo.zip", "Team": "teamname"}`,
			))
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("get a translation by Installation ID", func(t *testing.T) {
		installationID := "installationID"
		translationID := "translationID"

		store.EXPECT().
			GetTranslationsByInstallation(installationID).
			Return([]*model.Translation{{ID: translationID, InstallationID: installationID}}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/installation/translation/%s", ts.URL, installationID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		translations, err := model.NewTranslationStatusListFromReader(resp.Body)
		require.NoError(t, err)
		require.NotEmpty(t, translations)
		assert.Equal(t, translationID, translations[0].ID)
	})
}

func TestImports(t *testing.T) {
	logger := testlib.MakeLogger(t)
	mockController := gomock.NewController(t)
	store := mock_api.NewMockStore(mockController)
	router := mux.NewRouter()
	Register(router, &Context{
		Store:  store,
		Logger: logger,
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("unknown import", func(t *testing.T) {
		store.EXPECT().
			GetImport("bogusID").
			Return(nil, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/import/bogusID", ts.URL))
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("fetch an import successfully", func(t *testing.T) {
		importID := "importID"
		translationID := "translationID"

		store.EXPECT().
			GetImport(importID).
			Return(&model.Import{ID: importID, TranslationID: translationID}, nil).
			Times(1)

		store.EXPECT().
			GetTranslation(translationID).
			Return(&model.Translation{ID: translationID}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/import/%s", ts.URL, importID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		imprt, err := model.NewImportStatusFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, importID, imprt.ID)
	})

	t.Run("fetch an import by Translation ID", func(t *testing.T) {
		importID := "importID"
		translationID := "translationID"

		store.EXPECT().
			GetImportsByTranslation(translationID).
			Return([]*model.Import{{ID: importID, TranslationID: translationID}}, nil).
			Times(1)

		store.EXPECT().
			GetTranslation(translationID).
			Return(&model.Translation{ID: translationID}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/translation/%s/import", ts.URL, translationID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		imports, err := model.NewImportStatusListFromReader(resp.Body)
		require.NoError(t, err)
		require.NotEmpty(t, imports)
		assert.Equal(t, importID, imports[0].ID)
	})

	t.Run("encounter an error from the db", func(t *testing.T) {
		importID := model.NewID()
		store.EXPECT().
			GetImport(importID).
			Return(nil, errors.New("problem talking to database")).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/import/%s", ts.URL, importID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("fetch all imports", func(t *testing.T) {
		importID := model.NewID()
		translationID := "translationID"

		store.EXPECT().
			GetAllImports().
			Return([]*model.Import{
				{ID: importID, TranslationID: translationID},
			}, nil).
			Times(1)

		store.EXPECT().
			GetTranslation(translationID).
			Return(&model.Translation{ID: translationID}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/imports", ts.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		imports, err := model.NewImportStatusListFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, 1, len(imports))
		assert.Equal(t, importID, imports[0].ID)
	})

	t.Run("start an import", func(t *testing.T) {
		importID := "importID"
		translationID := "translationID"

		store.EXPECT().
			GetAndClaimNextReadyImport("provisionerID").
			Return(&model.Import{ID: importID, TranslationID: translationID, State: model.ImportStateRequested}, nil).
			Times(1)

		store.EXPECT().
			GetTranslation(translationID).
			Return(&model.Translation{ID: translationID}, nil).
			Times(1)

		resp, err := http.Post(
			fmt.Sprintf("%s/import", ts.URL),
			"application/json",
			strings.NewReader(
				`{"ProvisionerID":"provisionerID"}`,
			))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		imprt, err := model.NewImportStatusFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, importID, imprt.ID)
		assert.Equal(t, "import-requested", imprt.State)
	})

	t.Run("mark an import as completed", func(t *testing.T) {
		importID := "importID"
		translationID := "translationID"

		store.EXPECT().
			GetImport(importID).
			Return(
				&model.Import{
					ID:            importID,
					TranslationID: translationID},
				nil).
			Times(1)

		store.EXPECT().
			UpdateImport(
				&model.Import{
					ID:            importID,
					CompleteAt:    1000,
					Error:         "",
					TranslationID: translationID}).
			Return(nil).
			Times(1)

		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/import", ts.URL),
			strings.NewReader(
				`{"ID":"importID", "CompleteAt": 1000, "Error": ""}`,
			))
		require.NoError(t, err)

		req.Header.Add("content-type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("release lock on a locked Import", func(t *testing.T) {
		importID := "importID"
		translationID := "translationID"

		store.EXPECT().
			GetImport(importID).
			Return(
				&model.Import{
					ID:            importID,
					TranslationID: translationID},
				nil).
			Times(1)

		store.EXPECT().
			UpdateImport(
				&model.Import{
					ID:            importID,
					LockedBy:      "",
					TranslationID: translationID}).
			Return(nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/import/%s/release", ts.URL, importID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("get an import by Installation ID", func(t *testing.T) {
		importID := "importID"
		installationID := "installationID"
		translationID := "translationID"

		store.EXPECT().
			GetImportsByInstallation(installationID).
			Return([]*model.Import{{ID: importID, TranslationID: translationID}}, nil).
			Times(1)

		store.EXPECT().
			GetTranslation(translationID).
			Return(&model.Translation{ID: translationID}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/installation/import/%s", ts.URL, installationID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		imports, err := model.NewImportStatusListFromReader(resp.Body)
		require.NoError(t, err)
		require.NotEmpty(t, imports)
		assert.Equal(t, importID, imports[0].ID)
	})
}
