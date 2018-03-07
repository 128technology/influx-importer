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

// GetLegacyAlarmHistory retrieves the historical alarms for a router.
// This method is use on routers that are 3.1.X.
func (client *Client) GetLegacyAlarmHistory(router string, startTime time.Time, endTime time.Time) ([]AuditEvent, error) {
	values := make(url.Values)
	values.Add("router", router)
	values.Add("start", startTime.UTC().Format(time.RFC3339))
	values.Add("end", endTime.UTC().Format(time.RFC3339))

	url := fmt.Sprintf("%v/api/v1/audit/alarms?%v", client.baseURL, values.Encode())
	var response []struct {
		Node     string    `json:"node"`
		Time     time.Time `json:"time"`
		ID       string    `json:"id"`
		Message  string    `json:"message"`
		Category string    `json:"category"`
		Severity string    `json:"severity"`
		Process  string    `json:"process"`
		Source   string    `json:"source"`
		Event    string    `json:"event"`
	}

	err := client.makeJSONRequest(url, "GET", nil, &response)
	if err != nil {
		return nil, err
	}

	events := make([]AuditEvent, len(response))
	for idx, e := range response {
		events[idx] = AuditEvent{
			Type:      "alarm",
			Router:    router,
			Node:      e.Node,
			Timestamp: e.Time,
			Data: map[string]interface{}{
				"uuid":     e.ID,
				"process":  e.Process,
				"source":   e.Source,
				"category": e.Category,
				"severity": e.Severity,
				"type":     e.Event,
				"message":  e.Message,
			},
		}
	}

	return events, nil
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
		err = errors.New("request successful but token was empty")
		return
	}

	token = responseBody.Token
	return
}
