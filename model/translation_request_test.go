package model_test

import (
	"testing"

	"github.com/mattermost/awat/model"
	"github.com/stretchr/testify/assert"
)

func TestTranslationRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		requireError bool
		request      *model.TranslationRequest
	}{
		{
			"no installation ID",
			true,
			&model.TranslationRequest{
				Type:    model.MattermostWorkspaceBackupType,
				Archive: "test.zip",
			},
		},
		{
			"no type",
			true,
			&model.TranslationRequest{
				InstallationID: model.NewID(),
				Archive:        "test.zip",
			},
		},
		{
			"slack, but no team",
			true,
			&model.TranslationRequest{
				Type:           model.SlackWorkspaceBackupType,
				InstallationID: model.NewID(),
				Archive:        "test.zip",
			},
		},
		{
			"not zip file",
			true,
			&model.TranslationRequest{
				Type:           model.MattermostWorkspaceBackupType,
				InstallationID: model.NewID(),
				Archive:        "test.tar.gz",
			},
		},
		{
			"no filename",
			true,
			&model.TranslationRequest{
				Type:           model.MattermostWorkspaceBackupType,
				InstallationID: model.NewID(),
				Archive:        ".zip",
			},
		},
		{
			"valid",
			false,
			&model.TranslationRequest{
				Type:           model.MattermostWorkspaceBackupType,
				InstallationID: model.NewID(),
				Archive:        "test.zip",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			if tc.requireError {
				assert.Error(t, tc.request.Validate())
			} else {
				assert.NoError(t, tc.request.Validate())
			}
		})
	}
}
