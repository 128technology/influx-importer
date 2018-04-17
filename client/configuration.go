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
	Routers []Router `json:"router"`
}

// Router represents a 128T Router
type Router struct {
	Name     string `json:"name"`
	Location string `json:"locationCoordinates"`
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
