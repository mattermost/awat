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

func (c *Client) CreateTranslation(translationRequest *TranslationRequest) (*TranslationStatus, error) {
	resp, err := c.doPost(c.buildURL("/translate"), translationRequest)
	if err != nil {
		return nil, err
	}

	return NewTranslationStatusFromReader(resp.Body)
}

func (c *Client) GetTranslationStatus(translationId string) (*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/translation/%s", translationId))
	if err != nil {
		return nil, err
	}

	return NewTranslationStatusFromReader(resp.Body)
}

func (c *Client) GetTranslationStatusesByInstallation(installationId string) ([]*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/installation/%s", installationId))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	return NewTranslationStatusListFromReader(resp.Body)
}

func (c *Client) GetAllTranslations() ([]*TranslationStatus, error) {
	resp, err := c.doGet(c.buildURL("/translations"))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	return NewTranslationStatusListFromReader(resp.Body)
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
