package mocks

import (
	"archive/zip"
	"os"
	"path/filepath"
)

const dummyMattermostArchiveContents = `{"type":"version","version":1}`

// GenerateDummyMattermostArchive creates a dummy zip file in a temporary directory so we can use
// as a mock archive for tests and properly test flows where archive validation is present.
func GenerateDummyMattermostArchive() (archivePath string, err error) {
	tempDir, err := os.MkdirTemp("", "awat-test")
	if err != nil {
		return archivePath, err
	}
	archivePath = filepath.Join(tempDir, "archive.zip")

	archive, err := os.Create(archivePath)
	if err != nil {
		panic(err)
	}
	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	zipDataFile, err := zipWriter.Create("data.jsonl")
	if err != nil {
		panic(err)
	}
	if _, err := zipDataFile.Write([]byte(dummyMattermostArchiveContents)); err != nil {
		return archivePath, err
	}

	return
}
