package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type oneTwentyEightClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

type configuration struct {
	Authority authority `json:"authority"`
}

type authority struct {
	Routers        []router       `json:"router"`
	Services       []service      `json:"service"`
	Tenants        []tenant       `json:"tenant"`
	ServiceClasses []serviceClass `json:"serviceClass"`
}

type router struct {
	Name          string         `json:"name"`
	Nodes         []node         `json:"node"`
	ServiceRoutes []serviceRoute `json:"service_route"`
}

type node struct {
	Name             string            `json:"name"`
	DeviceInterfaces []deviceInterface `json:"deviceInterface"`
}

type serviceRoute struct {
	Name string `json:"name"`
}

type deviceInterface struct {
	ID                int                `json:"id"`
	NetworkInterfaces []networkInterface `json:"networkInterface"`
}

type networkInterface struct {
	Name string `json:"name"`
}

type tenant struct {
	Name string `json:"name"`
}

type service struct {
	Name         string `json:"name"`
	ServiceGroup string `json:"serviceGroup"`
}

type serviceClass struct {
	Name string `json:"name"`
}

type analyticPoint struct {
	Value float64 `json:"value"`
	Time  string  `json:"date"`
}

type analyticParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type analyticWindow struct {
	End   string `json:"end"`
	Start string `json:"start"`
}

type analyticRequest struct {
	Metric     string              `json:"metric"`
	Transform  string              `json:"transform"`
	Parameters []analyticParameter `json:"parameters"`
	Window     analyticWindow      `json:"window"`
}

func createClient(baseURL string, token string, httpClient *http.Client) *oneTwentyEightClient {
	return &oneTwentyEightClient{
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		token:      token,
	}
}

func (client *oneTwentyEightClient) makeJSONRequest(url string, method string, requestBody interface{}, responseBody interface{}) (err error) {
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

func (client *oneTwentyEightClient) getAnalytic(request *analyticRequest) ([]analyticPoint, error) {
	url := fmt.Sprintf("%v/api/v1/analytics/runningTransform", client.baseURL)
	var response []analyticPoint
	err := client.makeJSONRequest(url, "POST", request, &response)
	return response, err
}

func (client *oneTwentyEightClient) getConfiguration() (configuration, error) {
	url := fmt.Sprintf("%v/api/v1/config/getJSON?source=running", client.baseURL)
	var response configuration
	err := client.makeJSONRequest(url, "GET", nil, &response)
	return response, err
}
