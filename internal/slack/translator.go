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
	"github.com/sirupsen/logrus"

	"github.com/mattermost/awat/model"
)

type SlackTranslator struct {
	bucket     string
	workingDir string
}

func NewSlackTranslator(bucket, workingDir string) *SlackTranslator {
	return &SlackTranslator{bucket: bucket, workingDir: workingDir}
}

func (st *SlackTranslator) Translate(translation *model.Translation) (string, error) {
	workdir := fmt.Sprintf("%s/%s", st.workingDir, translation.ID)
	err := os.Mkdir(workdir, 0700)
	if err != nil {
		return "", err
	}

	logger := logrus.New()

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

	mbifName := fmt.Sprintf("%s/%s_MBIF.jsonl", workdir, translation.InstallationID)
	logger.Infof("Transforming Slack archive for Translation %s to MBIF", translation.ID)
	err = TransformSlack(
		archiveWithFilesName,
		mbifName,
		translation.Team,
		attachmentDirName,
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to transform Slack archive to MBIF")
	}

	outputZip, err := st.createOutputZipfile(logger, attachmentDirName, mbifName, translation.ID)
	if err != nil {
		return "", err
	}

	err = st.uploadTransformedZip(outputZip, st.bucket)
	if err != nil {
		return "", err
	}

	outputNameSplitPath := strings.Split(outputZip, "/")
	outputShortName := outputNameSplitPath[len(outputNameSplitPath)-1]
	logger.Infof("Finished translation %s", translation.ID)

	return outputShortName, nil
}

func (st *SlackTranslator) fetchSlackArchive(logger logrus.FieldLogger, workdir, resource string) (string, error) {

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

func (st *SlackTranslator) addFilesToSlackArchive(logger logrus.FieldLogger, workdir, attachmentDirName, inputArchiveName string) (string, error) {

	logger.Infof("Downloading attached files to %s", attachmentDirName)

	withFiles, err := os.Create(workdir + "/inputWithFiles.zip")
	if err != nil {
		return "", errors.Wrap(err, "failed to open temp file to convert input archive to")
	}

	err = FetchAttachedFiles(inputArchiveName, withFiles.Name())
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch attached files")
	}

	err = os.Remove(inputArchiveName)
	if err != nil {
		logger.Errorf("failed to remove file %s", inputArchiveName)
	}

	err = os.Mkdir(attachmentDirName, 0700)
	if err != nil {
		return "", errors.Wrap(err, "failed to create attachments directory")
	}

	return withFiles.Name(), nil
}

func (st *SlackTranslator) createOutputZipfile(logger logrus.FieldLogger, attachmentDirName, mbifName, translationID string) (string, error) {
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
		attachmentInZipfile, err := outputZipfile.Create(fmt.Sprintf("attachments/%s", attachment.Name()))
		if err != nil {
			logger.WithError(err).Error("failed to write attachment")
			continue
		}
		attachmentFile, err := os.Open(fmt.Sprintf("%s/%s", attachmentDirName, attachment.Name()))
		if err != nil {
			logger.WithError(err).Errorf("failed to open attachment file %s", attachmentDirName+attachment.Name())
			continue
		}
		_, err = io.Copy(attachmentInZipfile, attachmentFile)
		_ = attachmentFile.Close()
		if err != nil {
			logger.
				WithError(err).
				Error("failed to copy attachment %s", attachment.Name())
			continue
		}
	}

	return output.Name(), nil
}

func (st *SlackTranslator) uploadTransformedZip(output, bucket string) error {
	sess := session.Must(session.NewSession())
	uploader := s3manager.NewUploader(sess)
	body, err := os.Open(output)
	if err != nil {
		return nil
	}

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
