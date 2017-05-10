package influx

import (
	"time"

	t128 "github.com/128technology/influx-importer/client"
	influx "github.com/influxdata/influxdb/client/v2"
)

// Client represents a connection to an InfluxDB instance
type Client struct {
	httpClient influx.Client
	database   string
}

// CreateClient creates an InfluxDB client
func CreateClient(address string, database string, username string, password string) (*Client, error) {
	config := influx.HTTPConfig{
		Addr:     address,
		Username: username,
		Password: password,
	}

	httpClient, err := influx.NewHTTPClient(config)
	if err != nil {
		return nil, err
	}

	client := &Client{
		httpClient: httpClient,
		database:   database,
	}

	return client, nil
}

// Send flushes a series of AnalyticPoints to InfluxDB
func (client Client) Send(metric string, parameters []t128.AnalyticParameter, points []t128.AnalyticPoint) error {
	config := influx.BatchPointsConfig{
		Database:  client.database,
		Precision: "ms",
	}

	bp, err := influx.NewBatchPoints(config)
	if err != nil {
		return err
	}

	for _, point := range points {
		timestamp, err := time.Parse(time.RFC3339, point.Time)
		if err != nil {
			return err
		}

		fields := map[string]interface{}{"value": point.Value}
		tags := map[string]string{}

		for _, param := range parameters {
			tags[param.Name] = param.Value
		}

		pt, err := influx.NewPoint(metric, tags, fields, timestamp)
		if err != nil {
			return err
		}

		bp.AddPoint(pt)
	}

	return client.httpClient.Write(bp)
}
