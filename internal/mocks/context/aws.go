package mock_context

import "github.com/mattermost/awat/internal/mocks"

type MockAWS struct {
	ResourceExists bool

	dummyArchiveFilePath string
}

func (a *MockAWS) GetBucketName() string {
	return "test"
}

func (a *MockAWS) CheckBucketFileExists(file string) (bool, error) {
	return a.ResourceExists, nil
}

func (a *MockAWS) UploadArchiveToS3(uploadFileName, destKeyName string) error {
	return nil
}

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
