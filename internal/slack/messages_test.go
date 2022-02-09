// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package slack

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/awat/model"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformSlack(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "awat-slack-test-transform-slack")
	require.NoError(t, err)

	os.MkdirAll(tempDir+"/attachments", 0666)

	mbifOutputFile, err := ioutil.TempFile(tempDir, "mbif")
	require.NoError(t, err)

	err = TransformSlack(&model.Translation{
		ID:             model.NewID(),
		InstallationID: model.NewID(),
		Team:           "some team",
		Users:          100,
		Type:           "slack",
		Resource:       "dummy-slack-workspace-archive.zip",
		CreateAt:       time.Now().UnixNano() / 1000,
		StartAt:        time.Now().UnixNano() / 1000,
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
	mbifOutputFile.Close()

	mbifOutputFile, err = os.Open(mbifOutputFile.Name())
	require.NoError(t, err)
	mbifRaw, _ := ioutil.ReadAll(mbifOutputFile)
	lines := strings.SplitAfter(string(mbifRaw), "\n")

	found := false
	// find a known line of output just to make sure things went more or less okay
	for _, l := range lines {
		if strings.HasPrefix(l, `{"type":"post","post":{"team":"some team","channel":"testingsomemoreee","user":"jason_24","message":"@jasonbot","props":null,`) {
			found = true
		}
	}

	assert.True(t, found)
}
