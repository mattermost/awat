// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// Client is the programmatic interface to the AWAT API.
type Client struct {
	address    string
	headers    map[string]string
	httpClient *http.Client
}

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
	switch resp.StatusCode {
	case http.StatusAccepted:
		return NewTranslationStatusFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetTranslationStatus returns the TranslationStatus struct returned
// from the API for the given Translation ID
func (c *Client) GetTranslationStatus(translationId string) (*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/translation/%s", translationId))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)
	switch resp.StatusCode {
	case http.StatusOK:
		return NewTranslationStatusFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetTranslationStatusesByInstallation returns all Translations that
// pertain to an Installation
func (c *Client) GetTranslationStatusesByInstallation(installationId string) ([]*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/installation/translation/%s", installationId))
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
	case http.StatusNotFound:
		return nil, nil

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

// CompleteImport marks an Import as finished, with or without an error
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

// ListImports returns all Imports on the AWAT
// TODO pagination
func (c *Client) ListImports() ([]*ImportStatus, error) {
	resp, err := c.doGet(c.buildURL("/imports"))
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

// closeBody ensures the Body of an http.Response is properly closed.
func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = ioutil.ReadAll(r.Body)
		_ = r.Body.Close()
	}
}

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
