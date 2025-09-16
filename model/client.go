// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Client is the programmatic interface to the AWAT API.
type Client struct {
	address    string
	headers    map[string]string
	httpClient *http.Client
}

// NewClient creates a new instance of Client.
func NewClient(address string) *Client {
	return &Client{
		address:    address,
		headers:    make(map[string]string),
		httpClient: &http.Client{},
	}
}

// CreateTranslation creates a new Translation which will start
// shortly after being created
func (c *Client) CreateTranslation(translationRequest *TranslationRequest) (*TranslationStatus, error) {
	resp, err := c.doPost(c.buildURL("/translate"), translationRequest)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response body")
	}

	switch resp.StatusCode {
	case http.StatusAccepted:
		return NewTranslationStatusFromBytes(bodyBytes)

	default:
		return nil, errors.Errorf("failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

// GetTranslationStatus returns the TranslationStatus struct returned
// from the API for the given Translation ID
func (c *Client) GetTranslationStatus(translationID string) (*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/translation/%s", translationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)
	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, nil
	case http.StatusOK:
		return NewTranslationStatusFromReader(resp.Body)
	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetTranslationStatusesByInstallation returns all Translations that
// pertain to an Installation
func (c *Client) GetTranslationStatusesByInstallation(installationID string) ([]*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/installation/translation/%s", installationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewTranslationStatusListFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetImportStatusesByInstallation returns all Imports that
// pertain to an Installation
func (c *Client) GetImportStatusesByInstallation(installationID string) ([]*ImportStatus, error) {
	resp, err := c.doGet(c.buildURL("/installation/import/%s", installationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewImportStatusListFromReader(resp.Body)
	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetImportStatusesByTranslation retrieves all import statuses related to a translation.
func (c *Client) GetImportStatusesByTranslation(translationID string) ([]*ImportStatus, error) {
	resp, err := c.doGet(c.buildURL("/translation/%s/import", translationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewImportStatusListFromReader(resp.Body)
	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}

}

// GetAllTranslations gets all Translations from the API and returns
// them as a JSON list
func (c *Client) GetAllTranslations() ([]*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/translations"))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewTranslationStatusListFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetAllImports gets all Imports from the API and returns
// them as a JSON list
func (c *Client) GetAllImports() ([]*ImportStatus, error) {
	resp, err := c.doGet(c.buildURL("/imports"))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewImportStatusListFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetTranslationReadyToImport gets and claims the next Import waiting
// to be imported. The Import will be claimed for the caller specified
// in the ProvisionerID filed of the request argument
func (c *Client) GetTranslationReadyToImport(request *ImportWorkRequest) (*ImportStatus, error) {
	resp, err := c.doPost(c.buildURL("/import"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewImportStatusFromReader(resp.Body)
	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// ReleaseLockOnImport marks an Import as finished, with or without an error
func (c *Client) ReleaseLockOnImport(importID string) error {
	resp, err := c.doGet(c.buildURL("/import/%s/release", importID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// CompleteImport marks an Import as finished, with or without an error
func (c *Client) CompleteImport(completed *ImportCompletedWorkRequest) error {
	resp, err := c.doPut(c.buildURL("/import"), completed)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetImportStatus returns the status of a single import specified by
// ID
func (c *Client) GetImportStatus(importID string) (*ImportStatus, error) {
	resp, err := c.doGet(c.buildURL("/import/%s", importID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewImportStatusFromReader(resp.Body)
	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// closeBody ensures the Body of an http.Response is properly closed.
func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}
}

// buildURL builds a complete URL from a path and arguments.
func (c *Client) buildURL(urlPath string, args ...interface{}) string {
	return fmt.Sprintf("%s%s", c.address, fmt.Sprintf(urlPath, args...))
}

func (c *Client) doGet(u string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

func (c *Client) doPost(u string, request interface{}) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) doPut(u string, request interface{}) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPut, u, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

func (c *Client) doDelete(u string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

// GetUpload returns the Upload with the given ID.
func (c *Client) GetUpload(uploadID string) (*Upload, error) {
	resp, err := c.doGet(c.buildURL("/upload/%s", uploadID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, nil
	case http.StatusOK:
		return NewUploadFromReader(resp.Body)
	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetUploads returns all uploads from the AWAT.
func (c *Client) GetUploads() ([]*Upload, error) {
	resp, err := c.doGet(c.buildURL("/uploads"))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewUploadListFromReader(resp.Body)
	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UploadArchiveForTranslation uploads the file specified as an argument to S3 via the AWAT
func (c *Client) UploadArchiveForTranslation(filename string, archiveType BackupType) (string, error) {
	inputFile, err := os.Open(filename)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read input file %s", filename)
	}

	req, err := http.NewRequest("POST", c.buildURL("/upload?type=%s", archiveType), inputFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to create HTTP request")
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	stat, err := inputFile.Stat()
	if err != nil {
		return "", errors.Wrapf(err, "failed to determine file stats for %s", inputFile.Name())
	}

	size := stat.Size()
	if size == 0 {
		return "", errors.New("provided file appears to be empty")
	}

	req.ContentLength = size
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to send HTTP request to AWAT")
	}
	defer closeBody(resp)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to read response body")
	}

	if resp.StatusCode != http.StatusAccepted {
		return "", errors.Errorf("received unexpected code %d from AWAT: %s", resp.StatusCode, string(bodyBytes))
	}

	if bodyBytes == nil {
		return "", errors.New("invalid response data")
	}

	return string(bodyBytes), nil
}

func (c *Client) checkIfUploadComplete(uploadID string) (bool, error) {
	resp, err := http.Get(c.buildURL("/upload/%s", uploadID))
	if err != nil {
		return false, err
	}
	defer closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return false, errors.Errorf("got unexpected status code %d", resp.StatusCode)
	}

	upload := new(Upload)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(body, upload)
	if err != nil {
		return false, err
	}
	if upload.CompleteAt != 0 {
		if upload.Error != "" {
			return true, errors.New(upload.Error)
		}
		return true, nil
	}
	return false, nil
}

// WaitForUploadToComplete waits for an upload to complete on the AWAT server.
func (c *Client) WaitForUploadToComplete(uploadID string) error {
	logger := log.New()

	// 3 hour timeout, picked somewhat arbitrarily
	for i := 0; i < (60 * 60 * 3); i++ {
		complete, err := c.checkIfUploadComplete(uploadID)
		if complete {
			return err
		}
		if err != nil {
			return errors.Wrapf(err, "failed to check if upload %s is complete; will stop checking, but upload may complete anyway", uploadID)
		}
		logger.Infof("Waiting for upload to complete..")
		time.Sleep(time.Second)
	}

	return errors.Errorf("timed out waiting for upload %s to complete", uploadID)
}
