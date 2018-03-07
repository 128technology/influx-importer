package client

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/mmcloughlin/geohash"
)

// Configuration represents the root container for the 128T configuration hierarchy
type Configuration struct {
	Authority Authority `json:"authority"`
}

// Authority represents an 128T Authority
type Authority struct {
	Routers        []Router       `json:"router"`
	Services       []Service      `json:"service"`
	Tenants        []Tenant       `json:"tenant"`
	ServiceClasses []ServiceClass `json:"serviceClass"`
}

// Router represents a 128T Router
type Router struct {
	Name          string         `json:"name"`
	Nodes         []Node         `json:"node"`
	Location      string         `json:"locationCoordinates"`
	ServiceRoutes []ServiceRoute `json:"service_route"`
}

// Node represents a 128T Node
type Node struct {
	Name             string            `json:"name"`
	DeviceInterfaces []DeviceInterface `json:"deviceInterface"`
}

// ServiceRoute represents a 128T ServiceRoute
type ServiceRoute struct {
	Name string `json:"name"`
}

// DeviceInterface represents a 128T DeviceInterface
type DeviceInterface struct {
	ID                int                `json:"id"`
	NetworkInterfaces []NetworkInterface `json:"networkInterface"`
}

// NetworkInterface represents a 128T NetworkInterface
type NetworkInterface struct {
	Name        string      `json:"name"`
	Vlan        int         `json:"vlan"`
	Adjacencies []Adjacency `json:"adjacency"`
}

// Adjacency represents a network adjacency between routers
type Adjacency struct {
	Peer      string `json:"peer"`
	IPAddress string `json:"ipAddress"`
}

// Tenant represents a 128T Tenant
type Tenant struct {
	Name string `json:"name"`
}

// Service represents a 128T Service
type Service struct {
	Name         string `json:"name"`
	ServiceGroup string `json:"serviceGroup"`
}

// ServiceClass represents a 128T ServiceClass
type ServiceClass struct {
	Name string `json:"name"`
}

// Alarm represents an alarm event object.
type Alarm map[string]interface{}

// AuditEvent represents an event object.
type AuditEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Router    string                 `json:"router"`
	Node      string                 `json:"node"`
	Data      map[string]interface{} `json:"data"`
}

// SystemInformation represents information about the connected 128T server.
type SystemInformation struct {
	Version string `json:"softwareVersion"`
}

// ServiceGroups retrieves a unique list of service groups
func (a Authority) ServiceGroups() []string {
	serviceGroups := map[string]string{}
	for _, service := range a.Services {
		if service.ServiceGroup != "" {
			serviceGroups[service.ServiceGroup] = service.ServiceGroup
		}
	}

	ret := make([]string, 0, len(serviceGroups))
	for k := range serviceGroups {
		ret = append(ret, k)
	}

	return ret
}

// LocationGeohash converts the ISO location field into a geohash
func (a Router) LocationGeohash() (string, error) {
	if a.Location == "" {
		return "", nil
	}

	ISOCoord := regexp.MustCompile(`(\+|-)\d+\.?\d*`)
	temp := ISOCoord.FindAllString(a.Location, 2)
	if len(temp) < 2 {
		return "", fmt.Errorf("location is in incorrect format")
	}

	lat, err := strconv.ParseFloat(temp[0], 64)
	if err != nil {
		return "", err
	}

	lon, err := strconv.ParseFloat(temp[1], 64)
	if err != nil {
		return "", err
	}

	return geohash.Encode(lat, lon), nil
}
