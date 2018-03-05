package influx

import (
	"fmt"
	"strings"
	"time"

	t128 "github.com/128technology/influx-importer/client"
	influx "github.com/influxdata/influxdb/client/v2"
)

// Client represents a connection to an InfluxDB instance
type Client struct {
	httpClient influx.Client
	database   string
}

// Record represents an influx data point
type Record struct {
	Tags   map[string]string
	Fields map[string]interface{}
	Time   time.Time
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
func (client Client) Send(metric string, tags map[string]string, points []t128.AnalyticPoint) error {
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

		pt, err := influx.NewPoint(metric, tags, fields, timestamp)
		if err != nil {
			return err
		}

		bp.AddPoint(pt)
	}

	return client.httpClient.Write(bp)
}

// Insert adds multiple records to a series in a batch
func (client Client) Insert(series string, records []Record) error {
	config := influx.BatchPointsConfig{
		Database:  client.database,
		Precision: "ms",
	}

	bp, err := influx.NewBatchPoints(config)
	if err != nil {
		return err
	}

	for _, r := range records {
		pt, err := influx.NewPoint(series, r.Tags, r.Fields, r.Time)
		if err != nil {
			return err
		}

		bp.AddPoint(pt)
	}

	return client.httpClient.Write(bp)
}

// LastRecordedTime retrieves the last time a record was added for a metric
func (client Client) LastRecordedTime(metric string, tags map[string]string) (*time.Time, error) {
	whereClauses := make([]string, 0, len(tags))

	for k, v := range tags {
		where := fmt.Sprintf("\"%v\" = '%v'", k, v)
		whereClauses = append(whereClauses, where)
	}

	var whereClause string
	if len(whereClauses) > 0 {
		whereClause = "where " + strings.Join(whereClauses, " and ")
	}

	query := fmt.Sprintf("SELECT time, value from \"%v\" %v order by time desc limit 1", metric, whereClause)

	res, err := client.httpClient.Query(influx.Query{
		Database: client.database,
		Command:  query,
	})

	if err != nil {
		return nil, err
	}

	if len(res.Results) == 0 || len(res.Results[0].Series) == 0 || len(res.Results[0].Series[0].Values) == 0 {
		return nil, fmt.Errorf("previous recorded time does not exist for %v", metric)
	}

	row := res.Results[0].Series[0].Values[0]
	t, err := time.Parse(time.RFC3339, row[0].(string))
	return &t, err
}
