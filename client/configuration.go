package client

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
	Name string `json:"name"`
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
