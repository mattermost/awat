// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package slack

import (
	"archive/zip"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO make it so that this doesn't actually reach out to Slack's servers..
// but it's only a few MB and it's public and probably nobody will notice or care
func TestFetchAttachedFiles(t *testing.T) {
	tempFile, err := ioutil.TempFile(os.TempDir(), "awat-slack-unittest")
	require.NoError(t, err)
	logger := logrus.New()

	err = FetchAttachedFiles(logger, "../../test/dummy-slack-workspace-archive.zip", tempFile.Name())
	assert.NoError(t, err)

	zr, err := zip.OpenReader(tempFile.Name())
	require.NoError(t, err)

	uploads := map[string]bool{
		"__uploads/F01TP8NLE00/kitten1.jpg":                 false,
		"__uploads/F01SZEKQ2H0/20200609_151659.jpg":         false,
		"__uploads/F01TC38M2HX/emacs.png":                   false,
		"__uploads/F01SJM9E2BZ/Headphone_Stand_2_parts.zip": false,
		"__uploads/F01SZLXD3PD/Ergodox_Tent.zip":            false,
		"__uploads/F01SWCK89C5/Untitled":                    false,
		"__uploads/F01T5KQN1U4/Untitled":                    false,
		"__uploads/F01SZG8FHFU/20200310_121556.jpg":         false,
		"__uploads/F01TPAKE3AL/20200303_131253.jpg":         false,
		"__uploads/F01TC55H8TB/20190609_200033.jpg":         false,
	}

	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "__uploads") {
			continue
		}
		_, ok := uploads[f.Name]
		assert.True(t, ok)
		uploads[f.Name] = true
	}

	for _, v := range uploads {
		assert.True(t, v)
	}
}
