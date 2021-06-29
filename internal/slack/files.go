// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE for license information.
//

package slack

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// FetchAttachedFiles takes the Slack input file at absolute path
// inputArchive and parses it. Upon discovering references to attached
// files, those files are fetched from Slack's servers and added to
// outputArchive, which at the end will contain all of the data from
// inputArchive as well as all attached files
func FetchAttachedFiles(logger logrus.FieldLogger, inputArchive string, outputArchive string) error {
	// Open the input archive.
	r, err := zip.OpenReader(inputArchive)
	if err != nil {
		return fmt.Errorf("could not open input archive for reading: %s\n", inputArchive)
	}
	defer r.Close()

	// Open the output archive.
	f, err := os.Create(outputArchive)
	if err != nil {
		return fmt.Errorf("could not open the output archive for writing: %s\n\n%s", outputArchive, err)
	}
	defer f.Close()

	// Create a zip writer on the output archive.
	w := zip.NewWriter(f)

	// Run through all the files in the input archive.
	for _, file := range r.File {

		// Open the file from the input archive.
		inReader, err := file.Open()
		if err != nil {
			logger.WithError(err).Errorf("failed to open file in input archive: %s", file.Name)
			continue
		}

		// Read the file into a byte array.
		inBuf, err := ioutil.ReadAll(inReader)
		if err != nil {
			logger.WithError(err).Errorf("Failed to read file in input archive: %s\n\n%s", file.Name, err)
			continue
		}

		// Now write this file to the output archive.
		outFile, err := w.Create(file.Name)
		if err != nil {
			return errors.Wrapf(err, "failed to create file in output archive: %s", file.Name)
		}
		_, err = outFile.Write(inBuf)
		if err != nil {
			logger.WithError(err).Errorf("failed to write file in output archive: %s", file.Name)
			continue
		}

		// Check if the file name matches the pattern for files we need to parse.
		splits := strings.Split(file.Name, "/")
		if len(splits) == 2 && !strings.HasPrefix(splits[0], "__") && strings.HasSuffix(splits[1], ".json") {
			// Parse this file.
			err = processChannelPostsWithFiles(logger, w, file.Name, inBuf)
			if err != nil {
				logger.WithError(err).Errorf("failed to process file %s", file.Name)
				continue
			}
		}
	}

	// Close the output zip writer.
	err = w.Close()
	if err != nil {
		logger.WithError(err).Warnf("failed to close the output archive %s", outputArchive)
	}

	return nil
}

// processChannelPostsWithFiles actually fetches and adds a found file to the
// archive specified at file
func processChannelPostsWithFiles(logger logrus.FieldLogger, w *zip.Writer, fileName string, inBuf []byte) error {
	// Parse the JSON of the file.
	var posts []SlackPost
	if err := json.Unmarshal(inBuf, &posts); err != nil {
		return errors.Wrapf(err, "failed to parse the JSON file: %s", fileName)
	}

	// Loop through all the posts.
	for _, post := range posts {
		// Support for legacy file_share posts.
		if post.Subtype == "file_share" {
			// Check there's a File property.
			if post.File == nil {
				logger.Warnf("file_share post has no File property: %s", post.Ts)
				continue
			}

			// Add the file as a single item in the array of the post's files.
			post.Files = []*SlackFile{post.File}
		}

		// If the post doesn't contain any files, move on.
		if post.Files == nil {
			continue
		}

		// Loop through all the files.
		for _, file := range post.Files {
			processSingleFile(logger, w, file, &post)
		}
	}

	return nil
}

func processSingleFile(logger logrus.FieldLogger, w *zip.Writer, file *SlackFile, post *SlackPost) error {
	// Check there's an Id, Name and either UrlPrivateDownload or UrlPrivate property.
	if len(file.Id) < 1 || len(file.Name) < 1 || !(len(file.UrlPrivate) > 0 || len(file.UrlPrivateDownload) > 0) {
		return errors.New("file_share post has missing properties on it's File object: " + post.Ts + "\n")
	}

	// Figure out the download URL to use.
	var downloadUrl string
	if len(file.UrlPrivateDownload) > 0 {
		downloadUrl = file.UrlPrivateDownload
	} else {
		downloadUrl = file.UrlPrivate
	}

	// Build the output file path.
	outputPath := "__uploads/" + file.Id + "/" + file.Name

	// Create the file in the zip output file.
	outFile, err := w.Create(outputPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create output file in output archive: %s", outputPath)
	}

	// Fetch the file.
	response, err := http.Get(downloadUrl)
	if err != nil {
		return errors.Wrapf(err, "failed to download the file: %s", downloadUrl)
	}
	defer response.Body.Close()

	// Save the file to the output zip file.
	_, err = io.Copy(outFile, response.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to write the downloaded file to the output archive: %s", outputPath)
	}

	// Success at last.
	logger.Debugf("Downloaded attachment into output archive: %s.\n", file.Id)
	return nil
}

// SlackFile is a holding type for files attached to Slack messages
type SlackFile struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	UrlPrivate         string `json:"url_private"`
	UrlPrivateDownload string `json:"url_private_download"`
}

// SlackPost is a holding type for Slack posts
type SlackPost struct {
	User    string       `json:"user"`
	Type    string       `json:"type"`
	Subtype string       `json:"subtype"`
	Text    string       `json:"text"`
	Ts      string       `json:"ts"`
	File    *SlackFile   `json:"file"`
	Files   []*SlackFile `json:"files"`
}

// SlackUser is a holding type for Users in the Slack representation
// As it appears in users.json and /api/users.list.  There're
// obviously many more fields, but we only need a couple of them.
type SlackUser struct {
	Id      string           `json:"id"`
	Profile SlackUserProfile `json:"profile"`
}

// SlackUserProfile is a holding type for extracting users' emails out
// of a Slack export archive
type SlackUserProfile struct {
	Email string `json:"email"`
}
