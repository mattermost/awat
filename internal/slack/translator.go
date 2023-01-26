// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package slack

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/awat/model"
)

type SlackTranslator struct {
	bucket             string
	workingDir         string
	outputZipLocalPath string
}

func NewSlackTranslator(bucket, workingDir string) *SlackTranslator {
	return &SlackTranslator{bucket: bucket, workingDir: workingDir}
}

// Translate satisfies the Translator interface for the
// SlackTranslator. It performs the Translation represented by the
// input struct and uploads the resulting .zip archive to S3. On
// success it returns the file name of the output zip file without a
// path and on error it returns the error and an empty string
func (st *SlackTranslator) Translate(translation *model.Translation) (string, error) {
	workdir := fmt.Sprintf("%s/%s", st.workingDir, translation.ID)
	err := os.Mkdir(workdir, 0700)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workdir)

	logger := log.New()

	inputArchiveName, err := st.fetchSlackArchive(logger, workdir, translation.Resource)
	if err != nil {
		return "", err
	}

	attachmentDirName := fmt.Sprintf("%s/attachments", workdir)
	archiveWithFilesName, err := st.addFilesToSlackArchive(
		logger,
		workdir,
		attachmentDirName,
		inputArchiveName,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed add files to slack archive")
	}

	mbifName := fmt.Sprintf("%s/%s_MBIF.jsonl", workdir, translation.InstallationID)
	logger.Infof("Transforming Slack archive for Translation %s to MBIF", translation.ID)
	err = TransformSlack(
		translation,
		archiveWithFilesName,
		mbifName,
		attachmentDirName,
		workdir,
		logger,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to transform Slack archive to MBIF")
	}

	logger.Infof("Preparing Mattermost archive for Translation %s for upload", translation.ID)
	st.outputZipLocalPath, err = st.createOutputZipfile(logger, attachmentDirName, mbifName, translation.ID)
	if err != nil {
		return "", err
	}

	logger.Infof("Uploading Mattermost archive for Translation %s", translation.ID)
	err = st.uploadTransformedZip(st.outputZipLocalPath, st.bucket)
	if err != nil {
		return "", err
	}

	outputNameSplitPath := strings.Split(st.outputZipLocalPath, "/")
	outputShortName := outputNameSplitPath[len(outputNameSplitPath)-1]
	logger.Infof("Finished translation %s", translation.ID)

	return outputShortName, nil
}

func (st *SlackTranslator) GetOutputArchiveLocalPath() (string, error) {
	return st.outputZipLocalPath, nil
}

func (st *SlackTranslator) Cleanup() error {
	if st.outputZipLocalPath == "" {
		return nil
	}

	return os.Remove(st.outputZipLocalPath)
}

// fetchSlackArchive is responsible for downloading the input archive
// from S3 and writing it out to workdir, which is assumed to be of
// sufficient capacity for the archive
func (st *SlackTranslator) fetchSlackArchive(logger log.FieldLogger, workdir, resource string) (string, error) {
	sess := session.Must(session.NewSession())

	downloader := s3manager.NewDownloader(sess)

	inputArchive, err := os.Create(workdir + "/input.zip")
	if err != nil {
		return "", errors.Wrap(err, "failed to open temp file to download input archive to")
	}

	nBytes, err := downloader.Download(inputArchive,
		&s3.GetObjectInput{
			Bucket: &st.bucket,
			Key:    &resource,
		})

	if err != nil {
		return "", errors.Wrapf(err, "failed to download %s from bucket %s", resource, st.bucket)
	}

	err = inputArchive.Close()
	if err != nil {
		return "", errors.Wrap(err, "failed to close temporary file after writing incoming archive to it")
	}

	logger.Debugf("Successfully downloaded %d bytes from bucket %s key %s",
		nBytes, st.bucket, resource)

	return inputArchive.Name(), nil
}

// addFilesToSlackArchive prepares the input and fetches attached
// files, writing the output to workdir and removing the input archive
// when complete
func (st *SlackTranslator) addFilesToSlackArchive(logger log.FieldLogger, workdir, attachmentDirName, inputArchiveName string) (string, error) {
	defer func() {
		err := os.Remove(inputArchiveName)
		if err != nil {
			logger.Errorf("failed to remove file %s", inputArchiveName)
		}
	}()

	logger.Infof("Downloading attached files to %s", attachmentDirName)

	withFiles, err := os.Create(workdir + "/inputWithFiles.zip")
	if err != nil {
		return "", errors.Wrap(err, "failed to open temp file to convert input archive to")
	}

	err = FetchAttachedFiles(logger, inputArchiveName, withFiles.Name())
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch attached files")
	}

	err = os.MkdirAll(attachmentDirName, 0700)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create attachments directory %s", attachmentDirName)
	}

	return withFiles.Name(), nil
}

// createOutputZip file compresses the output from the Translate
// process into a .zip that can be injested by Mattermost
func (st *SlackTranslator) createOutputZipfile(logger log.FieldLogger, attachmentDirName, mbifName, translationID string) (string, error) {
	output, err := os.Create(fmt.Sprintf("%s/%s.zip", st.workingDir, translationID))
	if err != nil {
		return "", err
	}
	defer output.Close()

	outputZipfile := zip.NewWriter(output)
	defer outputZipfile.Close()

	mbifInOutputZipfile, err := outputZipfile.Create("MBIF.jsonl")
	if err != nil {
		return "", err
	}

	mbifInputFile, err := os.Open(mbifName)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(mbifInOutputZipfile, mbifInputFile)
	if err != nil {
		return "", err
	}

	mbifInputFile.Close()

	attachmentFiles, err := ioutil.ReadDir(attachmentDirName)
	if err != nil {
		return "", err
	}

	for _, attachment := range attachmentFiles {
		if attachment.IsDir() {
			continue
		}
		attachmentInZipfile, err := outputZipfile.Create(fmt.Sprintf("/data/attachments/%s", attachment.Name()))
		if err != nil {
			logger.WithError(err).Error("failed to write attachment")
			continue
		}
		attachmentFile, err := os.Open(fmt.Sprintf("%s/%s", attachmentDirName, attachment.Name()))
		if err != nil {
			logger.WithError(err).Errorf("failed to open attachment file %s", attachmentDirName+attachment.Name())
			continue
		}
		defer attachmentFile.Close()
		_, err = io.Copy(attachmentInZipfile, attachmentFile)
		if err != nil {
			logger.
				WithError(err).
				Errorf("failed to copy attachment %s", attachment.Name())
			continue
		}
	}

	return output.Name(), nil
}

// uploadTransformedZip uploads the prepared Mattermost-compatible
// archive to S3 for future import
func (st *SlackTranslator) uploadTransformedZip(output, bucket string) error {
	sess := session.Must(session.NewSession())
	uploader := s3manager.NewUploader(sess)
	body, err := os.Open(output)
	if err != nil {
		return nil
	}
	defer body.Close()

	outputNameSplitPath := strings.Split(output, "/")
	outputShortName := outputNameSplitPath[len(outputNameSplitPath)-1]
	_, err = uploader.Upload(
		&s3manager.UploadInput{
			Bucket: &bucket,
			Body:   body,
			Key:    &outputShortName,
		})
	if err != nil {
		return err
	}
	return nil
}
