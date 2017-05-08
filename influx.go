package main

import (
	"time"

	client "github.com/influxdata/influxdb/client/v2"
)

func sendAnalytis(metric string, parameters []analyticParameter, points []analyticPoint) error {
	config := client.HTTPConfig{
		Addr: *influxAddress,
	}

	if influxUser != nil {
		config.Username = *influxUser
	}

	if influxPass != nil {
		config.Password = *influxPass
	}

	c, err := client.NewHTTPClient(config)
	if err != nil {
		return err
	}

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  *influxDatabase,
		Precision: "ms",
	})
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

		pt, err := client.NewPoint(metric, tags, fields, timestamp)
		if err != nil {
			return err
		}

		bp.AddPoint(pt)
	}

	if err := c.Write(bp); err != nil {
		return err
	}

	return nil
}
