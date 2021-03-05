package slack

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mattermost/awat/internal/model"
)

type SlackTranslator struct{}

func (_ *SlackTranslator) Translate(translation *model.Translation, bucket string) error {
	logger := logrus.New().WithField("translation", translation.ID)

	// fetch messages using .. the other tool,
	sess := session.Must(session.NewSession())

	downloader := s3manager.NewDownloader(sess)

	archive, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("%s-", translation.ID))
	if err != nil {
		return errors.Wrap(err, "failed to open temp file to download input archive to")
	}

	nBytes, err := downloader.Download(archive,
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &translation.Resource,
		})

	if err != nil {
		return errors.Wrapf(err, "failed to download %s from bucket %s", translation.Resource, bucket)
	}

	err = archive.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close temporary file after writing incoming archive to it")
	}

	logger.Debugf("Successfully downloaded %d bytes from bucket %s key %s",
		nBytes, bucket, translation.Resource)

	// fetch attached files and store them in S3
	err = FetchAttachedFiles(translation.Resource, archive.Name(), bucket)
	if err != nil {
		return err
	}

	// transform the actual messages
	err = TransformSlackMessages(
		archive.Name(),
		fmt.Sprintf("%s-MBIF.jsonl", archive.Name()),
		translation.Team)

	if err != nil {
		return errors.Wrap(err, "failed to transform Slack archive to MBIF")
	}

	logger.Infof("Successfully translated %s", translation.ID)
	return nil
}
