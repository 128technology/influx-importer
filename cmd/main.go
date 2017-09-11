package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/abiosoft/semaphore"

	t128 "github.com/128technology/influx-importer/client"
	"github.com/128technology/influx-importer/config"
	"github.com/128technology/influx-importer/influx"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var build = "development"

var (
	stdout = log.New(os.Stdout, "", 0)
	stderr = log.New(os.Stderr, "", 0)
)

var (
	app = kingpin.New("influx-importer", "An application for extracting 128T metrics and loading them into Influx")

	initCommand = app.Command("init", "Initialize the app by outputting a settings file.")

	extractCommand = app.Command("extract", "Extract metrics from a 128T instance and load them into Influx")
	configFile     = extractCommand.Flag("config", "The configuration filename.").Required().String()
)

func extract() error {
	cfg, err := config.Load(*configFile)
	if err != nil {
		return err
	}

	client := t128.CreateClient(cfg.Target.URL, cfg.Target.Token)
	metrics := make([]t128.MetricDescriptor, len(cfg.Metrics))
	window := t128.AnalyticWindow{
		End:   "now",
		Start: fmt.Sprintf("now-%v", cfg.Application.QueryTime),
	}

	for idx, metric := range cfg.Metrics {
		descriptor := t128.FindMetricByID(metric)
		if descriptor != nil {
			metrics[idx] = *descriptor
		} else {
			return fmt.Errorf("%v is not a valid metric", metric)
		}
	}

	influxClient, err := influx.CreateClient(cfg.Influx.Address, cfg.Influx.Database, cfg.Influx.Username, cfg.Influx.Password)
	if err != nil {
		return err
	}

	extractAndSend := func(routerName string, metrics []t128.MetricDescriptor, filter t128.AnalyticMetricFilter, tags map[string]string) {
		paramStr := filter.ToString()
		filters := []t128.AnalyticMetricFilter{filter}

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

			if err = influxClient.Send(metric.ID, tags, points); err != nil {
				stderr.Printf("Influx write for %v(%v) failed return successfully: %v\n", metric.ID, paramStr, err.Error())
				continue
			}

			stdout.Printf("Successfully exported %v(%v).", metric.ID, paramStr)
		}
	}

	config, err := client.GetConfiguration()
	if err != nil {
		panic(fmt.Errorf("Unable to retrieve 128T configuration. %v", err.Error()))
	}

	serviceGroups := map[string]string{}
	for _, service := range config.Authority.Services {
		if service.ServiceGroup != "" {
			serviceGroups[service.ServiceGroup] = service.ServiceGroup
		}
	}

	var wg sync.WaitGroup
	sem := semaphore.New(cfg.Application.MaxConcurrentRouters)

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

			for serviceGroup := range serviceGroups {
				serviceGroupMetrics := getMetricsByType(metrics, "service-group")
				serviceGroupFilter := t128.AnalyticMetricFilter{"service_group": serviceGroup}
				extractAndSend(router.Name, serviceGroupMetrics, serviceGroupFilter, map[string]string{
					"router":        router.Name,
					"service_group": serviceGroup,
				})
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

func main() {
	kingpin.Version(build)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case initCommand.FullCommand():
		config.PrintConfig(t128.Metrics)
	case extractCommand.FullCommand():
		if err := extract(); err != nil {
			panic(err)
		}
	}
}
