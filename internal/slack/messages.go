// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package slack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/awat/model"
	mmetl "github.com/mattermost/mmetl/services/slack"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TransformSlack takes a absolute filepath inputFilePath which points
// to a Slack workspace archive which already contains file
// attachments and outputs an MBIF to outputFilePath with references
// in the JSONL lines that make up the MBIF referring to any attached
// files in attachmentsDir. The attached files will also be extracted
// from the file at inputFilePath and stored in attachmentsDir
func TransformSlack(translation *model.Translation, inputFilePath, outputFilePath, attachmentsDir, workdir string, logger log.FieldLogger) error {
	logger.Debug("Reading zip file")

	fileReader, err := os.Open(inputFilePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	zipFileInfo, err := fileReader.Stat()
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(fileReader, zipFileInfo.Size())
	if err != nil {
		return err
	}

	logger.Debug("Running mmetl transformation processes")

	slackExport, err := mmetl.ParseSlackExportFile(translation.Team, zipReader, false)
	if err != nil {
		return err
	}

	intermediate, err := mmetl.Transform(slackExport, attachmentsDir, false, true)
	if err != nil {
		return err
	}

	// TODO maybe change mmetl to include the correct paths during
	// Transform -- however this seems to be fairly involved so for now
	// just fix these paths after the fact
	for _, post := range intermediate.Posts {
		for i, attachment := range post.Attachments {
			path, err := filepath.Abs("/data" + strings.TrimPrefix(attachment, workdir))
			if err != nil {
				return err
			}
			post.Attachments[i] = path
		}
	}

	// this total may include bots
	translation.Users = len(intermediate.UsersById)

	err = mmetl.Export(translation.Team, intermediate, outputFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to run mmetl export")
	}

	logger.Info("Transformation succeeded")
	return nil
}
