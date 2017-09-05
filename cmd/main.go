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
	influx "github.com/128technology/influx-importer/influx"
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

func parametersToString(parameters []t128.AnalyticParameter) string {
	var str []string
	for _, param := range parameters {
		str = append(str, fmt.Sprintf("%v=%v", param.Name, param.Value))
	}
	return strings.Join(str, ",")
}

func extract() error {
	client := t128.CreateClient(*url, *token)
	window := t128.AnalyticWindow{End: "now", Start: "now-3600"}

	influxClient, err := influx.CreateClient(*influxAddress, *influxDatabase, *influxUser, *influxPass)
	if err != nil {
		return err
	}

	extractAndSend := func(metrics []string, parameters []t128.AnalyticParameter) {
		paramStr := parametersToString(parameters)

		for _, metric := range metrics {
			points, err := client.GetAnalytic(&t128.AnalyticRequest{
				Metric:     metric,
				Transform:  "sum",
				Window:     window,
				Parameters: parameters,
			})

			if err != nil {
				stderr.Printf("HTTP request for %v(%v) failed return successfully: %v\n", metric, paramStr, err.Error())
				continue
			}

			if err = influxClient.Send(metric, parameters, points); err != nil {
				stderr.Printf("Influx write for %v(%v) failed return successfully: %v\n", metric, paramStr, err.Error())
				continue
			}

			stdout.Printf("Successfully exported %v(%v).", metric, paramStr)
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
				extractAndSend(t128.NodeMetrics, []t128.AnalyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "node", Value: node.Name},
				})

				for _, deviceInterface := range node.DeviceInterfaces {
					extractAndSend(t128.DeviceInterfaceMetrics, []t128.AnalyticParameter{
						{Name: "router", Value: router.Name},
						{Name: "node", Value: node.Name},
						{Name: "device_interface", Value: strconv.Itoa(deviceInterface.ID)},
					})

					for _, networkInterface := range deviceInterface.NetworkInterfaces {
						extractAndSend(t128.NetworkInterfaceMetrics, []t128.AnalyticParameter{
							{Name: "router", Value: router.Name},
							{Name: "node", Value: node.Name},
							{Name: "network_interface", Value: networkInterface.Name},
						})
					}
				}
			}

			for _, service := range config.Authority.Services {
				extractAndSend(t128.ServiceMetrics, []t128.AnalyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "service", Value: service.Name},
				})
			}

			for _, tenant := range config.Authority.Tenants {
				extractAndSend(t128.TenantMetrics, []t128.AnalyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "tenant", Value: tenant.Name},
				})
			}

			for _, serviceClass := range config.Authority.ServiceClasses {
				extractAndSend(t128.ServiceClassMetrics, []t128.AnalyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "service_class", Value: serviceClass.Name},
				})
			}

			for _, serviceRoute := range router.ServiceRoutes {
				extractAndSend(t128.ServiceRouteMetrics, []t128.AnalyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "service_route", Value: serviceRoute.Name},
				})
			}

			for serviceGroup := range serviceGroups {
				extractAndSend(t128.ServiceGroupMetrics, []t128.AnalyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "service_group", Value: serviceGroup},
				})
			}
		}(router)
	}

	wg.Wait()
	return nil
}

func main() {
	kingpin.Version(build)
	kingpin.Parse()

	if *maxConcurrentRouters <= 0 {
		stderr.Println("Error: The maximum concurrent routers must be greater than 0")
		os.Exit(-1)
	}

	if err := extract(); err != nil {
		panic(err)
	}
}
