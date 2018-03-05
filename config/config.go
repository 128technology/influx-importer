package config

import (
	"fmt"

	"github.com/go-ini/ini"

	"github.com/128technology/influx-importer/client"
)

// InfluxConfig represents the influx porition of the config
type InfluxConfig struct {
	Address  string `ini:"address"`
	Username string `ini:"username"`
	Password string `ini:"password"`
	Database string `ini:"database"`
}

// ApplicationConfig represents the application porition of the config
type ApplicationConfig struct {
	MaxConcurrentRouters int `ini:"max-concurrent-routers"`
}

// TargetConfig represents the target porition of the config
type TargetConfig struct {
	URL   string `ini:"url"`
	Token string `ini:"token"`
}

// AlarmHistoryConfig represents the alarms portion of the config
type AlarmHistoryConfig struct {
	Enabled   bool `ini:"enabled"`
	QueryTime int  `ini:"max-query-time"`
}

// MetricsConfig represents the metric portion of the config
type MetricsConfig struct {
	QueryTime int `ini:"max-query-time"`
	Metrics   []string
}

// Config represents the application's configuration
type Config struct {
	Target       TargetConfig
	Application  ApplicationConfig
	Influx       InfluxConfig
	AlarmHistory AlarmHistoryConfig
	Metrics      MetricsConfig
}

// Load loads a configuration file and returns a configuration object
func Load(filename string) (*Config, error) {
	loadOpts := ini.LoadOptions{AllowBooleanKeys: true}

	ini, err := ini.LoadSources(loadOpts, filename)
	if err != nil {
		return nil, err
	}

	application, err := getApplicationConfig(ini)
	if err != nil {
		return nil, err
	}

	alarmHistory, err := getAlarmHistoryConfig(ini)
	if err != nil {
		return nil, err
	}

	metrics, err := getMetricsConfig(ini)
	if err != nil {
		return nil, err
	}

	target, err := getTargetConfig(ini)
	if err != nil {
		return nil, err
	}

	influx, err := getInfluxConfig(ini)
	if err != nil {
		return nil, err
	}

	return &Config{
		Application:  *application,
		Influx:       *influx,
		Metrics:      *metrics,
		Target:       *target,
		AlarmHistory: *alarmHistory,
	}, nil
}

func getApplicationConfig(ini *ini.File) (*ApplicationConfig, error) {
	applicationConfig := new(ApplicationConfig)
	err := ini.Section("application").MapTo(applicationConfig)
	if err != nil {
		return nil, err
	}

	if applicationConfig.MaxConcurrentRouters <= 0 {
		return nil, fmt.Errorf("Error: The maximum concurrent routers must be greater than 0")
	}

	return applicationConfig, nil
}

func getAlarmHistoryConfig(ini *ini.File) (*AlarmHistoryConfig, error) {
	config := new(AlarmHistoryConfig)
	err := ini.Section("alarm-history").MapTo(config)
	if err != nil {
		return nil, err
	}

	if config.QueryTime <= 0 {
		return nil, fmt.Errorf("alarms max-query-time must be greater than 0 seconds")
	}

	return config, nil
}

func getMetricsConfig(ini *ini.File) (*MetricsConfig, error) {
	metricsConfig := new(MetricsConfig)
	metricsSection := ini.Section("metrics")

	if err := metricsSection.MapTo(metricsConfig); err != nil {
		return nil, err
	}

	// For older versions of the config, thre may be a query-time in the applications
	// section that we'll use.
	if metricsConfig.QueryTime == 0 {
		applicationSection := ini.Section("application")
		if applicationSection.HasKey("query-time") {
			metricsConfig.QueryTime = applicationSection.Key("query-time").MustInt(0)
		}
	}

	if metricsConfig.QueryTime <= 0 {
		return nil, fmt.Errorf("metric max-query-time must be greater than 0 seconds")
	}

	metricKeys := metricsSection.Keys()
	metricsConfig.Metrics = make([]string, 0, len(metricKeys))
	for _, key := range metricKeys {
		// We must ignore the keys that are reflected upon to erroneously picking them up as
		// keys to metrics
		if key.Name() == "max-query-time" {
			continue
		}

		metricsConfig.Metrics = append(metricsConfig.Metrics, key.Name())
	}

	return metricsConfig, nil
}

func getTargetConfig(ini *ini.File) (*TargetConfig, error) {
	targetConfig := new(TargetConfig)
	err := ini.Section("target").MapTo(targetConfig)
	if err != nil {
		return nil, err
	}

	if len(targetConfig.URL) == 0 {
		return nil, fmt.Errorf("you must have a 128T URL set in the configuration file")
	}
	if len(targetConfig.Token) == 0 {
		return nil, fmt.Errorf("you must have a 128T token set in the configuration file")
	}

	return targetConfig, nil
}

func getInfluxConfig(ini *ini.File) (*InfluxConfig, error) {
	influxConfig := new(InfluxConfig)
	err := ini.Section("influx").MapTo(influxConfig)
	if err != nil {
		return nil, err
	}

	if len(influxConfig.Address) == 0 {
		return nil, fmt.Errorf("you must have a Influx address set in the configuration file")
	}
	if len(influxConfig.Database) == 0 {
		return nil, fmt.Errorf("you must have a Influx database set in the configuration file")
	}

	return influxConfig, nil
}

// PrintConfig prints the given metrics to the stdout
func PrintConfig(metrics []client.MetricDescriptor) {
	fmt.Println("[application]")
	fmt.Println("# The maximum number of routers to query at a given time.")
	fmt.Println("max-concurrent-routers=10")
	fmt.Println()
	fmt.Println("[target]")
	fmt.Println("# The fully qualified URL to the 128T Web Instance. E.g: https://10.0.1.29")
	fmt.Println("url=")
	fmt.Println("# The JWT token acquired when logging into the 128T application.")
	fmt.Println("token=")
	fmt.Println()
	fmt.Println("[influx]")
	fmt.Println("# The address of the Influx instance which is typically a HTTP address.")
	fmt.Println("address=")
	fmt.Println("username=")
	fmt.Println("password=")
	fmt.Println("database=")
	fmt.Println()
	fmt.Println("[alarm-history]")
	fmt.Println("enabled=true")
	fmt.Println("# The maximum time, in seconds, to go back and collect alarms for.")
	fmt.Println("max-query-time=3600")
	fmt.Println()
	fmt.Println("[metrics]")
	fmt.Println("# The maximum time, in seconds, to go back and collect metrics for.")
	fmt.Println("max-query-time=3600")
	fmt.Println()
	fmt.Println("# All metrics are, by default, disabled.")
	fmt.Println("# Uncomment the desired stat to begin pulling for it.")
	fmt.Println("# Keep in mind that the more stats you enable the longer query times take")
	fmt.Println("# and the more consistent burden you place on the 128T routers.")
	fmt.Println()

	for _, metric := range metrics {
		fmt.Printf("# %v: %v\n", metric.Label, metric.Description)
		fmt.Printf("#%v\n\n", metric.ID)
	}
}
