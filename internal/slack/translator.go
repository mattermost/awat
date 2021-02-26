package slack

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
		return err
	}

	nBytes, err := downloader.Download(archive,
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &translation.Resource,
		})

	if err != nil {
		return err
	}

	err = archive.Close()
	if err != nil {
		return err
	}

	logger.Debugf("Successfully downloaded %d bytes from bucket %s key %s",
		nBytes, bucket, translation.Resource)

	// fetch attached files
	err = FetchAttachedFiles(archive.Name(), bucket)
	if err != nil {
		return err
	}

	//TODO translate messages, too

	return nil
}
