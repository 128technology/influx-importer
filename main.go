package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"strconv"

	"github.com/jeffail/tunny"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	build string
)

var (
	stdout = log.New(os.Stdout, "", 0)
	stderr = log.New(os.Stderr, "", 0)
)

var (
	token          = kingpin.Flag("token", "The API token.").Short('t').Required().OverrideDefaultFromEnvar("TOKEN").String()
	url            = kingpin.Flag("url", "The url URL.").Short('u').Required().String()
	workers        = kingpin.Flag("workers", "The number of works to process data.").Short('w').Default("2").Int()
	influxAddress  = kingpin.Flag("influx-address", "The HTTP address of the influx instance.").Required().String()
	influxUser     = kingpin.Flag("influx-user", "The username for the influx instance.").OverrideDefaultFromEnvar("INFLUX_USER").String()
	influxPass     = kingpin.Flag("influx-pass", "The password for the influx instance.").OverrideDefaultFromEnvar("INFLUX_PASS").String()
	influxDatabase = kingpin.Flag("influx-database", "The influx database to store analytics.").Required().String()
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func parametersToString(parameters []analyticParameter) string {
	var str []string
	for _, param := range parameters {
		str = append(str, fmt.Sprintf("%v=%v", param.Name, param.Value))
	}
	return strings.Join(str, ",")
}

func extract() error {
	client := createClient(*url, *token, httpClient)
	window := analyticWindow{End: "now", Start: "now-3600"}

	var wg sync.WaitGroup

	pool, err := tunny.CreatePoolGeneric(*workers).Open()
	if err != nil {
		return err
	}

	enqueueAnalytics := func(metrics []string, parameters []analyticParameter) {
		paramStr := parametersToString(parameters)

		for _, metric := range metrics {
			wg.Add(1)

			pool.SendWork(func() {
				defer wg.Done()

				points, err := client.getAnalytic(&analyticRequest{
					Metric:     metric,
					Transform:  "sum",
					Window:     window,
					Parameters: parameters,
				})

				if err != nil {
					stderr.Printf("HTTP request for %v(%v) failed return successfully: %v\n", metric, paramStr, err.Error())
					return
				}

				if err = sendAnalytis(metric, parameters, points); err != nil {
					stderr.Printf("Influx write for %v(%v) failed return successfully: %v\n", metric, paramStr, err.Error())
					return
				}

				stdout.Printf("Successfully exported %v(%v).", metric, paramStr)
			})
		}
	}

	config, err := client.getConfiguration()
	if err != nil {
		return fmt.Errorf("Unable to retrieve 128T configuration. %v", err.Error())
	}

	serviceGroups := map[string]string{}
	for _, service := range config.Authority.Services {
		if service.ServiceGroup != "" {
			serviceGroups[service.ServiceGroup] = service.ServiceGroup
		}
	}

	for _, router := range config.Authority.Routers {
		for _, node := range router.Nodes {
			enqueueAnalytics(nodeMetrics, []analyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "node", Value: node.Name},
			})

			for _, deviceInterface := range node.DeviceInterfaces {
				enqueueAnalytics(deviceInterfaceMetrics, []analyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "device_interface", Value: strconv.Itoa(deviceInterface.ID)},
				})

				for _, networkInterface := range deviceInterface.NetworkInterfaces {
					enqueueAnalytics(networkInterfaceMetrics, []analyticParameter{
						{Name: "router", Value: router.Name},
						{Name: "network_interface", Value: networkInterface.Name},
					})
				}
			}
		}

		for _, service := range config.Authority.Services {
			enqueueAnalytics(serviceMetrics, []analyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service", Value: service.Name},
			})
		}

		for _, tenant := range config.Authority.Tenants {
			enqueueAnalytics(tenantMetrics, []analyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service", Value: tenant.Name},
			})
		}

		for _, serviceClass := range config.Authority.ServiceClasses {
			enqueueAnalytics(serviceClassMetrics, []analyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service_class", Value: serviceClass.Name},
			})
		}

		for _, serviceRoute := range router.ServiceRoutes {
			enqueueAnalytics(serviceRouteMetrics, []analyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service_route", Value: serviceRoute.Name},
			})
		}

		for serviceGroup := range serviceGroups {
			enqueueAnalytics(serviceGroupMetrics, []analyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service_group", Value: serviceGroup},
			})
		}
	}

	wg.Wait()
	return nil
}

func main() {
	kingpin.Version(build)
	kingpin.Parse()

	if err := extract(); err != nil {
		panic(err)
	}
}
