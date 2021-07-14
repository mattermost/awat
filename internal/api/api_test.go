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
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mock_api "github.com/mattermost/awat/internal/mocks/api"
	"github.com/mattermost/awat/model"
)

func TestGetTranslations(t *testing.T) {
	logger := MakeLogger(t)
	mockController := gomock.NewController(t)
	store := mock_api.NewMockStore(mockController)
	router := mux.NewRouter()
	Register(router, &Context{
		Store:  store,
		Logger: logger,
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
		// translation :=
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
		// translation :=
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
		// translation :=
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
}

// MakeLogger creates a log.FieldLogger that routes to tb.Log.
func MakeLogger(tb testing.TB) log.FieldLogger {
	logger := log.New()
	logger.SetOutput(&testingWriter{tb})
	logger.SetLevel(log.TraceLevel)

	return logger
}

// testingWriter is an io.Writer that writes through t.Log.
type testingWriter struct {
	tb testing.TB
}

func (tw *testingWriter) Write(b []byte) (int, error) {
	tw.tb.Log(strings.TrimSpace(string(b)))
	return len(b), nil
}
