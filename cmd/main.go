package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jeffail/tunny"

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
	token          = kingpin.Flag("token", "The API token.").Short('t').Required().OverrideDefaultFromEnvar("TOKEN").String()
	url            = kingpin.Flag("url", "The url URL.").Short('u').Required().String()
	workers        = kingpin.Flag("workers", "The number of works to process data.").Short('w').Default("2").Int()
	influxAddress  = kingpin.Flag("influx-address", "The HTTP address of the influx instance.").Required().String()
	influxUser     = kingpin.Flag("influx-user", "The username for the influx instance.").Default("").OverrideDefaultFromEnvar("INFLUX_USER").String()
	influxPass     = kingpin.Flag("influx-pass", "The password for the influx instance.").Default("").OverrideDefaultFromEnvar("INFLUX_PASS").String()
	influxDatabase = kingpin.Flag("influx-database", "The influx database to store analytics.").Required().String()
)

func parametersToString(parameters []t128.AnalyticParameter) string {
	var str []string
	for _, param := range parameters {
		str = append(str, fmt.Sprintf("%v=%v", param.Name, param.Value))
	}
	return strings.Join(str, ",")
}

func extract() error {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client := t128.CreateClient(*url, *token, httpClient)
	window := t128.AnalyticWindow{End: "now", Start: "now-3600"}

	influxClient, err := influx.CreateClient(*influxAddress, *influxDatabase, *influxUser, *influxPass)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	pool, err := tunny.CreatePoolGeneric(*workers).Open()
	if err != nil {
		return err
	}

	enqueueAnalytics := func(metrics []string, parameters []t128.AnalyticParameter) {
		paramStr := parametersToString(parameters)

		for _, metric := range metrics {
			wg.Add(1)

			pool.SendWork(func() {
				defer wg.Done()

				points, err := client.GetAnalytic(&t128.AnalyticRequest{
					Metric:     metric,
					Transform:  "sum",
					Window:     window,
					Parameters: parameters,
				})

				if err != nil {
					stderr.Printf("HTTP request for %v(%v) failed return successfully: %v\n", metric, paramStr, err.Error())
					return
				}

				if err = influxClient.Send(metric, parameters, points); err != nil {
					stderr.Printf("Influx write for %v(%v) failed return successfully: %v\n", metric, paramStr, err.Error())
					return
				}

				stdout.Printf("Successfully exported %v(%v).", metric, paramStr)
			})
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

	for _, router := range config.Authority.Routers {
		for _, node := range router.Nodes {
			enqueueAnalytics(t128.NodeMetrics, []t128.AnalyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "node", Value: node.Name},
			})

			for _, deviceInterface := range node.DeviceInterfaces {
				enqueueAnalytics(t128.DeviceInterfaceMetrics, []t128.AnalyticParameter{
					{Name: "router", Value: router.Name},
					{Name: "device_interface", Value: strconv.Itoa(deviceInterface.ID)},
				})

				for _, networkInterface := range deviceInterface.NetworkInterfaces {
					enqueueAnalytics(t128.NetworkInterfaceMetrics, []t128.AnalyticParameter{
						{Name: "router", Value: router.Name},
						{Name: "network_interface", Value: networkInterface.Name},
					})
				}
			}
		}

		for _, service := range config.Authority.Services {
			enqueueAnalytics(t128.ServiceMetrics, []t128.AnalyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service", Value: service.Name},
			})
		}

		for _, tenant := range config.Authority.Tenants {
			enqueueAnalytics(t128.TenantMetrics, []t128.AnalyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "tenant", Value: tenant.Name},
			})
		}

		for _, serviceClass := range config.Authority.ServiceClasses {
			enqueueAnalytics(t128.ServiceClassMetrics, []t128.AnalyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service_class", Value: serviceClass.Name},
			})
		}

		for _, serviceRoute := range router.ServiceRoutes {
			enqueueAnalytics(t128.ServiceRouteMetrics, []t128.AnalyticParameter{
				{Name: "router", Value: router.Name},
				{Name: "service_route", Value: serviceRoute.Name},
			})
		}

		for serviceGroup := range serviceGroups {
			enqueueAnalytics(t128.ServiceGroupMetrics, []t128.AnalyticParameter{
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
