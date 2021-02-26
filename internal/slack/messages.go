package slack

import (
	"archive/zip"
	"log"
	"os"

	mmetlSlack "github.com/mattermost/mmetl/services/slack"
)

func TransformSlack(inputFilePath, outputFilePath, team, attachmentsDir string, skipConvertPosts, skipAttachments, discardInvalidProps bool) error {
	// input file
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
	if err != nil || zipReader.File == nil {
		return err
	}

	slackExport, err := mmetlSlack.ParseSlackExportFile(team, zipReader, skipConvertPosts)
	if err != nil {
		return err
	}

	intermediate, err := mmetlSlack.Transform(slackExport, attachmentsDir, skipAttachments)
	if err != nil {
		return err
	}

	if err = mmetlSlack.Export(team, intermediate, outputFilePath); err != nil {
		return err
	}

	log.Println("Transformation succeeded!!")
	return nil
}
