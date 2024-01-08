package mockcontext

import "github.com/mattermost/awat/internal/mocks"

// MockAWS is a mock implementation of the AWS interface used for testing.
type MockAWS struct {
	ResourceExists bool

	dummyArchiveFilePath string
}

// GetBucketName returns a mock bucket name.
func (a *MockAWS) GetBucketName() string {
	return "test"
}

// CheckBucketFileExists returns a mocked response indicating whether a resource exists in the bucket.
func (a *MockAWS) CheckBucketFileExists(file string) (bool, error) {
	return a.ResourceExists, nil
}

// UploadArchiveToS3 mocks the upload process of an archive to S3.
func (a *MockAWS) UploadArchiveToS3(uploadFileName, destKeyName string) error {
	return nil
}

// DownloadArchiveFromS3 mocks the download process of an archive from S3.
func (a *MockAWS) DownloadArchiveFromS3(archiveName string) (string, func(), error) {
	if a.dummyArchiveFilePath == "" {
		var err error
		a.dummyArchiveFilePath, err = mocks.GenerateDummyMattermostArchive()
		if err != nil {
			return "", func() {}, err
		}
	}

	return a.dummyArchiveFilePath, func() {}, nil
}
