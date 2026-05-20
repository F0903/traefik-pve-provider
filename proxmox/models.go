package proxmox

import "encoding/json"

type Node struct {
	Name   string `json:"node"`
	Status string `json:"status,omitempty"`
}

type Resource struct {
	VMID   int    `json:"vmid"`
	Node   string `json:"node,omitempty"`
	Type   string `json:"type,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Tags   string `json:"tags,omitempty"`
}

type GuestConfig struct {
	Name        string            `json:"name,omitempty"`
	Hostname    string            `json:"hostname,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        string            `json:"tags,omitempty"`
	IPConfigs   map[string]string `json:"-"`
}

type GuestAgentInterfaces struct {
	Result []NetworkInterface `json:"result"`
}

type NetworkInterface struct {
	Name            string      `json:"name,omitempty"`
	HardwareAddress string      `json:"hardware-address,omitempty"`
	HWAddr          string      `json:"hwaddr,omitempty"`
	Inet            string      `json:"inet,omitempty"`
	Inet6           string      `json:"inet6,omitempty"`
	IPAddresses     []IPAddress `json:"ip-addresses,omitempty"`
}

type IPAddress struct {
	Address string      `json:"ip-address,omitempty"`
	Type    string      `json:"ip-address-type,omitempty"`
	Prefix  json.Number `json:"prefix,omitempty"`
}
