package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/abiosoft/semaphore"
	"github.com/howeyc/gopass"

	t128 "github.com/128technology/influx-importer/client"
	"github.com/128technology/influx-importer/config"
	"github.com/128technology/influx-importer/influx"
	"github.com/128technology/influx-importer/logger"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var build = "development"

const alarmHistorySeriesName = "alarm-history"

var (
	app = kingpin.New("influx-importer", "An application for extracting 128T metrics and loading them into Influx")

	initCommand = app.Command("init", "Initialize the app by outputting a settings file.")
	initOutFile = initCommand.Flag("out", "The output configuration filename.").Required().String()

	extractCommand = app.Command("extract", "Extract metrics from a 128T instance and load them into Influx")
	configFile     = extractCommand.Flag("config", "The configuration filename.").Required().String()
)

type extractor struct {
	config       *config.Config
	client       *t128.Client
	influxClient *influx.Client
}

func createExtractor() (*extractor, error) {
	cfg, err := config.Load(*configFile)
	if err != nil {
		return nil, err
	}

	client := t128.CreateClient(cfg.Target.URL, cfg.Target.Token)

	influxClient, err := influx.CreateClient(cfg.Influx.Address, cfg.Influx.Database, cfg.Influx.Username, cfg.Influx.Password)
	if err != nil {
		return nil, err
	}

	return &extractor{
		client:       client,
		influxClient: influxClient,
		config:       cfg,
	}, nil
}

func (e *extractor) extractAndSend(routerName string, metricID string, filter t128.AnalyticMetricFilter) {
	paramStr := filter.ToString()

	window := t128.AnalyticWindow{End: "now"}

	lastRecordedTime, err := e.influxClient.LastRecordedTime(metricID, filter)
	if err != nil {
		logger.Log.Warn("requesting last recorded time for %v: %s. Defaulting to last %v seconds\n",
			metricID, err.Error(), e.config.Metrics.QueryTime)
		lastRecordedTime = &time.Time{}
	}

	endTime := int32(math.Min(float64(e.config.Metrics.QueryTime), time.Since(*lastRecordedTime).Seconds()))
	window.Start = fmt.Sprintf("now-%v", endTime)

	routerlessFilter := make(t128.AnalyticMetricFilter)
	for k := range filter {
		if k != "router" {
			routerlessFilter[k] = filter[k]
		}
	}

	points, err := e.client.GetMetric(routerName, &t128.AnalyticMetricRequest{
		ID:        "/stats/" + metricID,
		Transform: "sum",
		Window:    window,
		Filters:   []t128.AnalyticMetricFilter{routerlessFilter},
	})

	if err != nil {
		logger.Log.Error("HTTP request for %v(%v) failed: %v\n", metricID, paramStr, err.Error())
		return
	}

	if err = e.influxClient.Send(metricID, filter, points); err != nil {
		logger.Log.Error("Influx write for %v(%v) failed: %v\n", metricID, paramStr, err.Error())
		return
	}

	logger.Log.Info("Exported last %v seconds of %v(%v).", endTime, metricID, paramStr)
}

func (e *extractor) extract() error {
	config, err := e.client.GetConfiguration()
	if err != nil {
		return fmt.Errorf("unable to retrieve configuration: %v", err.Error())
	}

	metricDescriptors, err := e.client.GetMetricMetadata()
	if err != nil {
		return fmt.Errorf("unable to retrieve metric metadata: %v", err.Error())
	}

	descriptorMap := make(map[string]*t128.MetricDescriptor)
	for _, desc := range metricDescriptors {
		descriptorMap[desc.ID] = desc
	}

	var wg sync.WaitGroup
	sem := semaphore.New(e.config.Application.MaxConcurrentRouters)

	for _, router := range config.Authority.Routers {
		wg.Add(1)

		go func(router t128.Router) {
			sem.Acquire()
			defer sem.Release()
			defer wg.Done()

			for _, metricID := range e.config.Metrics.Metrics {
				descriptor, ok := descriptorMap[metricID]
				if !ok {
					logger.Log.Warn("%v is not a valid metric within the system. Skipping...", metricID)
					continue
				}

				permutations, err := e.client.GetMetricPermutations(router.Name, *descriptor)
				if err != nil {
					logger.Log.Error("Error retriving permutations for %v on router %v: %v\n", metricID, router.Name, err)
				}

				for _, permutation := range permutations {
					filter := make(t128.AnalyticMetricFilter)
					filter["router"] = router.Name

					for key := range permutation.Parameters {
						filter[key] = permutation.Parameters[key]
					}

					e.extractAndSend(router.Name, descriptor.ID, filter)
				}

			}

			if e.config.AlarmHistory.Enabled {
				if err := e.collectAlarmHistory(router); err != nil {
					logger.Log.Error("Failed retriving alarm history for %v: %v\n", router.Name, err.Error())
				}
			}
		}(router)
	}

	wg.Wait()
	return nil
}

