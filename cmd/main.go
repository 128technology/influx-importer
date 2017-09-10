package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/abiosoft/semaphore"

	t128 "github.com/128technology/influx-importer/client"
	"github.com/128technology/influx-importer/influx"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var build = "development"

var (
	stdout = log.New(os.Stdout, "", 0)
	stderr = log.New(os.Stderr, "", 0)
)

var (
	token                = kingpin.Flag("token", "The API token.").Short('t').Required().OverrideDefaultFromEnvar("TOKEN").String()
	url                  = kingpin.Flag("url", "The url URL.").Short('u').Required().String()
	influxAddress        = kingpin.Flag("influx-address", "The HTTP address of the influx instance.").Required().String()
	influxUser           = kingpin.Flag("influx-user", "The username for the influx instance.").Default("").OverrideDefaultFromEnvar("INFLUX_USER").String()
	influxPass           = kingpin.Flag("influx-pass", "The password for the influx instance.").Default("").OverrideDefaultFromEnvar("INFLUX_PASS").String()
	influxDatabase       = kingpin.Flag("influx-database", "The influx database to store analytics.").Required().String()
	maxConcurrentRouters = kingpin.Flag("max-concurrent-routers", "The number of routers to query concurrently.").Short('r').Default("10").Int()
)

func parametersToString(parameters t128.AnalyticMetricFilter) string {
	var str []string
	for key, val := range parameters {
		str = append(str, fmt.Sprintf("%v=%v", key, val))
	}
	return strings.Join(str, ",")
}

func extract(metrics []t128.MetricDescriptor) error {
	client := t128.CreateClient(*url, *token)
	window := t128.AnalyticWindow{End: "now", Start: "now-3600"}

	influxClient, err := influx.CreateClient(*influxAddress, *influxDatabase, *influxUser, *influxPass)
	if err != nil {
		return err
	}

	extractAndSend := func(metrics []t128.MetricDescriptor, filter t128.AnalyticMetricFilter) {
		paramStr := parametersToString(filter)

		routerName, routerPresent := filter["router"]
		if !routerPresent {
			panic(fmt.Errorf("Router key not present in filter"))
		}

		newFilter := make(t128.AnalyticMetricFilter)
		for k, v := range filter {
			if k != "router" {
				newFilter[k] = v
			}
		}

		filters := []t128.AnalyticMetricFilter{newFilter}

		for _, metric := range metrics {
			points, err := client.GetMetric(routerName, &t128.AnalyticMetricRequest{
				ID:        "/stats/" + metric.ID,
				Transform: "sum",
				Window:    window,
				Filters:   filters,
			})

			if err != nil {
				stderr.Printf("HTTP request for %v(%v) failed return successfully: %v\n", metric.ID, paramStr, err.Error())
				continue
			}

			if err = influxClient.Send(metric.ID, filter, points); err != nil {
				stderr.Printf("Influx write for %v(%v) failed return successfully: %v\n", metric.ID, paramStr, err.Error())
				continue
			}

			stdout.Printf("Successfully exported %v(%v).", metric.ID, paramStr)
		}
	}

	config, err := client.GetConfiguration()
	if err != nil {
		return fmt.Errorf("Unable to retrieve 128T configuration. %v", err.Error())
	}

	serviceGroups := map[string]string{}
	for _, service := range config.Authority.Services {
		if service.ServiceGroup != "" {
			serviceGroups[service.ServiceGroup] = service.ServiceGroup
		}
	}

	var wg sync.WaitGroup
	sem := semaphore.New(*maxConcurrentRouters)

	for _, router := range config.Authority.Routers {
		wg.Add(1)

		go func(router t128.Router) {
			sem.Acquire()
			defer sem.Release()
			defer wg.Done()

			for _, node := range router.Nodes {
				nodeMetrics := getMetricsByType(metrics, "node")
				extractAndSend(nodeMetrics, t128.AnalyticMetricFilter{
					"router": router.Name,
					"node":   node.Name,
				})

				for _, deviceInterface := range node.DeviceInterfaces {
					deviceInterfaceMetrics := getMetricsByType(metrics, "device-interface")
					extractAndSend(deviceInterfaceMetrics, t128.AnalyticMetricFilter{
						"router":           router.Name,
						"node":             node.Name,
						"device_interface": strconv.Itoa(deviceInterface.ID),
					})

					for _, networkInterface := range deviceInterface.NetworkInterfaces {
						deviceInterfaceMetrics := getMetricsByType(metrics, "network-interface")
						extractAndSend(deviceInterfaceMetrics, t128.AnalyticMetricFilter{
							"router":            router.Name,
							"node":              node.Name,
							"network_interface": networkInterface.Name,
						})
					}
				}
			}

			for _, service := range config.Authority.Services {
				serviceMetrics := getMetricsByType(metrics, "service")
				extractAndSend(serviceMetrics, t128.AnalyticMetricFilter{
					"router":  router.Name,
					"service": service.Name,
				})
			}

			for _, tenant := range config.Authority.Tenants {
				tenantMetrics := getMetricsByType(metrics, "tenant")
				extractAndSend(tenantMetrics, t128.AnalyticMetricFilter{
					"router": router.Name,
					"tenant": tenant.Name,
				})
			}

			for _, serviceClass := range config.Authority.ServiceClasses {
				serviceClassMetrics := getMetricsByType(metrics, "service-class")
				extractAndSend(serviceClassMetrics, t128.AnalyticMetricFilter{
					"router":        router.Name,
					"service_class": serviceClass.Name,
				})
			}

			for _, serviceRoute := range router.ServiceRoutes {
				serviceRouteMetrics := getMetricsByType(metrics, "service-route")
				extractAndSend(serviceRouteMetrics, t128.AnalyticMetricFilter{
					"router":        router.Name,
					"service_route": serviceRoute.Name,
				})
			}

			for serviceGroup := range serviceGroups {
				serviceGroupMetrics := getMetricsByType(metrics, "service-group")
				extractAndSend(serviceGroupMetrics, t128.AnalyticMetricFilter{
					"router":        router.Name,
					"service_group": serviceGroup,
				})
			}
		}(router)
	}

	wg.Wait()
	return nil
}

func getMetrics() ([]t128.MetricDescriptor, error) {
	metricIDs := []string{
		"ssc/clients",
		"registered-services/events",
	}

	var metrics []t128.MetricDescriptor
	for _, ID := range metricIDs {
		descriptor := t128.FindMetricByID(ID)
		if descriptor != nil {
			metrics = append(metrics, *descriptor)
		} else {
			err := fmt.Errorf("%v is not a valid metric", ID)
			return []t128.MetricDescriptor{}, err
		}
	}

	return metrics, nil
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

func main() {
	kingpin.Version(build)
	kingpin.Parse()

	if *maxConcurrentRouters <= 0 {
		stderr.Println("Error: The maximum concurrent routers must be greater than 0")
		os.Exit(-1)
	}

	metrics, err := getMetrics()
	if err != nil {
		stderr.Printf("Error: %v\n", err.Error())
		os.Exit(-1)
	}
	if len(metrics) == 0 {
		stderr.Println("Error: You must have atleast one metric")
	}

	if err := extract(metrics); err != nil {
		panic(err)
	}
}
