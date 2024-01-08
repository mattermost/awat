// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"context"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/mattermost/awat/internal/common"
	cloudModel "github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	Store     Store
	Logger    logrus.FieldLogger
	AWS       AWS
	Workdir   string
	RequestID string
}

// AWS provides an interface to interact with AWS services.
type AWS interface {
	GetBucketName() string
	CheckBucketFileExists(file string) (bool, error)
	UploadArchiveToS3(uploadFileName, destKeyName string) error
	DownloadArchiveFromS3(filename string) (string, func(), error)
}

// AWSContext implements the AWS interface and provides methods to interact with AWS S3.
type AWSContext struct {
	s3Client *s3.Client
	bucket   string
}

// NewAWSContext creates a new AWSContext with the specified bucket name.
func NewAWSContext(bucket string) (*AWSContext, error) {
	awsConfig, err := common.NewAWSConfig()
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(awsConfig)

	return &AWSContext{
		s3Client: s3Client,
		bucket:   bucket,
	}, nil
}

// GetBucketName returns the name of the S3 bucket used in AWSContext.
func (a *AWSContext) GetBucketName() string {
	return a.bucket
}

// CheckBucketFileExists checks if a file exists in the S3 bucket.
func (a *AWSContext) CheckBucketFileExists(file string) (bool, error) {
	_, err := a.s3Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(a.bucket),
		Key:    aws.String(file),
	})

	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) {
			switch awsErr.ErrorCode() {
			case "NoSuchBucket":
				return false, errors.Errorf("bucket %s does not exist", a.bucket)
			case "NotFound":
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

// UploadArchiveToS3 uploads a file to an S3 bucket.
func (a *AWSContext) UploadArchiveToS3(uploadFileName, destKeyName string) error {
	uploadFile, err := os.Open(uploadFileName)
	if err != nil {
		return errors.New("failed to open file before upload")
	}

	uploader := s3manager.NewUploader(a.s3Client)
	_, err = uploader.Upload(
		context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(a.GetBucketName()),
			Key:    &destKeyName,
			Body:   uploadFile,
		})
	return err
}

// DownloadArchiveFromS3 downloads a file from an S3 bucket.
func (a *AWSContext) DownloadArchiveFromS3(archiveName string) (path string, cleanup func(), err error) {
	tempFile, err := os.CreateTemp("", "awat-archive-")
	if err != nil {
		return "", nil, errors.Wrap(err, "error creating tempoorary file to write to")
	}

	downloader := s3manager.NewDownloader(a.s3Client)

	_, err = downloader.Download(
		context.TODO(),
		tempFile,
		&s3.GetObjectInput{
			Bucket: aws.String(a.GetBucketName()),
			Key:    aws.String(archiveName),
		})
	if err != nil {
		return "", nil, errors.Wrap(err, "error downloading from s3")
	}

	path = tempFile.Name()
	cleanup = func() {
		os.Remove(path)
	}

	return path, cleanup, err
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		Store:   c.Store,
		Logger:  c.Logger,
		AWS:     c.AWS,
		Workdir: c.Workdir,
	}
}

type contextHandlerFunc func(c *Context, w http.ResponseWriter, r *http.Request)

type contextHandler struct {
	context *Context
	handler contextHandlerFunc
}

// ServeHTTP satisfies the Handler interface for contextHandler
func (h contextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := h.context.Clone()
	context.RequestID = cloudModel.NewID()
	context.Logger = context.Logger.WithFields(
		logrus.Fields{
			"path":    r.URL.Path,
			"request": context.RequestID,
		})

	h.handler(context, w, r)
}

func newContextHandler(context *Context, handler contextHandlerFunc) *contextHandler {
	return &contextHandler{
		context: context,
		handler: handler,
	}
}
