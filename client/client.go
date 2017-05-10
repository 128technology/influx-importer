package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Client represents a HTTP connection to a 128T host.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// CreateClient creates a Client object given a baseURL, token, and httpClient.
func CreateClient(baseURL string, token string, httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		token:      token,
	}
}

func (client *Client) makeJSONRequest(url string, method string, requestBody interface{}, responseBody interface{}) (err error) {
	var body []byte

	if requestBody != nil {
		body, err = json.Marshal(requestBody)
		if err != nil {
			return
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bearer "+client.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = errors.New("Invalid status code: " + resp.Status)
		return
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(responseBody)
	return
}

// GetAnalytic retrieves an array of AnalyticPoint.
func (client *Client) GetAnalytic(request *AnalyticRequest) ([]AnalyticPoint, error) {
	url := fmt.Sprintf("%v/api/v1/analytics/runningTransform", client.baseURL)
	var response []AnalyticPoint
	err := client.makeJSONRequest(url, "POST", request, &response)
	return response, err
}

// GetConfiguration retrieves the configuration.
func (client *Client) GetConfiguration() (Configuration, error) {
	url := fmt.Sprintf("%v/api/v1/config/getJSON?source=running", client.baseURL)
	var response Configuration
	err := client.makeJSONRequest(url, "GET", nil, &response)
	return response, err
}
