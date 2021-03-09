package slack

import (
	"archive/zip"
	"log"
	"os"

	mmetl "github.com/mattermost/mmetl/services/slack"
)

func TransformSlack(inputFilePath, outputFilePath, team, attachmentsDir string) error {
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

	slackExport, err := mmetl.ParseSlackExportFile(team, zipReader, false)
	if err != nil {
		return err
	}

	intermediate, err := mmetl.Transform(slackExport, attachmentsDir, false, true)
	if err != nil {
		return err
	}

	if err = mmetl.Export(team, intermediate, outputFilePath); err != nil {
		return err
	}

	log.Println("Transformation succeeded!!")
	return nil
}
