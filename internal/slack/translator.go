package slack

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mattermost/awat/internal/model"
)

type SlackTranslator struct {
	bucket     string
	workingDir string
}

func NewSlackTranslator(bucket, workingDir string) *SlackTranslator {
	return &SlackTranslator{bucket: bucket, workingDir: workingDir}
}

func (st *SlackTranslator) Translate(translation *model.Translation) error {
	workdir := fmt.Sprintf("%s/%s", st.workingDir, translation.ID)
	err := os.Mkdir(workdir, 0700)
	if err != nil {
		return err
	}

	logger := logrus.New().WithField("translation", translation.ID)

	// fetch messages using .. the other tool,
	sess := session.Must(session.NewSession())

	downloader := s3manager.NewDownloader(sess)

	inputArchive, err := os.Create(workdir + "/input.zip")
	if err != nil {
		return errors.Wrap(err, "failed to open temp file to download input archive to")
	}

	nBytes, err := downloader.Download(inputArchive,
		&s3.GetObjectInput{
			Bucket: &st.bucket,
			Key:    &translation.Resource,
		})

	if err != nil {
		return errors.Wrapf(err, "failed to download %s from bucket %s", translation.Resource, st.bucket)
	}

	err = inputArchive.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close temporary file after writing incoming archive to it")
	}

	logger.Debugf("Successfully downloaded %d bytes from bucket %s key %s",
		nBytes, st.bucket, translation.Resource)

	withFiles, err := os.Create(workdir + "/inputWithFiles.zip")
	if err != nil {
		return errors.Wrap(err, "failed to open temp file to convert input archive to")
	}

	err = FetchAttachedFiles(inputArchive.Name(), withFiles.Name())
	if err != nil {
		return errors.Wrap(err, "failed to fetch attached files")
	}

	err = os.Remove(inputArchive.Name())
	if err != nil {
		logger.Errorf("failed to remove file %s", inputArchive.Name())
	}

	attachmentDirName := fmt.Sprintf("%s/attachments", workdir)
	err = os.Mkdir(attachmentDirName, 0700)
	if err != nil {
		return errors.Wrap(err, "failed to create attachments directory")
	}

	mbifName := workdir + "/" + translation.InstallationID + "_MBIF.jsonl"

	logger.Infof("Downloading attached files to %s", attachmentDirName)
	err = TransformSlack(
		withFiles.Name(),
		mbifName,
		translation.Team,
		attachmentDirName)

	if err != nil {
		return err
	}

	if err != nil {
		return errors.Wrap(err, "failed to transform Slack archive to MBIF")
	}

	output, err := os.Create(fmt.Sprintf("%s/%s.zip", st.workingDir, translation.ID))
	if err != nil {
		return err
	}
	defer output.Close()

	outputZipfile := zip.NewWriter(output)
	defer outputZipfile.Close()

	mbifInOutputZipfile, err := outputZipfile.Create("MBIF.jsonl")
	if err != nil {
		return err
	}

	mbifInputFile, err := os.Open(mbifName)
	if err != nil {
		return err
	}

	_, err = io.Copy(mbifInOutputZipfile, mbifInputFile)
	if err != nil {
		return err
	}

	mbifInputFile.Close()

	attachmentFiles, err := ioutil.ReadDir(attachmentDirName)
	if err != nil {
		return err
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

	logger.Infof("Finished translation %s", translation.ID)
	return nil
}
