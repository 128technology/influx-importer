package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client represents a HTTP connection to a 128T host.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// create a default HTTP client
func createHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

// CreateClient creates a Client object given a baseURL, token, and httpClient.
func CreateClient(baseURL string, token string) *Client {
	return &Client{
		httpClient: createHTTPClient(),
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

// GetMetric retrieves an array of AnalyticPoint for a given metric.
func (client *Client) GetMetric(router string, request *AnalyticMetricRequest) ([]AnalyticPoint, error) {
	url := fmt.Sprintf("%v/api/v1/router/%v/metrics", client.baseURL, router)
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

// GetNodeVersion retrieves the version of the given node.
func (client *Client) GetNodeVersion(router string, node string) (string, error) {
	url := fmt.Sprintf("%v/api/v1/router/%v/node/%v/version", client.baseURL, router, node)
	var response struct {
		Version string `json:"version"`
	}

	err := client.makeJSONRequest(url, "GET", nil, &response)
	if err != nil {
		return "", err
	}

	return response.Version, nil
}

// GetToken requests a JWT token from the server to be used in future requests
func GetToken(baseURL string, username string, password string) (token *string, err error) {
	requestBody := map[string]string{
		"username": username,
		"password": password,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return
	}

	url := fmt.Sprintf("%v/api/v1/login", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := createHTTPClient().Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = errors.New("Invalid status code: " + resp.Status)
		return
	}

	var responseBody struct {
		Token *string `json:"token"`
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&responseBody)
	if err != nil {
		return
	}

	if responseBody.Token == nil {
		err = errors.New("Request successful but token was empty!")
		return
	}

	token = responseBody.Token
	return
}
