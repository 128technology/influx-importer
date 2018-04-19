package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

// GetMetric retrieves an array of AnalyticPoint for a given metric.
func (client *Client) GetMetric(router string, request *AnalyticMetricRequest) ([]AnalyticPoint, error) {
	url := fmt.Sprintf("%v/api/v1/router/%v/metrics", client.baseURL, router)
	var response []AnalyticPoint
	err := client.makeJSONRequest(url, "POST", request, &response)
	return response, err
}

// GetRouters retrieves a list of the routers
func (client *Client) GetRouters() ([]Router, error) {
	url := fmt.Sprintf("%v/api/v1/router", client.baseURL)
	var response []Router
	err := client.makeJSONRequest(url, "GET", nil, &response)
	return response, err
}

// GetSystemInfo retrieves the version of the given node.
func (client *Client) GetSystemInfo() (SystemInformation, error) {
	url := fmt.Sprintf("%v/api/v1/system", client.baseURL)
	var response SystemInformation
	err := client.makeJSONRequest(url, "GET", nil, &response)
	return response, err
}

// GetAlarms retrieves the currently active alarms for a given router
func (client *Client) GetAlarms(router string) ([]Alarm, error) {
	url := fmt.Sprintf("%v/api/v1/router/%v/alarms", client.baseURL, router)
	var response []Alarm
	err := client.makeJSONRequest(url, "GET", nil, &response)
	return response, err
}

// GetAuditEvents retrieves the historical audit events for a router
func (client *Client) GetAuditEvents(router string, filter []string, startTime time.Time, endTime time.Time) ([]AuditEvent, error) {
	values := make(url.Values)
	values.Add("router", router)
	values.Add("start", startTime.UTC().Format(time.RFC3339))
	values.Add("end", endTime.UTC().Format(time.RFC3339))

	for _, v := range filter {
		values.Add("filter", v)
	}

	url := fmt.Sprintf("%v/api/v1/audit?%v", client.baseURL, values.Encode())
	var response []AuditEvent
	err := client.makeJSONRequest(url, "GET", nil, &response)
	return response, err
}

// GetMetricMetadata asks the server for all available metric descriptors
func (client *Client) GetMetricMetadata() ([]*MetricDescriptor, error) {
	body := map[string]interface{}{
		"query": `
		{
			metrics {
				metadata {
					id
					description
					units
					arguments
				}
			}
		}
		`,
	}

	var response struct {
		Data struct {
			Metrics struct {
				Metadata []struct {
					ID          string   `json:"id"`
					Description string   `json:"description"`
					Arguments   []string `json:"arguments"`
				} `json:"metadata"`
			} `json:"metrics"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	var descriptors []*MetricDescriptor
	url := fmt.Sprintf("%v/api/v1/graphql", client.baseURL)
	err := client.makeJSONRequest(url, "GET", body, &response)
	if err != nil {
		return descriptors, err
	}

	if len(response.Errors) > 0 {
		return descriptors, fmt.Errorf("%v", response.Errors[0].Message)
	}

	descriptors = make([]*MetricDescriptor, 0, len(response.Data.Metrics.Metadata))

	for _, m := range response.Data.Metrics.Metadata {
		descriptors = append(descriptors, &MetricDescriptor{
			ID:          m.ID,
			Description: m.Description,
			Keys:        m.Arguments,
		})
	}

	return descriptors, nil
}

// GetMetricPermutations retrieves, for a given metric, all the parameter permutations available.
func (client *Client) GetMetricPermutations(router string, descriptor MetricDescriptor) ([]*MetricPermutation, error) {
	url := fmt.Sprintf("%v/api/v1/router/%v/stats/%v", client.baseURL, router, descriptor.ID)
	var permutations []*MetricPermutation

	var response []struct {
		ID           string `json:"id"`
		Permutations []struct {
			Parameters []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"parameters"`
		} `json:"permutations"`
	}

	type parameter struct {
		Name    string `json:"name"`
		Itemize bool   `json:"itemize"`
	}

	params := make([]parameter, 0, len(descriptor.Keys))
	for _, k := range descriptor.Keys {
		if k == "router" {
			continue
		}

		params = append(params, parameter{
			Name:    k,
			Itemize: true,
		})
	}

	body := struct {
		Parameters []parameter `json:"parameters"`
	}{
		Parameters: params,
	}

	err := client.makeJSONRequest(url, "POST", body, &response)
	if err != nil {
		return permutations, err
	}

	for _, resp := range response {
		for _, perm := range resp.Permutations {
			permutation := &MetricPermutation{Parameters: make(map[string]string)}

			for _, param := range perm.Parameters {
				permutation.Parameters[param.Name] = param.Value
			}

			permutations = append(permutations, permutation)
		}
	}

	return permutations, nil
}

// GetToken requests a JWT token from the server to be used in future requests
func GetToken(baseURL string, username string, password string) (token *string, err error) {
	requestBody := map[string]string{
		"username": username,
		"password": password,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%v/api/v1/login", baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := createHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("Invalid status code: " + resp.Status)
	}

	var responseBody struct {
		Token *string `json:"token"`
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&responseBody)
	if err != nil {
		return nil, err
	}

	if responseBody.Token == nil {
		return nil, errors.New("request successful but token was empty")
	}

	return responseBody.Token, nil
}
