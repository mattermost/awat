// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

type AWS interface {
	GetBucketName() string
	CheckBucketFileExists(file string) (bool, error)
	UploadArchiveToS3(uploadFileName, destKeyName string) error
	DownloadArchiveFromS3(filename string) (string, error)
}
type AWSContext struct {
	Session *session.Session
	Bucket  string
}

func (a *AWSContext) GetBucketName() string {
	return a.Bucket
}

func (a *AWSContext) CheckBucketFileExists(file string) (bool, error) {
	s3client := s3.New(a.Session)
	_, err := s3client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(a.Bucket),
		Key:    aws.String(file),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return false, errors.Errorf("bucket %s does not exist", a.Bucket)
			case "NotFound": // s3.ErrCodeNoSuchKey does not work, aws is missing this error code so we hardwire a string
				return false, nil
			default:
				return false, err
			}
		}
		return false, err
	}

	return true, nil
}

func (a *AWSContext) UploadArchiveToS3(uploadFileName, destKeyName string) error {
	s3client := s3.New(a.Session)
	uploadFile, err := os.Open(uploadFileName)
	if err != nil {
		return errors.New("failed to open file before upload")
	}

	uploader := s3manager.NewUploaderWithClient(s3client)
	_, err = uploader.Upload(
		&s3manager.UploadInput{
			Bucket: aws.String(a.GetBucketName()),
			Key:    &destKeyName,
			Body:   uploadFile,
		})
	return err
}

func (a *AWSContext) DownloadArchiveFromS3(archiveName string) (string, error) {
	s3client := s3.New(a.Session)

	tempFile, err := os.CreateTemp("", "awat-archive-")
	if err != nil {
		return "", errors.Wrap(err, "error creating tempoorary file to write to")
	}

	downloader := s3manager.NewDownloaderWithClient(s3client)

	_, err = downloader.Download(tempFile, &s3.GetObjectInput{
		Bucket: aws.String(a.GetBucketName()),
		Key:    aws.String(archiveName),
	})
	if err != nil {
		return "", errors.Wrap(err, "error downloading from s3")
	}

	return tempFile.Name(), nil
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
