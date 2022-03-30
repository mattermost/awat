// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package slack

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/mattermost/awat/model"
	mmetl "github.com/mattermost/mmetl/services/slack"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformSlack(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "awat-slack-test-transform-slack")
	require.NoError(t, err)

	err = os.MkdirAll(tempDir+"/attachments", 0666)
	require.NoError(t, err)

	mbifOutputFile, err := ioutil.TempFile(tempDir, "mbif")
	require.NoError(t, err)
	defer mbifOutputFile.Close()

	err = TransformSlack(&model.Translation{
		ID:             model.NewID(),
		InstallationID: model.NewID(),
		Team:           "some team",
		Users:          0,
		Type:           model.SlackWorkspaceBackupType,
		Resource:       "dummy-slack-workspace-archive.zip",
		CreateAt:       model.GetMillis(),
		StartAt:        model.GetMillis(),
		CompleteAt:     0,
		LockedBy:       "",
	},
		"../../test/dummy-slack-workspace-archive.zip",
		mbifOutputFile.Name(),
		tempDir+"/attachments",
		tempDir,
		log.New(),
	)
	require.NoError(t, err)

	mbifOutputFile, err = os.Open(mbifOutputFile.Name())
	require.NoError(t, err)
	defer mbifOutputFile.Close()

	mbifRaw, err := ioutil.ReadAll(mbifOutputFile)
	require.NoError(t, err)
	lines := strings.SplitAfter(string(mbifRaw), "\n")

	found := false
	// find a known line of output just to make sure things went more or less okay
	for _, l := range lines {
		if strings.HasPrefix(l, `{"type":"post","post":{"team":"some team","channel":"testingsomemoreee","user":"jason_24","type":null,"message":"@jasonbot","props":null,`) {
			found = true
		}
	}

	// TODO: greatly improve checking in tests
	assert.True(t, found)
	assert.GreaterOrEqual(t, len(lines), 200)
}

func TestValidateIntermediate(t *testing.T) {
	var testCases = []struct {
		name         string
		intermediate *mmetl.Intermediate
		valid        bool
	}{
		{"no users", &mmetl.Intermediate{Posts: []*mmetl.IntermediatePost{{Message: "test"}}}, false},
		{"no posts", &mmetl.Intermediate{UsersById: map[string]*mmetl.IntermediateUser{"user1": {Username: "user1"}}}, false},
		{"valid", &mmetl.Intermediate{UsersById: map[string]*mmetl.IntermediateUser{"user1": {Username: "user1"}}, Posts: []*mmetl.IntermediatePost{{Message: "test"}}}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.valid {
				assert.NoError(t, validateIntermediate(tc.intermediate))
			} else {
				assert.Error(t, validateIntermediate(tc.intermediate))
			}
		})
	}
}