func (e *extractor) collectAlarmHistory(router t128.Router) error {
	maxStartTime := time.Now().Add(-time.Duration(e.config.AlarmHistory.QueryTime) * time.Second)

	lastRecordedTime, err := e.influxClient.LastRecordedTime(alarmHistorySeriesName, map[string]string{
		"router": router.Name,
	})
	if err != nil {
		logger.Log.Warn("Unable to retrieve last recorded time for alarm-history: %v. Starting from %v\n",
			err.Error(), maxStartTime.Format(time.RFC3339))
		lastRecordedTime = &maxStartTime
	} else {
		if lastRecordedTime.Unix() < maxStartTime.Unix() {
			lastRecordedTime = &maxStartTime
		}
	}

	// We have to adjust the last recorded time by the smallest amount possible so that
	// we don't keep picking up the same event at that time over and over again
	startTime := lastRecordedTime.Add(1 * time.Second)
	timeDelta := time.Now().Sub(startTime).Seconds()

	events, err := e.client.GetAuditEvents(router.Name, []string{"ALARM"}, startTime, time.Now())
	if err != nil {
		return err
	}

	geohash, err := router.LocationGeohash()
	if err != nil {
		logger.Log.Warn(
			"Failed translating router %v's location %v to geo hash: %v",
			router.Name, router.Location, err.Error())
	}

	records := make([]influx.Record, len(events))
	for i, event := range events {
		// IMPORTANT: records that are time & tag matches will end up replacing previous items
		// within the influx database. This means that we need to differentiate items. To do
		// this we could add a tag but that would most likely prove hazardous as it would significatly
		// increase the cardinality of the index which influx doesn't like. So, we'll use the nanosecond
		// resolution of the time to encode the index of the event.
		timestamp := event.Timestamp.Add(time.Duration(i) * time.Nanosecond)

		record := influx.Record{
			Time:   timestamp,
			Fields: map[string]interface{}{},
			Tags: map[string]string{
				"router":  event.Router,
				"node":    event.Node,
				"geohash": geohash,
			},
		}

		for k, v := range event.Data {
			if v != nil {
				record.Fields[k] = v
			}
		}

		records[i] = record
	}

	recordCount := len(records)
	logger.Log.Info("Exported last %v seconds (%v items) of alarm history from %v\n",
		int(timeDelta), recordCount, router.Name)
	if recordCount != 0 {
		return e.influxClient.Insert(alarmHistorySeriesName, records)
	}

	return nil
}

func initConfig() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("128T Hostname: ")
	host, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	fmt.Printf("Username: ")
	user, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	fmt.Printf("Password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return err
	}

	fmt.Println("Retriving token...")
	url := fmt.Sprintf("https://%v", strings.TrimSpace(host))
	token, err := t128.GetToken(url, strings.TrimSpace(user), string(pass))
	if err != nil {
		return err
	}

	client := t128.CreateClient(url, *token)
	descriptors, err := client.GetMetricMetadata()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(*initOutFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	config.PrintConfig(url, *token, descriptors, f)

	fmt.Printf("Configuration successfully writen to \"%v\"\n", *initOutFile)
	fmt.Printf("Additional changes are required within the configuration file before you start the application.\n")
	return nil
}

func main() {
	kingpin.Version(build)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case initCommand.FullCommand():
		if err := initConfig(); err != nil {
			panic(err)
		}
	case extractCommand.FullCommand():
		ext, err := createExtractor()
		if err != nil {
			panic(err)
		}

		if err := ext.extract(); err != nil {
			panic(err)
		}
	}
}
