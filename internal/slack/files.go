package slack

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func FetchAttachments(inputArchive, destinationS3URI string) error {
	// Open the input archive.
	r, err := zip.OpenReader(inputArchive)
	if err != nil {
		return fmt.Errorf("Could not open input archive for reading: %s\n", inputArchive)
	}
	defer r.Close()

	// Run through all the files in the input archive.
	for _, file := range r.File {

		// Open the file from the input archive.
		inReader, err := file.Open()
		if err != nil {
			fmt.Printf("Failed to open file in input archive: %s\n\n%s", file.Name, err)
			os.Exit(1)
		}

		// Read the file into a byte array.
		inBuf, err := ioutil.ReadAll(inReader)
		if err != nil {
			fmt.Printf("Failed to read file in input archive: %s\n\n%s", file.Name, err)
		}

		// Now upload this file to S3
		// TODO
		if strings.HasSuffix(file.Name, "/") {
			log.Printf("found directory %s", file.Name)
			continue
		} else {
			log.Printf("found file %s", file.Name)
		}

		// Check if the file name matches the pattern for files we need to parse.
		splits := strings.Split(file.Name, "/")
		if len(splits) == 2 && strings.HasSuffix(splits[1], ".json") {
			// Parse this file.
			log.Printf("processing file %s", file.Name)
			err = processChannelFile(file, inBuf, destinationS3URI)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func processChannelFile(file *zip.File, inBuf []byte, bucket string) error {

	// Parse the JSON of the file.
	var posts []SlackPost
	if err := json.Unmarshal(inBuf, &posts); err != nil {
		return errors.New("Couldn't parse the JSON file: " + file.Name + "\n\n" + err.Error() + "\n")
	}

	// Loop through all the posts.
	for _, post := range posts {
		// Support for legacy file_share posts.
		if post.Subtype == "file_share" {
			// Check there's a File property.
			if post.File == nil {
				log.Print("file_share post has no File property: " + post.Ts + "\n")
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
			// Check there's an Id, Name and either UrlPrivateDownload or UrlPrivate property.
			if len(file.Id) < 1 || len(file.Name) < 1 || !(len(file.UrlPrivate) > 0 || len(file.UrlPrivateDownload) > 0) {
				log.Print("file_share post has missing properties on it's File object: " + post.Ts + "\n")
				continue
			}

			// Figure out the download URL to use.
			var downloadUrl string
			if len(file.UrlPrivateDownload) > 0 {
				downloadUrl = file.UrlPrivateDownload
			} else {
				downloadUrl = file.UrlPrivate
			}

			// Fetch the file.
			outputPath := file.Id + "/" + file.Name
			fmt.Printf("Downloading attachment %s to %s/%s.\n", file.Id, bucket, outputPath)
			response, err := http.Get(downloadUrl)
			if err != nil {
				log.Print("Failed to download the file: " + downloadUrl)
				continue
			}
			defer response.Body.Close()

			// The session the S3 Uploader will use
			sess := session.Must(session.NewSession())

			// Create an uploader with the session and default options
			uploader := s3manager.NewUploader(sess)
			// Success at last.

			output, err := uploader.Upload(&s3manager.UploadInput{
				Bucket: &bucket,
				Body:   response.Body,
				Key:    &outputPath,
			})

			if err != nil {
				log.Printf("failed to upload %s: %s", outputPath, err.Error())
				continue
			}

			log.Printf("uploaded %s", output.Location)
		}
	}

	return nil
}

type SlackFile struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	UrlPrivate         string `json:"url_private"`
	UrlPrivateDownload string `json:"url_private_download"`
}

type SlackPost struct {
	User    string       `json:"user"`
	Type    string       `json:"type"`
	Subtype string       `json:"subtype"`
	Text    string       `json:"text"`
	Ts      string       `json:"ts"`
	File    *SlackFile   `json:"file"`
	Files   []*SlackFile `json:"files"`
}

// As it appears in users.json and /api/users.list.
// There're obviously many more fields, but we only need a couple of them.
type SlackUser struct {
	Id      string           `json:"id"`
	Profile SlackUserProfile `json:"profile"`
}

type SlackUserProfile struct {
	Email string `json:"email"`
}
