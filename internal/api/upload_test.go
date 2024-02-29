// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mock_api "github.com/mattermost/awat/internal/mocks/api"
	"github.com/mattermost/awat/internal/testlib"
	"github.com/mattermost/awat/model"
)

func TestUpload(t *testing.T) {
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

	t.Run("unknown upload", func(t *testing.T) {
		store.EXPECT().
			GetUpload("bogusID").
			Return(nil, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/upload/bogusID", ts.URL))
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("fetch an upload successfully", func(t *testing.T) {
		uploadID := model.NewID()

		store.EXPECT().
			GetUpload(uploadID).
			Return(&model.Upload{ID: uploadID}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/upload/%s", ts.URL, uploadID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		upload, err := model.NewUploadFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, uploadID, upload.ID)
	})

	t.Run("fetch upload, internal DB error", func(t *testing.T) {
		uploadID := model.NewID()
		store.EXPECT().
			GetUpload(uploadID).
			Return(nil, errors.New("problem talking to database")).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/upload/%s", ts.URL, uploadID))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("fetch all imports", func(t *testing.T) {
		store.EXPECT().
			GetUploads().
			Return([]*model.Upload{
				{ID: model.NewID()},
				{ID: model.NewID()},
				{ID: model.NewID()},
			}, nil).
			Times(1)

		resp, err := http.Get(fmt.Sprintf("%s/uploads", ts.URL))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		imports, err := model.NewUploadListFromReader(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, 3, len(imports))
	})

}
