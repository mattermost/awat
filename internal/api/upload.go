// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mattermost/awat/internal/validators"
	"github.com/mattermost/awat/model"
)

func handleReceiveArchive(c *Context, w http.ResponseWriter, r *http.Request) {
	uploadFile, err := os.CreateTemp(c.Workdir, "upload-")
	if err != nil {
		c.Logger.Error("failed to open temp file to write upload to")
	}

	uploadLengthString := r.Header.Get("Content-Length")
	if uploadLengthString == "" {
		c.Logger.Debugln(r.Header)
		c.Logger.Error("Content-Length header must be set")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate parameters
	params, _ := model.NewArchiveUploadFromURLQuery(r.URL.Query())
	if err := params.Validate(); err != nil {
		c.Logger.WithError(err).Error("import parameters validation failed")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	uploadLength, err := strconv.Atoi(uploadLengthString)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to convert content-length %s to an integer", uploadLengthString)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Logger.Debugf("receiving file with size %d", uploadLength)
	uploadID := model.NewID()
	destKeyName := uploadID + ".zip"

	totalWritten, err := io.Copy(uploadFile, r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to copy body to temp file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if totalWritten != int64(uploadLength) {
		c.Logger.Errorf("written bytes %d doesn't match object size %d", totalWritten, uploadLength)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if params.Type == model.MattermostWorkspaceBackupType {
		c.Logger.Info("Validating upload")
		validator := validators.NewMattermostValidator()
		if err := validator.Validate(uploadFile.Name()); err != nil {
			c.Logger.WithError(err).Error("archive validation failed")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}

	err = c.Store.CreateUpload(uploadID, params.Type)
	if err != nil {
		c.Logger.WithError(err).Error("failed to store upload ID for tracking progress")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Logger.Debugf("finished reading and writing file; %d bytes written", totalWritten)
	go func(context *Context, uploadID, uploadFileName, destinationKeyName string) {
		err = c.AWS.UploadArchiveToS3(uploadFileName, destinationKeyName)
		defer os.Remove(uploadFileName)
		if err != nil {
			c.Logger.WithError(err).Error("failed to upload file to S3")
			storageErr := c.Store.CompleteUpload(uploadID, err.Error())
			if storageErr != nil {
				c.Logger.WithError(err).Errorf("failed to mark upload %s failed with error %s",
					uploadID, err.Error())
			}
		} else {
			err = c.Store.CompleteUpload(uploadID, "")
			if err != nil {
				c.Logger.WithError(err).Errorf("failed to mark upload %s complete without error", uploadID)
			}
		}
	}(c, uploadID, uploadFile.Name(), destKeyName)

	w.Header().Add("content-type", "text/plain")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(destKeyName))
}

func handleCheckUploadStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uploadID, ok := vars["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	upload, err := c.Store.GetUpload(uploadID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to look up upload %s", uploadID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if upload == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	output, err := json.Marshal(upload)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to marshal Upload %s", uploadID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(output)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to marshal Upload %s", uploadID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
