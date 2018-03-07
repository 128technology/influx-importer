package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
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

	extractCommand = app.Command("extract", "Extract metrics from a 128T instance and load them into Influx")
	configFile     = extractCommand.Flag("config", "The configuration filename.").Required().String()

	tokenCommand = app.Command("get-token", "Gets the JWT token for login.")
	tokenURL     = tokenCommand.Arg("url", "The URL to retrieve a token for.").Required().String()
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

func (e *extractor) extract() error {
	metrics := make([]t128.MetricDescriptor, len(e.config.Metrics.Metrics))

	for idx, metric := range e.config.Metrics.Metrics {
		descriptor := t128.FindMetricByID(metric)
		if descriptor != nil {
			metrics[idx] = *descriptor
		} else {
			return fmt.Errorf("%v is not a valid metric", metric)
		}
	}

	extractAndSend := func(routerName string, metrics []t128.MetricDescriptor, filter t128.AnalyticMetricFilter, tags map[string]string) {
		paramStr := filter.ToString()
		filters := []t128.AnalyticMetricFilter{filter}

		for _, metric := range metrics {
			window := t128.AnalyticWindow{End: "now"}

			lastRecordedTime, err := e.influxClient.LastRecordedTime(metric.ID, tags)
			if err != nil {
				logger.Log.Warn("requesting last recorded time for %v: %s. Defaulting to last %v seconds\n",
					metric.ID, err.Error(), e.config.Metrics.QueryTime)
				lastRecordedTime = &time.Time{}
			}

			endTime := int32(math.Min(float64(e.config.Metrics.QueryTime), time.Since(*lastRecordedTime).Seconds()))
			window.Start = fmt.Sprintf("now-%v", endTime)

			points, err := e.client.GetMetric(routerName, &t128.AnalyticMetricRequest{
				ID:        "/stats/" + metric.ID,
				Transform: "sum",
				Window:    window,
				Filters:   filters,
			})

			if err != nil {
				logger.Log.Error("HTTP request for %v(%v) failed: %v\n", metric.ID, paramStr, err.Error())
				continue
			}

			if err = e.influxClient.Send(metric.ID, tags, points); err != nil {
				logger.Log.Error("Influx write for %v(%v) failed: %v\n", metric.ID, paramStr, err.Error())
				continue
			}

			logger.Log.Info("Exported last %v seconds of %v(%v).", endTime, metric.ID, paramStr)
		}
	}

	systemInfo, err := e.client.GetSystemInfo()
	if err != nil {
		logger.Log.Fatal("Unable to retrieve system information: %v", err.Error())
	}

	config, err := e.client.GetConfiguration()
	if err != nil {
		logger.Log.Fatal("Unable to retrieve configuration: %v", err.Error())
	}

	serviceGroups := config.Authority.ServiceGroups()

	var wg sync.WaitGroup
	sem := semaphore.New(e.config.Application.MaxConcurrentRouters)

	for _, router := range config.Authority.Routers {
		wg.Add(1)

		go func(router t128.Router) {
			sem.Acquire()
			defer sem.Release()
			defer wg.Done()

			for _, node := range router.Nodes {
				nodeMetrics := getMetricsByType(metrics, "node")
				nodeFilter := t128.AnalyticMetricFilter{"node": node.Name}
				extractAndSend(router.Name, nodeMetrics, nodeFilter, map[string]string{
					"router": router.Name,
					"node":   node.Name,
				})

				for _, deviceInterface := range node.DeviceInterfaces {
					deviceInterfaceMetrics := getMetricsByType(metrics, "device-interface")
					deviceInterfaceFilter := t128.AnalyticMetricFilter{
						"device_interface": fmt.Sprintf("%v.%v", node.Name, deviceInterface.ID),
					}
					extractAndSend(router.Name, deviceInterfaceMetrics, deviceInterfaceFilter, map[string]string{
						"router":           router.Name,
						"node":             node.Name,
						"device_interface": strconv.Itoa(deviceInterface.ID),
					})

					for _, networkInterface := range deviceInterface.NetworkInterfaces {
						networkInterfaceMetrics := getMetricsByType(metrics, "network-interface")
						networkInterfaceFilter := t128.AnalyticMetricFilter{
							"network_interface": fmt.Sprintf("%v.%v", node.Name, networkInterface.Name),
						}
						extractAndSend(router.Name, networkInterfaceMetrics, networkInterfaceFilter, map[string]string{
							"router":            router.Name,
							"node":              node.Name,
							"network_interface": networkInterface.Name,
						})

						for _, adjacency := range networkInterface.Adjacencies {
							peerPathMetrics := getMetricsByType(metrics, "peer-path")
							peerPath := fmt.Sprintf("%v/%v/%v/%v/%v", adjacency.Peer, adjacency.IPAddress, node.Name, deviceInterface.ID, networkInterface.Vlan)
							peerPathFilter := t128.AnalyticMetricFilter{"peer_path": peerPath}
							extractAndSend(router.Name, peerPathMetrics, peerPathFilter, map[string]string{
								"router":    router.Name,
								"peer_path": peerPath,
							})
						}
					}
				}
			}

			for _, service := range config.Authority.Services {
				serviceMetrics := getMetricsByType(metrics, "service")
				serviceFilter := t128.AnalyticMetricFilter{"service": service.Name}
				extractAndSend(router.Name, serviceMetrics, serviceFilter, map[string]string{
					"router":  router.Name,
					"service": service.Name,
				})
			}

			for _, tenant := range config.Authority.Tenants {
				tenantMetrics := getMetricsByType(metrics, "tenant")
				tenantFilter := t128.AnalyticMetricFilter{"tenant": tenant.Name}
				extractAndSend(router.Name, tenantMetrics, tenantFilter, map[string]string{
					"router": router.Name,
					"tenant": tenant.Name,
				})
			}

			for _, serviceClass := range config.Authority.ServiceClasses {
				serviceClassMetrics := getMetricsByType(metrics, "service-class")
				serviceClassFilter := t128.AnalyticMetricFilter{"service_class": serviceClass.Name}
				extractAndSend(router.Name, serviceClassMetrics, serviceClassFilter, map[string]string{
					"router":        router.Name,
					"service_class": serviceClass.Name,
				})
			}

			for _, serviceRoute := range router.ServiceRoutes {
				serviceRouteMetrics := getMetricsByType(metrics, "service-route")
				serviceRouteFilter := t128.AnalyticMetricFilter{"service_route": serviceRoute.Name}
				extractAndSend(router.Name, serviceRouteMetrics, serviceRouteFilter, map[string]string{
					"router":        router.Name,
					"service_route": serviceRoute.Name,
				})
			}

			for _, serviceGroup := range serviceGroups {
				serviceGroupMetrics := getMetricsByType(metrics, "service-group")
				serviceGroupFilter := t128.AnalyticMetricFilter{"service_group": serviceGroup}
				extractAndSend(router.Name, serviceGroupMetrics, serviceGroupFilter, map[string]string{
					"router":        router.Name,
					"service_group": serviceGroup,
				})
			}

			if e.config.AlarmHistory.Enabled {
				if err := e.collectAlarmHistory(router, systemInfo.Version); err != nil {
					logger.Log.Error("Failed retriving alarm history for %v: %v\n", router.Name, err.Error())
				}
			}
		}(router)
	}

	wg.Wait()
	return nil
}

func getMetricsByType(metrics []t128.MetricDescriptor, metricType string) []t128.MetricDescriptor {
	var returnMetrics []t128.MetricDescriptor
	for _, metric := range metrics {
		for _, key := range metric.Keys {
			if key == metricType {
				returnMetrics = append(returnMetrics, metric)
				break
			}
		}
	}

	return returnMetrics
}

func (e *extractor) collectAlarmHistory(router t128.Router, routerVersion string) error {
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

	var events []t128.AuditEvent
	if strings.HasPrefix(routerVersion, "3.1.") {
		events, err = e.client.GetLegacyAlarmHistory(router.Name, startTime, time.Now())
	} else {
		events, err = e.client.GetAuditEvents(router.Name, []string{"ALARM"}, startTime, time.Now())
	}
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

func getToken() error {
	reader := bufio.NewReader(os.Stdin)

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
	token, err := t128.GetToken(*tokenURL, strings.TrimSpace(user), string(pass))
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", *token)
	return nil
}

func main() {
	kingpin.Version(build)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case initCommand.FullCommand():
		config.PrintConfig(t128.Metrics)
	case tokenCommand.FullCommand():
		if err := getToken(); err != nil {
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
